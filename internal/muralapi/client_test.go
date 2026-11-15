package muralapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/muralapi"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The secrets the tests store and Mural's fake hands out; none may appear in
// an error.
const (
	clientID      = "client-1234"
	clientSecret  = "secret-5678"
	refreshToken  = "refresh-abcd"
	rotatedToken  = "refresh-efgh"
	accessToken   = "access-0001"
	redirectURI   = "http://127.0.0.1:48080/mural/callback"
	tokenEndpoint = "/api/public/v1/authorization/oauth2/token" //nolint:gosec // an address, not a credential
)

// The token endpoint's answer's names, as Mural writes them.
const (
	accessField  = "access_token"
	expiresField = "expires_in"
	refreshField = "refresh_token"
)

// fakeMural answers the token endpoint as Mural does, recording each form it
// was sent.
type fakeMural struct {
	mutex    sync.Mutex
	forms    []url.Values
	status   int
	response map[string]any
}

func (fake *fakeMural) serve(writer http.ResponseWriter, request *http.Request) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	if request.Method != http.MethodPost || request.URL.Path != tokenEndpoint ||
		request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		http.Error(writer, "unexpected", http.StatusTeapot)

		return
	}

	_ = request.ParseForm()
	fake.forms = append(fake.forms, request.PostForm)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(fake.status)
	encoded, _ := json.Marshal(fake.response) //nolint:errchkjson // a test's own map of plain values
	_, _ = writer.Write(encoded)
}

func (fake *fakeMural) sent() []url.Values {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return append([]url.Values(nil), fake.forms...)
}

// clock is a time the test moves on.
type clock struct {
	mutex sync.Mutex
	at    time.Time
}

func (moment *clock) now() time.Time {
	moment.mutex.Lock()
	defer moment.mutex.Unlock()

	return moment.at
}

func (moment *clock) advance(by time.Duration) {
	moment.mutex.Lock()
	defer moment.mutex.Unlock()

	moment.at = moment.at.Add(by)
}

// setUp stores the client ID and secret as the editor does, bound to Mural.
func setUp(t *testing.T, store platform.CredentialStore, extra ...string) {
	t.Helper()

	stored := map[string]string{murals.ClientIDCredential: clientID, murals.ClientSecretCredential: clientSecret}
	for index := 0; index+1 < len(extra); index += 2 {
		stored[extra[index]] = extra[index+1]
	}

	for name, secret := range stored {
		writeError := store.WriteCredential(name, secret)
		if writeError == nil {
			writeError = platform.BindCredentialHost(store, name, murals.Host)
		}

		if writeError != nil {
			t.Fatal(writeError)
		}
	}
}

// newFake starts a fake Mural answering with a status and body, and a client
// against it reading the time from a clock.
func newFake(
	t *testing.T, status int, response map[string]any,
) (*fakeMural, *muralapi.Client, *testsupport.CredentialStoreTestDouble, *clock) {
	t.Helper()

	fake := &fakeMural{mutex: sync.Mutex{}, forms: nil, status: status, response: response}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)

	store := testsupport.NewCredentialStoreTestDouble()
	moment := &clock{mutex: sync.Mutex{}, at: time.Date(2026, time.November, 15, 9, 0, 0, 0, time.UTC)}

	return fake, muralapi.NewClientAt(server.Client(), store, server.URL, moment.now), store, moment
}

// requireStored fails unless the refresh token stored is the one given.
func requireStored(t *testing.T, store platform.CredentialStore, want string) {
	t.Helper()

	if stored, _ := store.ReadCredential(murals.RefreshTokenCredential); stored != want {
		t.Errorf("want refresh token %q kept, got %q", want, stored)
	}
}

// requireForm fails unless exactly one form was sent, holding each value
// given.
func requireForm(t *testing.T, forms []url.Values, want map[string]string) {
	t.Helper()

	if len(forms) != 1 {
		t.Fatalf("want one request, got %d", len(forms))
	}

	for name, value := range want {
		if got := forms[0].Get(name); got != value {
			t.Errorf("%s: want %q, got %q", name, value, got)
		}
	}
}

