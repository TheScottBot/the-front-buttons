package editor

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// The editor listens on loopback, which every account on the machine can
// reach, so every page needs a session. A session is a cookie holding the
// editor's token, given by a one time login link: the deck's editor key opens a
// fresh link at each press, and the -open-editor command asks for one with the
// token it reads from the state folder, so the token never appears in a command
// line another account could see (security sweep).
const (
	// SessionCookieName is the cookie a session lives in.
	SessionCookieName = "front_buttons_session"
	loginPath         = "/session"
	loginCodePath     = "/session/code"
	staticPrefix      = "/static/"
	loginCodeBytes    = 16
	loginCodeLifetime = 2 * time.Minute
	// stateLifetime is how long a sign in elsewhere, such as Mural's, may
	// take before the editor no longer expects its return.
	stateLifetime  = 10 * time.Minute
	stateBytes     = 32
	bearerPrefix   = "Bearer "
	refusedSession = "session"
)

// noSessionMessage is what a page without a session says.
const noSessionMessage = "This editor needs a session. Open it with the deck's editor key, " +
	"or run: the-front-buttons -open-editor\n"

// Sessions mints the one time login codes that give a session, each usable
// once within loginCodeLifetime.
type Sessions struct {
	token string
	now   func() time.Time

	mutex sync.Mutex
	codes map[string]time.Time
	// states are the sign ins elsewhere the editor started and expects back,
	// each with when it stops expecting it.
	states map[string]time.Time
}

// NewSessions returns the sessions for the editor's token.
func NewSessions(token string) *Sessions {
	return NewSessionsAt(token, time.Now)
}

// NewSessionsAt returns sessions reading the time from now.
func NewSessionsAt(token string, now func() time.Time) *Sessions {
	return &Sessions{
		token: token, now: now, mutex: sync.Mutex{}, codes: map[string]time.Time{}, states: map[string]time.Time{},
	}
}

// LoginURL mints a one time link that gives a session on the editor at the
// address.
func (sessions *Sessions) LoginURL(address string) (string, error) {
	code, mintError := sessions.mint(sessions.codes, loginCodeBytes, loginCodeLifetime)
	if mintError != nil {
		return "", mintError
	}

	return "http://" + address + loginPath + "?code=" + code, nil
}

// NewState mints the state a sign in elsewhere carries there and back, such
// as Mural's, usable once within stateLifetime. Mural's return carries no
// session cookie, as the cookie is SameSite Strict, so the state alone shows
// the editor started it.
func (sessions *Sessions) NewState() (string, error) {
	return sessions.mint(sessions.states, stateBytes, stateLifetime)
}

// TakeState reports whether a state is one minted and not yet used or
// expired, and uses it.
func (sessions *Sessions) TakeState(state string) bool {
	return sessions.take(sessions.states, state)
}

// Redeem reports whether a code is one minted and not yet used or expired,
// and uses it.
func (sessions *Sessions) Redeem(code string) bool {
	return sessions.take(sessions.codes, code)
}

// mint makes a random one time value, kept until its lifetime runs out, and
// drops any kept past theirs.
func (sessions *Sessions) mint(kept map[string]time.Time, length int, lifetime time.Duration) (string, error) {
	valueBytes := make([]byte, length)

	_, readError := rand.Read(valueBytes)
	if readError != nil {
		return "", fmt.Errorf("minting a one time value: %w", readError)
	}

	value := hex.EncodeToString(valueBytes)

	sessions.mutex.Lock()
	defer sessions.mutex.Unlock()

	now := sessions.now()
	for known, expires := range kept {
		if now.After(expires) {
			delete(kept, known)
		}
	}

	kept[value] = now.Add(lifetime)

	return value, nil
}

// take reports whether a one time value is kept and not expired, and uses it.
func (sessions *Sessions) take(kept map[string]time.Time, value string) bool {
	sessions.mutex.Lock()
	defer sessions.mutex.Unlock()

	expires, known := kept[value]
	delete(kept, value)

	return known && !sessions.now().After(expires)
}

func (sessions *Sessions) isToken(candidate string) bool {
	return candidate != "" && subtle.ConstantTimeCompare([]byte(candidate), []byte(sessions.token)) == 1
}

// hasSession reports whether a request carries the session cookie.
func (sessions *Sessions) hasSession(request *http.Request) bool {
	cookie, cookieError := request.Cookie(SessionCookieName)

	return cookieError == nil && sessions.isToken(cookie.Value)
}

// hasBearer reports whether a request carries the token as a bearer token, as
// the -open-editor command sends it.
func (sessions *Sessions) hasBearer(request *http.Request) bool {
	bearer, isBearer := strings.CutPrefix(request.Header.Get("Authorization"), bearerPrefix)

	return isBearer && sessions.isToken(bearer)
}

// login redeems a login link: a good code sets the session cookie and goes to
// the pages.
func (sessions *Sessions) login(writer http.ResponseWriter, request *http.Request) {
	if !sessions.Redeem(request.URL.Query().Get("code")) {
		http.Error(writer, "This login link has been used or has expired. "+noSessionMessage, http.StatusForbidden)

		return
	}

	// Not Secure: the editor is plain http on loopback, where a Secure cookie
	// would never be sent back. HttpOnly keeps it from scripts, SameSite Strict
	// from other sites, and the browser lets it end with the session.
	http.SetCookie(writer, &http.Cookie{ //nolint:gosec // see above
		Name: SessionCookieName, Value: sessions.token, Quoted: false, Path: "/", Domain: "",
		Expires: time.Time{}, RawExpires: "", MaxAge: 0, Secure: false, HttpOnly: true,
		SameSite: http.SameSiteStrictMode, Partitioned: false, Raw: "", Unparsed: nil,
	})
	http.Redirect(writer, request, "/page?page=0", http.StatusSeeOther)
}

// loginCode answers the -open-editor command with a fresh login link.
func (sessions *Sessions) loginCode(address string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		link, linkError := sessions.LoginURL(address)
		if linkError != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = writer.Write([]byte(link))
	}
}
