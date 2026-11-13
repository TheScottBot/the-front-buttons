package main

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
)

// TestEditorSession_KeepsItsTokenPrivateAndAcrossRestarts: the editor's token
// lives in the state folder, readable only by its owner, so a daemon restart
// keeps an open editor's session and the -open-editor command can read it.
func TestEditorSession_KeepsItsTokenPrivateAndAcrossRestarts(t *testing.T) {
	t.Parallel()

	stateDirectory := t.TempDir()

	first, firstError := editorSessionToken(stateDirectory)
	if firstError != nil || len(first) != 64 {
		t.Fatalf("want a 64 character token minted, got %q (%v)", first, firstError)
	}

	writeError := writeEditorSession(stateDirectory, "127.0.0.1:48080", first)
	if writeError != nil {
		t.Fatal(writeError)
	}

	info, statError := os.Stat(filepath.Join(stateDirectory, editorSessionFileName))
	if statError != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("want the session file readable by its owner only, got %v (%v)", info.Mode().Perm(), statError)
	}

	second, _ := editorSessionToken(stateDirectory)
	if second != first {
		t.Error("want the token kept across a restart")
	}

	address, token, readError := readEditorSession(stateDirectory)
	if readError != nil || address != "127.0.0.1:48080" || token != first {
		t.Errorf("want the address and token read back, got %q %q (%v)", address, token, readError)
	}
}

// TestOpenEditor_AsksTheDaemonForALoginLink: -open-editor sends the token as a
// header to the running editor and takes only a link back to that editor.
func TestOpenEditor_AsksTheDaemonForALoginLink(t *testing.T) {
	t.Parallel()

	listener := listenOnLoopback(t)
	address := listener.Addr().String()
	token := strings.Repeat("ab", 32)

	serve(t, listener, editor.NewHandler(editor.Options{
		Address: address, Token: token, Store: nil, Registry: nil, Presets: presets.Table{}, Grid: layout.Grid{},
		Renderer: nil, Icons: nil, IconFiles: nil, AppIcons: nil, Apps: nil, Credentials: nil, Platform: "",
		Sessions: nil, Counters: diagnostics.NewCounters(), Logger: slog.New(slog.DiscardHandler),
	}))

	link, linkError := requestLoginLink(t.Context(), http.DefaultClient, address, token)
	if linkError != nil || !strings.HasPrefix(link, "http://"+address+"/session?code=") {
		t.Fatalf("want a login link from the editor, got %q (%v)", link, linkError)
	}

	_, refusedError := requestLoginLink(t.Context(), http.DefaultClient, address, "wrong")
	if refusedError == nil {
		t.Error("want a wrong token refused")
	}

	other := listenOnLoopback(t)
	serve(t, other, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "https://attacker.test/session?code=1")
	}))

	_, foreignError := requestLoginLink(t.Context(), http.DefaultClient, other.Addr().String(), token)
	if foreignError == nil {
		t.Error("want a link to anywhere but the editor refused")
	}
}

// listenOnLoopback opens a listener on a free loopback port.
func listenOnLoopback(t *testing.T) net.Listener {
	t.Helper()

	listenConfig := new(net.ListenConfig)

	listener, listenError := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if listenError != nil {
		t.Fatal(listenError)
	}

	return listener
}

// serve serves a handler on a listener until the test ends.
func serve(t *testing.T, listener net.Listener, handler http.Handler) {
	t.Helper()

	server := new(http.Server)
	server.Handler = handler
	server.ReadHeaderTimeout = time.Second

	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })
}
