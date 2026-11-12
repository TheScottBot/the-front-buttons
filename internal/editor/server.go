package editor

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

//go:embed templates/*.html static/editor.css static/htmx.min.js
var embeddedFiles embed.FS

// loopbackHost binds the editor to this machine only (specification section 1:
// reaching the editor from another machine is out of scope).
const loopbackHost = "127.0.0.1"

// DefaultPort is the editor's port unless another program holds it. A new port
// at every launch kept the collision free start the TE9 decision wanted, but
// broke every bookmark, and on the real deck the author found the address
// changing all the time (evaluation, deck testing). The Host, Origin and token
// checks are what protect the editor, not an unpredictable port.
const DefaultPort = 48080

// tokenBytes is the size of the per launch token: 256 bits, far beyond guessing.
const tokenBytes = 32

// maximumRequestBytes bounds every request body before it is parsed (datum
// 0.10). The largest form, a key with every field filled, is a few kilobytes.
const maximumRequestBytes = 64 * 1024

// tokenFieldName is the form field every change carries its token in.
const tokenFieldName = "token"

// The reasons a request is refused, counted and logged by category.
const (
	refusedHost   = "host"
	refusedOrigin = "origin"
	refusedToken  = "token"
	refusedSize   = "size"
)

// ConfigurationStore is what the editor needs from the configuration store.
type ConfigurationStore interface {
	OnDisk() (config.Configuration, error)
	Current() (config.Configuration, bool)
	Save(configuration config.Configuration) error
	LastRejection() error
}

// IconFileLoader reads a key's own icon file by its name, for the preview.
type IconFileLoader interface {
	Load(fileName string) (image.Image, error)
}

// IconCatalogue lists the bundled icons' names, for the editor's icon gallery.
type IconCatalogue interface {
	Names() []string
}

// Options are the editor's collaborators, supplied by the composition root.
type Options struct {
	Address     string
	Token       string
	Store       ConfigurationStore
	Registry    *brick.Registry
	Presets     presets.Table
	Grid        layout.Grid
	Renderer    *render.Renderer
	Icons       IconCatalogue
	IconFiles   IconFileLoader
	Credentials platform.CredentialStore
	Counters    *diagnostics.Counters
	Logger      *slog.Logger
}

// Listen opens the editor's loopback listener on the preferred port, or, if
// that port is taken or none is preferred (zero), on a port the operating
// system chooses. It reports whether it had to fall back, so the caller can say
// why the address is not the usual one.
func Listen(listenContext context.Context, preferredPort int) (net.Listener, bool, error) {
	var listenConfig net.ListenConfig

	if preferredPort > 0 {
		listener, preferredError := listenConfig.Listen(listenContext, "tcp",
			net.JoinHostPort(loopbackHost, strconv.Itoa(preferredPort)))
		if preferredError == nil {
			return listener, false, nil
		}
	}

	listener, listenError := listenConfig.Listen(listenContext, "tcp", net.JoinHostPort(loopbackHost, "0"))
	if listenError != nil {
		return nil, false, fmt.Errorf("opening the editor's loopback port: %w", listenError)
	}

	return listener, preferredPort > 0, nil
}

// NewToken mints the token for this launch.
func NewToken() (string, error) {
	randomBytes := make([]byte, tokenBytes)

	_, readError := rand.Read(randomBytes)
	if readError != nil {
		return "", fmt.Errorf("minting the editor token: %w", readError)
	}

	return hex.EncodeToString(randomBytes), nil
}

