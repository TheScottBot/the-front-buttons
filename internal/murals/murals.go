// Package murals is what the core knows of Mural, the online whiteboard: the
// credentials its sign in keeps, the one host they may be sent to, and the
// ways using it fails. The client that talks to Mural is an adapter,
// internal/muralapi, built only in the composition root.
package murals

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"time"
)

// Host is the one host Mural's credentials are ever sent to. Each is bound to
// it in the credential store, so a configuration naming one as another key's
// credential cannot send it anywhere else.
const Host = "app.mural.co"

// The credentials Mural's sign in keeps in the credential store, never in
// configuration: the app's client ID and secret, from Mural's developer
// settings, and the refresh token signing in gives.
const (
	ClientIDCredential     = "mural-client-id"
	ClientSecretCredential = "mural-client-secret"
	RefreshTokenCredential = "mural-refresh-token"
)

// The ways using Mural fails.
var (
	// ErrNotSetUp reports no client ID or secret stored yet.
	ErrNotSetUp = errors.New("mural is not set up")
	// ErrNotConnected reports no sign in, or one Mural no longer accepts.
	ErrNotConnected = errors.New("not connected to mural")
	// ErrNotFound reports a mural Mural does not know, or one the person
	// signed in cannot reach.
	ErrNotFound = errors.New("mural not found")
	// ErrRefused reports a request Mural refused, by the reason it gave.
	ErrRefused = errors.New("mural refused the request")
	// ErrUnreachable reports Mural not reached, or not answering as it
	// should.
	ErrUnreachable = errors.New("mural could not be reached")
)

// Timer is a mural's timer, through Mural's API, for the person signed in:
// starting it for a duration, with or without its sound at the end, adding
// time to it, stopping it, pausing or resuming it, and reading it.
type Timer interface {
	StartTimer(timerContext context.Context, muralID string, duration time.Duration, sound bool) error
	AddTime(timerContext context.Context, muralID string, extra time.Duration) error
	StopTimer(timerContext context.Context, muralID string) error
	PauseTimer(timerContext context.Context, muralID string, paused bool) error
	ReadTimer(timerContext context.Context, muralID string) (TimerState, error)
}

// TimerState is a mural's timer as it is now: running or not, paused or not,
// and the time it has left, to the second.
type TimerState struct {
	Running   bool
	Paused    bool
	Remaining time.Duration
}

// boardPath is a board's address's path, as its browser shows it:
// /t/<workspace>/m/<workspace>/<number>, then the board's own part.
var boardPath = regexp.MustCompile(
	`^/t/([a-z0-9][a-z0-9_-]{0,63})/m/([a-z0-9][a-z0-9_-]{0,63})/([0-9]{1,20})(/[^/]*)?/?$`)

// idPattern is a mural ID as Mural's API takes it: its workspace and its
// number, joined by a dot.
var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}\.[0-9]{1,20}$`)

// BoardID is the mural a board's address names, as Mural's API knows it: the
// workspace it belongs to and its number, joined by a dot.
func BoardID(link string) (string, bool) {
	parsed, parseError := url.Parse(link)
	if parseError != nil || parsed.Scheme != "https" || parsed.Host != Host || parsed.User != nil {
		return "", false
	}

	// The workspace after /m/ is the one the board belongs to, which its ID
	// names; the one after /t/ is the workspace it is opened in.
	parts := boardPath.FindStringSubmatch(parsed.Path)
	if parts == nil {
		return "", false
	}

	return parts[2] + "." + parts[3], true
}

// IsID reports whether a mural ID has the form BoardID gives.
func IsID(muralID string) bool {
	return idPattern.MatchString(muralID)
}
