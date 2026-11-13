package editor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/editor"
)

// The editor pages the session tests ask for, named once.
const (
	firstPagePath   = "/page?page=0"
	credentialsPath = "/credentials"
)

// rawGet makes a same origin GET with no session cookie.
func (harness *editorHarness) rawGet(
	t *testing.T,
	path string,
	adjust func(request *http.Request),
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+testAddress+path, http.NoBody)
	adjust(request)

	recorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(recorder, request)

	return recorder
}

// TestEditor_RefusesAPageWithoutASession: the editor listens on loopback, which
// every account on the machine can reach, so every page needs the session a
// login link gives; the bare address says how to get one. The editor's own
// script and styles hold nothing, so they need none.
func TestEditor_RefusesAPageWithoutASession(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, path := range []string{firstPagePath, credentialsPath, "/key?page=0&row=0&column=0", "/diagnostics"} {
		refused := harness.rawGet(t, path, func(*http.Request) {})
		if refused.Code != http.StatusForbidden || !strings.Contains(refused.Body.String(), "-open-editor") {
			t.Errorf("%s: want 403 saying how to open the editor, got %d", path, refused.Code)
		}

		wrong := harness.rawGet(t, path, func(request *http.Request) {
			withCookie(request, editor.SessionCookieName, "not-the-token")
		})
		if wrong.Code != http.StatusForbidden {
			t.Errorf("%s: want a wrong session refused, got %d", path, wrong.Code)
		}

		if got := harness.get(t, path); got.Code != http.StatusOK {
			t.Errorf("%s: want 200 with the session, got %d", path, got.Code)
		}
	}

	if got := harness.rawGet(t, "/static/editor.css", func(*http.Request) {}); got.Code != http.StatusOK {
		t.Errorf("want the stylesheet served without a session, got %d", got.Code)
	}
}

// TestEditor_LoginLinkGivesASessionOnce: the deck's editor key and the
// -open-editor command open a one time link; it sets the session cookie,
// HttpOnly and SameSite Strict, and works once, for two minutes.
func TestEditor_LoginLinkGivesASessionOnce(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	link, linkError := harness.sessions.LoginURL(testAddress)
	if linkError != nil || !strings.HasPrefix(link, "http://"+testAddress+"/session?code=") {
		t.Fatalf("want a login link on the editor's address, got %q (%v)", link, linkError)
	}

	path := strings.TrimPrefix(link, "http://"+testAddress)

	login := harness.rawGet(t, path, func(*http.Request) {})
	if login.Code != http.StatusSeeOther || login.Header().Get("Location") != firstPagePath {
		t.Fatalf("want a redirect to the pages, got %d %s", login.Code, login.Header().Get("Location"))
	}

	cookie := login.Header().Get("Set-Cookie")
	for _, want := range []string{editor.SessionCookieName + "=" + testToken, "HttpOnly", "SameSite=Strict", "Path=/"} {
		if !strings.Contains(cookie, want) {
			t.Errorf("want %s in the session cookie, got %q", want, cookie)
		}
	}

	if again := harness.rawGet(t, path, func(*http.Request) {}); again.Code != http.StatusForbidden {
		t.Errorf("want a used link refused, got %d", again.Code)
	}

	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	clocked := editor.NewSessionsAt(testToken, func() time.Time { return now })
	stale, _ := clocked.LoginURL(testAddress)
	now = now.Add(3 * time.Minute)

	if clocked.Redeem(strings.TrimPrefix(stale, "http://"+testAddress+"/session?code=")) {
		t.Error("want a link older than two minutes refused")
	}
}

// TestEditor_GivesALoginLinkForTheToken: the -open-editor command reads the
// token from the state folder and asks for a link, so the token itself never
// appears in a command line another account could see.
func TestEditor_GivesALoginLinkForTheToken(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	ask := func(bearer string) *httptest.ResponseRecorder {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+testAddress+"/session/code",
			http.NoBody)
		if bearer != "" {
			request.Header.Set("Authorization", "Bearer "+bearer)
		}

		recorder := httptest.NewRecorder()
		harness.handler.ServeHTTP(recorder, request)

		return recorder
	}

	given := ask(testToken)
	if given.Code != http.StatusOK || !strings.HasPrefix(given.Body.String(), "http://"+testAddress+"/session?code=") {
		t.Fatalf("want a login link for the token, got %d %q", given.Code, given.Body.String())
	}

	for _, bearer := range []string{"", "not-the-token"} {
		if refused := ask(bearer); refused.Code != http.StatusForbidden {
			t.Errorf("bearer %q: want refused, got %d", bearer, refused.Code)
		}
	}
}