// NewHandler returns the editor's HTTP handler, guarded as the package
// documentation describes.
func NewHandler(options Options) http.Handler {
	pages := newPageHandlers(options)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", pages.redirectHome)
	mux.HandleFunc("GET /page", pages.showPage)
	mux.HandleFunc("POST /page/add", pages.addPage)
	mux.HandleFunc("POST /page/rename", pages.renamePage)
	mux.HandleFunc("POST /page/delete", pages.deletePage)
	mux.HandleFunc("GET /key", pages.showKey)
	mux.HandleFunc("POST /key", pages.saveKey)
	mux.HandleFunc("POST /key/delete", pages.deleteKey)
	mux.HandleFunc("POST /key/move", pages.moveKey)
	mux.HandleFunc("GET /preview.png", pages.preview)
	mux.HandleFunc("GET /icons", pages.showIcons)
	mux.HandleFunc("GET /icon.png", pages.iconPreview)
	mux.HandleFunc("GET /credentials", pages.showCredentials)
	mux.HandleFunc("POST /credentials", pages.storeCredential)
	mux.HandleFunc("GET /diagnostics", pages.showDiagnostics)
	mux.Handle("GET /static/", http.FileServerFS(embeddedFiles))

	return guard{options: options, next: mux}
}

// guard enforces the editor's request rules before any handler runs.
type guard struct {
	options Options
	next    http.Handler
}

func (requestGuard guard) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	setProtectiveHeaders(writer.Header())

	if !requestGuard.isOwnHost(request.Host) {
		requestGuard.refuse(writer, request, refusedHost, http.StatusForbidden)

		return
	}

	origin := request.Header.Get("Origin")
	changes := request.Method != http.MethodGet && request.Method != http.MethodHead

	if (origin != "" || changes) && !requestGuard.isOwnOrigin(origin) {
		requestGuard.refuse(writer, request, refusedOrigin, http.StatusForbidden)

		return
	}

	if changes {
		request.Body = http.MaxBytesReader(writer, request.Body, maximumRequestBytes)

		parseError := request.ParseForm()
		if _, tooLarge := errors.AsType[*http.MaxBytesError](parseError); tooLarge {
			requestGuard.refuse(writer, request, refusedSize, http.StatusRequestEntityTooLarge)

			return
		}

		if parseError != nil || !requestGuard.isOwnToken(request.PostForm.Get(tokenFieldName)) {
			requestGuard.refuse(writer, request, refusedToken, http.StatusForbidden)

			return
		}
	}

	requestGuard.next.ServeHTTP(writer, request)
}

// isOwnHost accepts the editor's address, or the same port under the name
// localhost, which a person may type.
func (requestGuard guard) isOwnHost(host string) bool {
	_, port, splitError := net.SplitHostPort(requestGuard.options.Address)

	return host == requestGuard.options.Address || (splitError == nil && host == net.JoinHostPort("localhost", port))
}

func (requestGuard guard) isOwnOrigin(origin string) bool {
	_, port, splitError := net.SplitHostPort(requestGuard.options.Address)

	return origin == "http://"+requestGuard.options.Address ||
		(splitError == nil && origin == "http://"+net.JoinHostPort("localhost", port))
}

func (requestGuard guard) isOwnToken(token string) bool {
	return token != "" && subtle.ConstantTimeCompare([]byte(token), []byte(requestGuard.options.Token)) == 1
}

// refuse counts and logs a refused request by reason, method and path, never
// its body, query or headers (datum 0.15).
func (requestGuard guard) refuse(writer http.ResponseWriter, request *http.Request, reason string, status int) {
	requestGuard.options.Counters.Increment("editor.rejected." + reason)
	requestGuard.options.Logger.WarnContext(request.Context(), "editor request refused",
		slog.String("reason", reason), slog.String("method", request.Method), slog.String("path", request.URL.Path))
	http.Error(writer, http.StatusText(status), status)
}

// setProtectiveHeaders keeps the editor's pages from running anything but its
// own files, from being framed, and from being cached or leaking their address.
// The referrer policy is same-origin, not no-referrer: under no-referrer a
// browser sends Origin: null on a plain form submit, which the guard refuses.
func setProtectiveHeaders(header http.Header) {
	header.Set("Content-Security-Policy",
		"default-src 'self'; img-src 'self'; style-src 'self'; script-src 'self'; frame-ancestors 'none'; form-action 'self'")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "same-origin")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Frame-Options", "DENY")
}