// requireNoSecret fails if an error names any secret.
func requireNoSecret(t *testing.T, failure error) {
	t.Helper()

	for _, secret := range []string{clientID, clientSecret, refreshToken, rotatedToken, accessToken} {
		if failure != nil && strings.Contains(failure.Error(), secret) {
			t.Errorf("want no secret in %q", failure)
		}
	}
}

// TestClient_AuthorisationAddressAsksForTheTimerOnly: signing in starts at
// Mural's authorisation address, with the app's client ID, the editor's
// callback, a state to check on return, and the one scope the timer keys
// need; before a client ID is stored there is none.
func TestClient_AuthorisationAddressAsksForTheTimerOnly(t *testing.T) {
	t.Parallel()

	_, client, store, _ := newFake(t, http.StatusOK, nil)

	_, notSetUp := client.AuthorizationURL(redirectURI, "state-1")
	if !errors.Is(notSetUp, murals.ErrNotSetUp) {
		t.Errorf("before set up: want not set up, got %v", notSetUp)
	}

	setUp(t, store)

	address, addressError := client.AuthorizationURL(redirectURI, "state-1")
	if addressError != nil {
		t.Fatal(addressError)
	}

	parsed, _ := url.Parse(address)
	query := parsed.Query()

	if parsed.Path != "/api/public/v1/authorization/oauth2/" || query.Get("client_id") != clientID ||
		query.Get("redirect_uri") != redirectURI || query.Get("state") != "state-1" ||
		query.Get("scope") != "murals:write" || query.Get("response_type") != "code" {
		t.Errorf("want Mural's authorisation address for the timer, got %s", address)
	}
}

// TestClient_ConnectTradesTheCodeAndKeepsTheRefreshToken: Mural's code is
// traded at its token endpoint for tokens, sending the client ID and secret
// in the form as Mural documents; the refresh token is stored, bound to
// Mural, and the access token used without asking again.
func TestClient_ConnectTradesTheCodeAndKeepsTheRefreshToken(t *testing.T) {
	t.Parallel()

	fake, client, store, _ := newFake(t, http.StatusOK, map[string]any{
		accessField: accessToken, expiresField: 900, refreshField: refreshToken,
	})
	setUp(t, store)

	connectError := client.Connect(t.Context(), "code-1", redirectURI)
	if connectError != nil {
		t.Fatal(connectError)
	}

	requireForm(t, fake.sent(), map[string]string{
		"grant_type": "authorization_code", "code": "code-1", "redirect_uri": redirectURI,
		"client_id": clientID, "client_secret": clientSecret,
	})

	stored, _ := store.ReadCredential(murals.RefreshTokenCredential)
	host, _ := platform.CredentialHost(store, murals.RefreshTokenCredential)

	if stored != refreshToken || host != murals.Host {
		t.Errorf("want the refresh token stored for %s, got %q for %q", murals.Host, stored, host)
	}

	token, tokenError := client.AccessToken(t.Context())
	if tokenError != nil || token != accessToken || len(fake.sent()) != 1 {
		t.Errorf("want the access token kept, got %q (%v) after %d requests", token, tokenError, len(fake.sent()))
	}
}

