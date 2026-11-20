package muralapi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// TestClient_AuthorisationAddressAsksToReadAndWriteMurals: signing in starts
// at Mural's authorisation address, with the app's client ID, the editor's
// callback, a state to check on return, and the two scopes the keys need, to
// read a board's state and to change it; before a client ID is stored there
// is none.
func TestClient_AuthorisationAddressAsksToReadAndWriteMurals(t *testing.T) {
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
		query.Get("scope") != "murals:read murals:write" || query.Get("response_type") != "code" {
		t.Errorf("want Mural's authorisation address for reading and writing murals, got %s", address)
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

// timerMural answers the token and timer endpoints as Mural does, recording
// each timer request.
type timerMural struct {
	mutex    sync.Mutex
	requests []string
	statuses []int
	refreshs int
	answer   string
}

func (fake *timerMural) serve(writer http.ResponseWriter, request *http.Request) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	writer.Header().Set("Content-Type", "application/json")

	if request.URL.Path == tokenEndpoint {
		fake.refreshs++
		encoded, _ := json.Marshal(map[string]any{ //nolint:errchkjson // a test's own map of plain values
			accessField: fmt.Sprintf("access-%d", fake.refreshs), expiresField: 900,
		})
		_, _ = writer.Write(encoded)

		return
	}

	body, _ := io.ReadAll(request.Body)
	fake.requests = append(fake.requests,
		request.Method+" "+request.URL.Path+" "+request.Header.Get("Authorization")+" "+string(body))

	status := http.StatusOK
	if len(fake.statuses) > 0 {
		status, fake.statuses = fake.statuses[0], fake.statuses[1:]
	}

	writer.WriteHeader(status)

	switch status {
	case http.StatusNotFound:
		_, _ = writer.Write([]byte(`{"code":"MURAL_NOT_FOUND","message":"Mural not found"}`))
	case http.StatusForbidden:
		_, _ = writer.Write([]byte(`{"code":"INVALID_SCOPE","message":"no"}`))
	case http.StatusNoContent:
	default:
		_, _ = writer.Write([]byte(fake.answer))
	}
}

// newTimerFake starts a Mural answering timer requests with the statuses
// given in turn, and a signed in client against it.
func newTimerFake(t *testing.T, statuses ...int) (*timerMural, *muralapi.Client) {
	t.Helper()

	return newTimerFakeAnswering(t, `{"value":{"duration":300}}`, statuses...)
}

// newTimerFakeAnswering is newTimerFake whose successful answers are answer.
func newTimerFakeAnswering(t *testing.T, answer string, statuses ...int) (*timerMural, *muralapi.Client) {
	t.Helper()

	fake := &timerMural{mutex: sync.Mutex{}, requests: nil, statuses: statuses, refreshs: 0, answer: answer}
	server := httptest.NewServer(http.HandlerFunc(fake.serve))
	t.Cleanup(server.Close)

	store := testsupport.NewCredentialStoreTestDouble()
	setUp(t, store, murals.RefreshTokenCredential, refreshToken)

	return fake, muralapi.NewClientAt(server.Client(), store, server.URL, time.Now)
}

// TestClient_StartsAndAddsToAMuralsTimer: starting posts the duration in
// seconds and the sound to the mural's timer, and adding patches its delta in
// seconds, each with the access token.
func TestClient_StartsAndAddsToAMuralsTimer(t *testing.T) {
	t.Parallel()

	fake, client := newTimerFake(t, http.StatusOK, http.StatusCreated)

	startError := client.StartTimer(t.Context(), "team.1712345678901", 5*time.Minute, true)
	addError := client.AddTime(t.Context(), "team.1712345678901", 30*time.Second)

	if startError != nil || addError != nil {
		t.Fatalf("want both done, got %v and %v", startError, addError)
	}

	want := []string{
		`POST /api/public/v1/murals/team.1712345678901/timer/start Bearer access-1 {"duration":300,"soundEnabled":true}`,
		`PATCH /api/public/v1/murals/team.1712345678901/timer Bearer access-1 {"delta":30}`,
	}

	if fmt.Sprint(fake.requests) != fmt.Sprint(want) {
		t.Errorf("want %v, got %v", want, fake.requests)
	}
}

// TestClient_SignsInAgainWhenMuralRefusesTheToken: an access token Mural no
// longer takes is refreshed and the request sent once more.
func TestClient_SignsInAgainWhenMuralRefusesTheToken(t *testing.T) {
	t.Parallel()

	fake, client := newTimerFake(t, http.StatusUnauthorized, http.StatusOK)

	addError := client.AddTime(t.Context(), "team.1712345678901", time.Minute)
	if addError != nil || len(fake.requests) != 2 || fake.refreshs != 2 ||
		!strings.Contains(fake.requests[1], "Bearer access-2") {
		t.Errorf("want one retry with a fresh token, got %v (%v)", fake.requests, addError)
	}

	_, again := newTimerFake(t, http.StatusUnauthorized, http.StatusUnauthorized)

	failure := again.AddTime(t.Context(), "team.1712345678901", time.Minute)
	if !errors.Is(failure, murals.ErrNotConnected) {
		t.Errorf("refused twice: want not connected, got %v", failure)
	}
}

// TestClient_StopsAndPausesAMuralsTimer: stopping posts to the timer's end,
// and pausing and resuming patch whether it is paused, as Mural documents.
func TestClient_StopsAndPausesAMuralsTimer(t *testing.T) {
	t.Parallel()

	fake, client := newTimerFake(t, http.StatusNoContent, http.StatusCreated, http.StatusCreated)

	stopError := client.StopTimer(t.Context(), "team.1712345678901")
	pauseError := client.PauseTimer(t.Context(), "team.1712345678901", true)
	resumeError := client.PauseTimer(t.Context(), "team.1712345678901", false)

	if stopError != nil || pauseError != nil || resumeError != nil {
		t.Fatalf("want all done, got %v, %v and %v", stopError, pauseError, resumeError)
	}

	want := []string{
		`POST /api/public/v1/murals/team.1712345678901/timer/end Bearer access-1 `,
		`PATCH /api/public/v1/murals/team.1712345678901/timer Bearer access-1 {"paused":true}`,
		`PATCH /api/public/v1/murals/team.1712345678901/timer Bearer access-1 {"paused":false}`,
	}

	if fmt.Sprint(fake.requests) != fmt.Sprint(want) {
		t.Errorf("want %v, got %v", want, fake.requests)
	}
}

// TestClient_ReadsAMuralsTimer: reading gets the timer, and tells running,
// paused and finished apart by its remaining seconds and paused timestamp.
func TestClient_ReadsAMuralsTimer(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		answer string
		want   murals.TimerState
	}{
		{
			"running", `{"value":{"duration":300,"initialTimestamp":1,"now":2,"pausedTimestamp":null,` +
				`"remainingSeconds":125.4,"serverTimestamp":2,"soundEnabled":true}}`,
			murals.TimerState{Running: true, Paused: false, Remaining: 125 * time.Second},
		},
		{
			"paused", `{"value":{"duration":300,"initialTimestamp":1,"now":2,"pausedTimestamp":1655312720372,` +
				`"remainingSeconds":60,"serverTimestamp":2,"soundEnabled":true}}`,
			murals.TimerState{Running: true, Paused: true, Remaining: time.Minute},
		},
		{
			"finished", `{"value":{"duration":300,"initialTimestamp":1,"now":2,"pausedTimestamp":null,` +
				`"remainingSeconds":0,"serverTimestamp":2,"soundEnabled":true}}`,
			murals.TimerState{Running: false, Paused: false, Remaining: 0},
		},
		{"never started", `{"value":{}}`, murals.TimerState{Running: false, Paused: false, Remaining: 0}},
	} {
		fake, client := newTimerFakeAnswering(t, test.answer)

		state, readError := client.ReadTimer(t.Context(), "team.1712345678901")
		if readError != nil || state != test.want {
			t.Errorf("%s: want %+v, got %+v (%v)", test.name, test.want, state, readError)
		}

		if want := "GET /api/public/v1/murals/team.1712345678901/timer Bearer access-1 "; fmt.Sprint(fake.requests) !=
			fmt.Sprint([]string{want}) {
			t.Errorf("%s: want one get of the timer, got %v", test.name, fake.requests)
		}
	}

	_, garbled := newTimerFakeAnswering(t, `{"value":`)

	_, garbledError := garbled.ReadTimer(t.Context(), "team.1712345678901")
	if !errors.Is(garbledError, murals.ErrUnreachable) {
		t.Errorf("want an answer that is not a timer unreachable, got %v", garbledError)
	}
}

