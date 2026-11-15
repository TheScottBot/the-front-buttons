package editor_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The Mural sign in's routes and the secrets the tests set it up with.
const (
	muralSetUpPath    = "/credentials/mural"
	muralConnectPath  = "/mural/connect"
	muralCallbackPath = "/mural/callback"
	muralClientID     = "client-1234"
	muralSecret       = "secret-5678"
	muralCallback     = "http://" + testAddress + muralCallbackPath
	stateField        = "state"
)

// muralConnectorFake stands in for Mural's API: it sends the person to a
// pretend authorisation page, and records each code traded.
type muralConnectorFake struct {
	mutex     sync.Mutex
	codes     []string
	redirects []string
	failure   error
}

func (fake *muralConnectorFake) AuthorizationURL(redirectURI string, state string) (string, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	if fake.failure != nil {
		return "", fake.failure
	}

	fake.redirects = append(fake.redirects, redirectURI)

	return "https://app.mural.co/authorise?" + url.Values{stateField: {state}}.Encode(), nil
}

func (fake *muralConnectorFake) Connect(_ context.Context, code string, redirectURI string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	fake.codes = append(fake.codes, code)
	fake.redirects = append(fake.redirects, redirectURI)

	return fake.failure
}

// withoutSession asks for a page as Mural's redirect back arrives: no
// session cookie, which is SameSite Strict.
func withoutSession(*http.Request) {}

// connectState follows Connect to Mural and returns the state it sent.
func connectState(t *testing.T, harness *editorHarness) string {
	t.Helper()

	response := harness.get(t, muralConnectPath)

	location, _ := url.Parse(response.Header().Get("Location"))
	if response.Code != http.StatusSeeOther || location == nil || location.Host != murals.Host {
		t.Fatalf("want a redirect to Mural, got %d to %q", response.Code, response.Header().Get("Location"))
	}

	return location.Query().Get(stateField)
}

// requireStoredForMural fails unless each credential is stored as given,
// bound to Mural.
func requireStoredForMural(t *testing.T, harness *editorHarness, want map[string]string) {
	t.Helper()

	for name, secret := range want {
		stored, _ := harness.credentials.ReadCredential(name)
		host, _ := platform.CredentialHost(harness.credentials, name)

		if stored != secret || host != murals.Host {
			t.Errorf("%s: want it stored for %s, got %q for %q", name, murals.Host, stored, host)
		}
	}
}

// TestEditor_CredentialsSetUpMural: the credentials page has a Mural section
// saying the address to register as the redirect URL; its client ID and
// secret are stored, bound to Mural, and never shown again; until both are,
// there is nothing to connect.
func TestEditor_CredentialsSetUpMural(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	before := harness.get(t, credentialsPath).Body.String()
	for _, want := range []string{
		`<h2 id="mural-heading">Mural</h2>`,
		"Not set up",
		muralCallback,
		`<form method="post" action="/credentials/mural">`,
		`<input id="mural-client-secret" name="client_secret" type="password" required autocomplete="off"`,
	} {
		if !strings.Contains(before, want) {
			t.Errorf("before set up: want %s", want)
		}
	}

	if strings.Contains(before, `href="/mural/connect"`) {
		t.Error("before set up: want nothing to connect")
	}

	response := harness.post(t, muralSetUpPath,
		url.Values{"client_id": {muralClientID}, "client_secret": {muralSecret}})
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/credentials?saved=mural" {
		t.Fatalf("want a redirect after setting up, got %d to %q", response.Code, response.Header().Get("Location"))
	}

	requireStoredForMural(t, harness, map[string]string{
		murals.ClientIDCredential: muralClientID, murals.ClientSecretCredential: muralSecret,
	})

	after := harness.get(t, "/credentials?saved=mural").Body.String()
	for _, want := range []string{
		"Set up, not connected", `<a class="button" href="/mural/connect">Connect to Mural</a>`,
	} {
		if !strings.Contains(after, want) {
			t.Errorf("after set up: want %s", want)
		}
	}

	if strings.Contains(after, muralSecret) || strings.Contains(after, muralClientID) {
		t.Error("want the client ID and secret never shown")
	}
}

