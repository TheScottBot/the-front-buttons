package editor_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestEditor_RefusesARequestForAnotherHost(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	// A DNS rebinding attack reaches the loopback port under the attacker's own
	// host name, so anything but the editor's own address is refused.
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://attacker.test/page", http.NoBody)
	recorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden || harness.counters.Snapshot()["editor.rejected.host"] != 1 {
		t.Fatalf("want 403 counted as a host rejection, got %d %v", recorder.Code, harness.counters.Snapshot())
	}
}

func TestEditor_RefusesARequestFromAnotherOrigin(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	form := url.Values{tokenField: {testToken}, nameField: {"Hostile"}}
	posted := harness.postRaw(t, "/page/add", form, func(request *http.Request) {
		request.Header.Set("Origin", "https://attacker.test")
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+testAddress+"/page", http.NoBody)
	request.Header.Set("Origin", "https://attacker.test")

	fetched := httptest.NewRecorder()
	harness.handler.ServeHTTP(fetched, request)

	if posted.Code != http.StatusForbidden || fetched.Code != http.StatusForbidden ||
		harness.counters.Snapshot()["editor.rejected.origin"] != 2 {
		t.Fatalf("want both refused as origin rejections, got %d and %d %v",
			posted.Code, fetched.Code, harness.counters.Snapshot())
	}

	if len(harness.onDisk(t).Pages) != 1 {
		t.Fatal("want the configuration untouched")
	}
}

func TestEditor_RefusesAChangeWithNoOrigin(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.postRaw(t, "/page/add", url.Values{tokenField: {testToken}, nameField: {"No origin"}},
		func(*http.Request) {})

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("want a change with no Origin refused, got %d", recorder.Code)
	}
}

func TestEditor_RefusesAChangeWithAMissingOrWrongToken(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	for _, token := range []string{"", "wrong-token"} {
		recorder := harness.postRaw(t, "/page/add", url.Values{tokenField: {token}, nameField: {"Forged"}},
			func(request *http.Request) { request.Header.Set("Origin", testOrigin) })

		if recorder.Code != http.StatusForbidden {
			t.Errorf("token %q: want 403, got %d", token, recorder.Code)
		}
	}

	if harness.counters.Snapshot()["editor.rejected.token"] != 2 || len(harness.onDisk(t).Pages) != 1 {
		t.Fatalf("want two token rejections and no change, got %v", harness.counters.Snapshot())
	}
}

func TestEditor_RefusesAnOversizedRequest(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, nil)

	recorder := harness.post(t, "/page/add", url.Values{nameField: {strings.Repeat("x", 128*1024)}})

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("want 413, got %d", recorder.Code)
	}
}

func TestEditor_SetsProtectiveHeaders(t *testing.T) {
	t.Parallel()

	recorder := newHarness(t, nil).get(t, "/page?page=0")

	headers := map[string]string{
		"Content-Security-Policy": "default-src 'self'",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
	}

	for name, wantPrefix := range headers {
		if !strings.HasPrefix(recorder.Header().Get(name), wantPrefix) {
			t.Errorf("%s: want %q, got %q", name, wantPrefix, recorder.Header().Get(name))
		}
	}
}