// TestClient_SaysWhyMuralRefusedTheTimer: no such mural is not found, a
// refusal is refused by Mural's code, a sign in without the scope a request
// needs must sign in again, and a mural ID that is not one is never sent.
func TestClient_SaysWhyMuralRefusedTheTimer(t *testing.T) {
	t.Parallel()

	_, client := newTimerFake(t, http.StatusNotFound, http.StatusConflict, http.StatusForbidden)

	missing := client.AddTime(t.Context(), "team.1712345678901", time.Minute)
	refused := client.AddTime(t.Context(), "team.1712345678901", time.Minute)
	unscoped := client.AddTime(t.Context(), "team.1712345678901", time.Minute)

	if !errors.Is(missing, murals.ErrNotFound) {
		t.Errorf("want not found, got %v", missing)
	}

	if !errors.Is(refused, murals.ErrRefused) || !strings.Contains(refused.Error(), "Conflict") {
		t.Errorf("want refused by its status, got %v", refused)
	}

	if !errors.Is(unscoped, murals.ErrNotConnected) || !strings.Contains(unscoped.Error(), "INVALID_SCOPE") {
		t.Errorf("want a sign in without the scope to sign in again, got %v", unscoped)
	}

	fake, unsent := newTimerFake(t)

	unsentFailure := unsent.AddTime(t.Context(), "../users", time.Minute)
	if !errors.Is(unsentFailure, murals.ErrNotFound) || len(fake.requests) != 0 {
		t.Errorf("want a mural ID that is not one never sent, got %v", unsentFailure)
	}
}