// TestEditor_RefusesAnIncompleteMuralSetUp: a blank client ID or secret is
// refused, and nothing stored.
func TestEditor_RefusesAnIncompleteMuralSetUp(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	response := harness.post(t, muralSetUpPath, url.Values{"client_id": {muralClientID}, "client_secret": {" "}})
	if response.Code != http.StatusUnprocessableEntity {
		t.Errorf("want a blank secret refused, got %d", response.Code)
	}

	_, readError := harness.credentials.ReadCredential(murals.ClientIDCredential)
	if readError == nil {
		t.Error("want nothing stored")
	}
}

// TestEditor_ConnectsToMuralThroughItsOwnPage: Connect to Mural goes to
// Mural's page with a state; Mural's redirect back, which carries no session,
// is taken once for that state, the code traded for the sign in with the
// editor's callback address, and the page it shows holds no editor token.
func TestEditor_ConnectsToMuralThroughItsOwnPage(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)
	state := connectState(t, harness)

	if len(state) < 32 {
		t.Errorf("want a long random state, got %q", state)
	}

	callback := muralCallbackPath + "?" + url.Values{"code": {"code-1"}, stateField: {state}}.Encode()

	response := harness.rawGet(t, callback, withoutSession)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Connected to Mural") {
		t.Fatalf("want the sign in taken, got %d", response.Code)
	}

	if strings.Contains(response.Body.String(), testToken) {
		t.Error("want no editor token on a page shown without a session")
	}

	harness.mural.mutex.Lock()
	codes, redirects := harness.mural.codes, harness.mural.redirects
	harness.mural.mutex.Unlock()

	if len(codes) != 1 || codes[0] != "code-1" || redirects[len(redirects)-1] != muralCallback ||
		redirects[0] != muralCallback {
		t.Errorf("want the code traded with the editor's callback address, got %v and %v", codes, redirects)
	}

	if again := harness.rawGet(t, callback, withoutSession); again.Code != http.StatusBadRequest {
		t.Errorf("want a state used once, got %d", again.Code)
	}
}

// TestEditor_RefusesAMuralCallbackItDidNotStart: a callback with a state the
// editor never gave is refused without trading its code; one where Mural
// says the person declined says so; and every other page still needs a
// session.
func TestEditor_RefusesAMuralCallbackItDidNotStart(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	forged := harness.rawGet(t, muralCallbackPath+"?code=code-1&state=made-up", withoutSession)
	if forged.Code != http.StatusBadRequest || strings.Contains(forged.Body.String(), testToken) {
		t.Errorf("want a made up state refused, got %d", forged.Code)
	}

	declined := harness.rawGet(t, muralCallbackPath+"?"+url.Values{
		"error": {"access_denied"}, stateField: {connectState(t, harness)},
	}.Encode(), withoutSession)
	if declined.Code != http.StatusOK ||
		!strings.Contains(declined.Body.String(), "Mural did not connect: access_denied") {
		t.Errorf("want the refusal said, got %d", declined.Code)
	}

	harness.mural.mutex.Lock()
	traded := len(harness.mural.codes)
	harness.mural.mutex.Unlock()

	if traded != 0 {
		t.Error("want no code traded")
	}

	if page := harness.rawGet(t, credentialsPath, withoutSession); page.Code != http.StatusForbidden {
		t.Errorf("want the credentials page to need a session, got %d", page.Code)
	}
}

// TestEditor_ShowsMuralConnected: with a sign in stored, the section says it
// is connected, and still offers to connect again.
func TestEditor_ShowsMuralConnected(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for name, secret := range map[string]string{
		murals.ClientIDCredential: muralClientID, murals.ClientSecretCredential: muralSecret,
		murals.RefreshTokenCredential: "refresh-abcd",
	} {
		_ = harness.credentials.WriteCredential(name, secret)
	}

	page := harness.get(t, credentialsPath).Body.String()
	if !strings.Contains(page, "Connected") || !strings.Contains(page, `href="/mural/connect"`) {
		t.Error("want the section to say it is connected and offer to connect again")
	}
}

// TestSessions_MuralStatesAreUsedOnceAndRunOut: a state is taken once, and
// not after ten minutes.
func TestSessions_MuralStatesAreUsedOnceAndRunOut(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	sessions := editor.NewSessionsAt(testToken, func() time.Time { return now })

	state, _ := sessions.NewState()
	if !sessions.TakeState(state) || sessions.TakeState(state) {
		t.Error("want a state taken once")
	}

	stale, _ := sessions.NewState()
	now = now.Add(11 * time.Minute)

	if sessions.TakeState(stale) || sessions.TakeState("") {
		t.Error("want a state older than ten minutes, or none, refused")
	}
}