// TestClient_RefreshesTheAccessTokenWhenItRunsOut: with no access token, or
// one about to run out, the refresh token buys another; a new refresh token,
// if Mural sends one, replaces the old, and one it leaves out keeps the old.
func TestClient_RefreshesTheAccessTokenWhenItRunsOut(t *testing.T) {
	t.Parallel()

	fake, client, store, moment := newFake(t, http.StatusOK, map[string]any{
		accessField: accessToken, expiresField: 900, refreshField: rotatedToken,
	})
	setUp(t, store, murals.RefreshTokenCredential, refreshToken)

	token, tokenError := client.AccessToken(t.Context())
	if tokenError != nil || token != accessToken {
		t.Fatalf("want an access token, got %q (%v)", token, tokenError)
	}

	requireForm(t, fake.sent(), map[string]string{
		"grant_type": refreshField, refreshField: refreshToken, "client_secret": clientSecret,
	})

	requireStored(t, store, rotatedToken)

	moment.advance(10 * time.Minute)

	_, _ = client.AccessToken(t.Context())

	if len(fake.sent()) != 1 {
		t.Error("ten minutes on: want the access token still used")
	}

	fake.mutex.Lock()
	fake.response = map[string]any{accessField: accessToken, expiresField: 900}
	fake.mutex.Unlock()
	moment.advance(5 * time.Minute)

	_, _ = client.AccessToken(t.Context())

	if len(fake.sent()) != 2 {
		t.Error("fifteen minutes on: want another refresh")
	}

	requireStored(t, store, rotatedToken)
}

// TestClient_SaysWhenItCannotSignIn: no refresh token, or one Mural refuses,
// is not connected; a code Mural refuses is refused by its reason; secrets
// bound to another host are never sent; and no error names a secret.
func TestClient_SaysWhenItCannotSignIn(t *testing.T) {
	t.Parallel()

	_, client, store, _ := newFake(t, http.StatusBadRequest, map[string]any{
		"error": "invalid_grant", "error_description": "The refresh token " + refreshToken + " is not valid",
	})
	setUp(t, store)

	_, missing := client.AccessToken(t.Context())
	if !errors.Is(missing, murals.ErrNotConnected) {
		t.Errorf("no refresh token: want not connected, got %v", missing)
	}

	setUp(t, store, murals.RefreshTokenCredential, refreshToken)

	_, refused := client.AccessToken(t.Context())
	if !errors.Is(refused, murals.ErrNotConnected) {
		t.Errorf("refresh refused: want not connected, got %v", refused)
	}

	connectError := client.Connect(t.Context(), "code-1", redirectURI)
	if !errors.Is(connectError, murals.ErrRefused) || !strings.Contains(connectError.Error(), "invalid_grant") {
		t.Errorf("code refused: want refused by its reason, got %v", connectError)
	}

	requireNoSecret(t, missing)
	requireNoSecret(t, refused)
	requireNoSecret(t, connectError)
}

// TestClient_SendsNothingForSecretsBoundElsewhere: a client secret bound to
// another host, as a GitHub token is, is never sent to Mural.
func TestClient_SendsNothingForSecretsBoundElsewhere(t *testing.T) {
	t.Parallel()

	fake, client, store, _ := newFake(t, http.StatusOK, map[string]any{
		accessField: accessToken, expiresField: 900, refreshField: refreshToken,
	})
	setUp(t, store)

	bindError := platform.BindCredentialHost(store, murals.ClientSecretCredential, platform.DefaultCredentialHost)
	if bindError != nil {
		t.Fatal(bindError)
	}

	connectError := client.Connect(t.Context(), "code-1", redirectURI)
	if !errors.Is(connectError, platform.ErrCredentialHost) || len(fake.sent()) != 0 {
		t.Errorf("want nothing sent and the host refused, got %v after %d requests", connectError, len(fake.sent()))
	}
}

// TestClient_RefusesAnAnswerWithoutTokens: an answer with no access token,
// or a sign in with no refresh token, is refused rather than kept.
func TestClient_RefusesAnAnswerWithoutTokens(t *testing.T) {
	t.Parallel()

	_, client, store, _ := newFake(t, http.StatusOK, map[string]any{accessField: accessToken, expiresField: 900})
	setUp(t, store)

	connectError := client.Connect(t.Context(), "code-1", redirectURI)
	if !errors.Is(connectError, murals.ErrUnreachable) {
		t.Errorf("no refresh token: want it refused, got %v", connectError)
	}

	_, readError := store.ReadCredential(murals.RefreshTokenCredential)
	if readError == nil {
		t.Error("want nothing stored")
	}
}
