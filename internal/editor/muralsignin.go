package editor

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Signing in to Mural, for the timer keys. The credentials page's Mural
// section stores the client ID and secret of the person's own Mural app,
// bound to Mural so they are sent nowhere else, and never shows them again.
// Connect to Mural mints a one time state and goes to Mural's own page, which
// asks the person to allow the app; Mural returns to the editor's callback
// with a code and that state. The return carries no session cookie, which is
// SameSite Strict, so the callback alone among the editor's pages needs no
// session: it takes the state once, within ten minutes, and only then trades
// the code for the sign in. The page it shows holds no token and no secret.

// The Mural sign in's paths and announcement.
const (
	muralCallbackPath = "/mural/callback"
	muralQuery        = "mural"
	muralMessage      = "Mural set up. Connect to Mural to sign in."
)

// errorCodePattern is the shape of an OAuth error code, such as access_denied;
// anything else in a callback is not repeated.
var errorCodePattern = regexp.MustCompile(`^[a-z_]{1,40}$`)

// MuralConnector signs in to Mural: where its sign in starts, and trading the
// code it returns.
type MuralConnector interface {
	AuthorizationURL(redirectURI string, state string) (string, error)
	Connect(connectContext context.Context, code string, redirectURI string) error
}

// muralView is the Mural section: whether it is set up and connected, and the
// address to register as the app's redirect URL.
type muralView struct {
	Status      string
	Connectable bool
	CallbackURL string
}

// muralCallbackView is the page Mural's return shows: what happened, and no
// token, as it is shown without a session.
type muralCallbackView struct {
	baseView

	Connected bool
	Reason    string
}

// muralCallbackURL is the editor's callback, which the person registers as
// their Mural app's redirect URL.
func (pages *pageHandlers) muralCallbackURL() string {
	return "http://" + pages.options.Address + muralCallbackPath
}

// muralView says how far Mural's sign in has got, from what the credential
// store holds.
func (pages *pageHandlers) muralView() muralView {
	stored := func(credentialName string) bool {
		_, readError := pages.options.Credentials.ReadCredential(credentialName)

		return readError == nil
	}

	view := muralView{Status: "Not set up", Connectable: false, CallbackURL: pages.muralCallbackURL()}

	switch {
	case !stored(murals.ClientIDCredential) || !stored(murals.ClientSecretCredential):
	case stored(murals.RefreshTokenCredential):
		view.Status, view.Connectable = "Connected", true
	default:
		view.Status, view.Connectable = "Set up, not connected", true
	}

	view.Connectable = view.Connectable && pages.options.Mural != nil

	return view
}

// setUpMural stores the client ID and secret, each bound to Mural.
func (pages *pageHandlers) setUpMural(writer http.ResponseWriter, request *http.Request) {
	clientID := strings.TrimSpace(request.PostForm.Get("client_id"))
	clientSecret := strings.TrimSpace(request.PostForm.Get("client_secret"))

	if clientID == "" || clientSecret == "" {
		pages.renderCredentials(writer, request, http.StatusUnprocessableEntity,
			errorPrefix+"Mural needs both the client ID and the client secret")

		return
	}

	for credentialName, secret := range map[string]string{
		murals.ClientIDCredential: clientID, murals.ClientSecretCredential: clientSecret,
	} {
		writeError := pages.options.Credentials.WriteCredential(credentialName, secret)
		if writeError == nil {
			writeError = platform.BindCredentialHost(pages.options.Credentials, credentialName, murals.Host)
		}

		if writeError != nil {
			pages.renderCredentials(writer, request, http.StatusUnprocessableEntity,
				errorPrefix+"Mural's client ID and secret could not be stored; each must be 1 to 2,048 characters")

			return
		}
	}

	http.Redirect(writer, request, "/credentials?saved="+muralQuery, http.StatusSeeOther)
}

// connectMural goes to Mural's sign in with a fresh state.
func (pages *pageHandlers) connectMural(writer http.ResponseWriter, request *http.Request) {
	if pages.options.Mural == nil {
		http.NotFound(writer, request)

		return
	}

	state, stateError := pages.options.Sessions.NewState()
	if stateError != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	address, addressError := pages.options.Mural.AuthorizationURL(pages.muralCallbackURL(), state)
	if addressError != nil {
		pages.renderCredentials(writer, request, http.StatusUnprocessableEntity,
			errorPrefix+"set up Mural with its client ID and secret before connecting")

		return
	}

	http.Redirect(writer, request, address, http.StatusSeeOther)
}

// muralCallback takes Mural's return: its state once, then its code, or the
// reason it gives for not signing in.
func (pages *pageHandlers) muralCallback(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	view := muralCallbackView{baseView: pages.base("Mural", request, nil), Connected: false, Reason: ""}
	view.Token = ""

	if pages.options.Mural == nil || !pages.options.Sessions.TakeState(query.Get("state")) {
		view.Reason = "this sign in was not started here, or has been used or has run out; " +
			"start it again from the credentials page"
		pages.render(writer, http.StatusBadRequest, "muralcallback.html", view)

		return
	}

	if reason := query.Get("error"); reason != "" {
		view.Reason = "unknown"
		if errorCodePattern.MatchString(reason) {
			view.Reason = reason
		}

		pages.render(writer, http.StatusOK, "muralcallback.html", view)

		return
	}

	connectError := pages.options.Mural.Connect(request.Context(), query.Get("code"), pages.muralCallbackURL())
	if connectError != nil {
		view.Reason = connectReason(connectError)
		pages.render(writer, http.StatusOK, "muralcallback.html", view)

		return
	}

	view.Connected = true
	pages.render(writer, http.StatusOK, "muralcallback.html", view)
}

// connectReason says why trading the code failed, in words that name no
// secret.
func connectReason(connectError error) string {
	switch {
	case errors.Is(connectError, murals.ErrNotSetUp):
		return "Mural is not set up; store its client ID and secret first"
	case errors.Is(connectError, murals.ErrRefused):
		return "Mural refused the sign in; check the client ID, secret and redirect URL"
	case errors.Is(connectError, platform.ErrCredentialHost):
		return "the client ID or secret is not stored for Mural; store them again"
	default:
		return "Mural could not be reached"
	}
}
