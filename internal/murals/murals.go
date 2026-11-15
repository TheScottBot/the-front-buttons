// Package murals is what the core knows of Mural, the online whiteboard: the
// credentials its sign in keeps, the one host they may be sent to, and the
// ways using it fails. The client that talks to Mural is an adapter,
// internal/muralapi, built only in the composition root.
package murals

import "errors"

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
	// ErrRefused reports a request Mural refused, by the reason it gave.
	ErrRefused = errors.New("mural refused the request")
	// ErrUnreachable reports Mural not reached, or not answering as it
	// should.
	ErrUnreachable = errors.New("mural could not be reached")
)
