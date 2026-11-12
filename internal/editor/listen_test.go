package editor_test

import (
	"net"
	"strconv"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/editor"
)

// loopback is the only address the editor may listen on.
const loopback = "127.0.0.1"

// listenAnyLoopback opens a loopback listener on a port the system chooses.
func listenAnyLoopback(t *testing.T) net.Listener {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, listenError := listenConfig.Listen(t.Context(), "tcp", net.JoinHostPort(loopback, "0"))
	if listenError != nil {
		t.Fatal(listenError)
	}

	return listener
}

// freePort finds a loopback port nothing is listening on, by opening and
// closing one the operating system chose.
func freePort(t *testing.T) int {
	t.Helper()

	probe := listenAnyLoopback(t)
	port := probe.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // a TCP listener's address is a TCP address

	closeError := probe.Close()
	if closeError != nil {
		t.Fatal(closeError)
	}

	return port
}

func listenedPort(t *testing.T, listener net.Listener) (string, int) {
	t.Helper()

	t.Cleanup(func() { _ = listener.Close() })

	address, _ := listener.Addr().(*net.TCPAddr)

	return address.IP.String(), address.Port
}

// TestListen_UsesThePreferredPort_WhenItIsFree: a new port at every launch broke
// every bookmark, and on the real deck the author found the editor's address
// changing all the time (evaluation, deck testing).
func TestListen_UsesThePreferredPort_WhenItIsFree(t *testing.T) {
	t.Parallel()

	preferred := freePort(t)

	listener, fellBack, listenError := editor.Listen(t.Context(), preferred)
	if listenError != nil {
		t.Fatal(listenError)
	}

	host, port := listenedPort(t, listener)
	if host != loopback || port != preferred || fellBack {
		t.Fatalf("want 127.0.0.1:%d without falling back, got %s:%d (fell back %v)", preferred, host, port, fellBack)
	}
}

// TestListen_FallsBackToAnyPort_WhenThePreferredOneIsTaken keeps the editor
// working when another program holds the port, and says so.
func TestListen_FallsBackToAnyPort_WhenThePreferredOneIsTaken(t *testing.T) {
	t.Parallel()

	holder := listenAnyLoopback(t)
	t.Cleanup(func() { _ = holder.Close() })

	taken := holder.Addr().(*net.TCPAddr).Port //nolint:forcetypeassert // a TCP listener's address is a TCP address

	listener, fellBack, listenError := editor.Listen(t.Context(), taken)
	if listenError != nil {
		t.Fatal(listenError)
	}

	host, port := listenedPort(t, listener)
	if host != loopback || port == taken || port == 0 || !fellBack {
		t.Fatalf("want another loopback port and a fall back reported, got %s:%d (fell back %v)", host, port, fellBack)
	}
}

// TestListen_ChoosesAnyPort_WhenNoneIsPreferred keeps zero meaning the
// operating system's choice.
func TestListen_ChoosesAnyPort_WhenNoneIsPreferred(t *testing.T) {
	t.Parallel()

	listener, fellBack, listenError := editor.Listen(t.Context(), 0)
	if listenError != nil {
		t.Fatal(listenError)
	}

	host, port := listenedPort(t, listener)
	if host != loopback || port == 0 || fellBack {
		t.Fatalf("want an operating system chosen loopback port, got %s:%s (fell back %v)",
			host, strconv.Itoa(port), fellBack)
	}
}

func TestDefaultEditorPort_IsFixed(t *testing.T) {
	t.Parallel()

	if editor.DefaultPort != 48080 {
		t.Fatalf("want the editor's default port to be 48080, got %d", editor.DefaultPort)
	}
}
