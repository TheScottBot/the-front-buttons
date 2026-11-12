package editor_test

import (
	"context"
	"image"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/bricks/allbricks"
	"github.com/TheScottBot/the-front-buttons/internal/config"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/layout"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/presets"
	"github.com/TheScottBot/the-front-buttons/internal/render"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The editor's address and token in these tests. The handler is exercised
// directly, so no port is opened.
const (
	testAddress = "127.0.0.1:47123"
	testOrigin  = "http://" + testAddress
	testToken   = "test-token-0123456789abcdef"
)

// The form field names the tests submit, named once.
const (
	pageField   = "page"
	rowField    = "row"
	columnField = "column"
	typeField   = "type"
	urlField    = "field.url"
	nameField   = "name"
	tokenField  = "token"
	openURLType = "kind:open_url"
)

const startingConfiguration = `schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: open_url
        parameters:
          url: https://example.test/docs
          label: Docs
`

// editorHarness is an editor over a real store in a temporary folder.
type editorHarness struct {
	handler     http.Handler
	store       *config.Store
	configPath  string
	credentials *testsupport.CredentialStoreTestDouble
	counters    *diagnostics.Counters
}

// noIcons is an icon set with no icons; the editor's tests are not about icons.
type noIcons struct{}

func (noIcons) IconMask(string, int) (*image.Alpha, bool) { return nil, false }

// galleryIcons names a handful of icons for the gallery's tests, out of order,
// as a real icon set's map would give them.
type galleryIcons struct{}

func (galleryIcons) Names() []string { return []string{"sword", "dice-5", "dragon", "brand-discord"} }

// installedApplications is a fixed catalogue for the app picker's tests, out of
// order as a real one might be.
type installedApplications struct{}

func (installedApplications) Applications(context.Context) ([]platform.Application, error) {
	return []platform.Application{
		{Name: "Terminal", Identifier: "com.apple.Terminal"},
		{Name: "Spotify", Identifier: "com.spotify.client"},
	}, nil
}

func harnessRegistry(t *testing.T, registerExtra func(registry *brick.Registry) error) *brick.Registry {
	t.Helper()

	registry := brick.NewRegistry()

	registerError := allbricks.Register(registry)
	if registerError != nil {
		t.Fatal(registerError)
	}

	if registerExtra != nil {
		extraError := registerExtra(registry)
		if extraError != nil {
			t.Fatal(extraError)
		}
	}

	return registry
}

func newHarness(t *testing.T, registerExtra func(registry *brick.Registry) error) *editorHarness {
	t.Helper()

	registry := harnessRegistry(t, registerExtra)

	presetTable, presetError := presets.BuiltIn(registry)
	if presetError != nil {
		t.Fatal(presetError)
	}

	grid, gridError := layout.NewGrid(4, 8)
	if gridError != nil {
		t.Fatal(gridError)
	}

	configPath := filepath.Join(t.TempDir(), "config.yaml")

	writeError := os.WriteFile(configPath, []byte(startingConfiguration), 0o600)
	if writeError != nil {
		t.Fatal(writeError)
	}

	store := config.NewStore(configPath, config.NewDecoder(registry, presetTable, grid), config.LocalFileSystem{})

	renderer, rendererError := render.NewRenderer(96, noIcons{})
	if rendererError != nil {
		t.Fatal(rendererError)
	}

	harness := &editorHarness{
		handler:     nil,
		store:       store,
		configPath:  configPath,
		credentials: testsupport.NewCredentialStoreTestDouble(),
		counters:    diagnostics.NewCounters(),
	}

	harness.handler = editor.NewHandler(editor.Options{
		Address:     testAddress,
		Token:       testToken,
		Store:       store,
		Registry:    registry,
		Presets:     presetTable,
		Grid:        grid,
		Renderer:    renderer,
		Icons:       galleryIcons{},
		IconFiles:   nil,
		Apps:        installedApplications{},
		Credentials: harness.credentials,
		Counters:    harness.counters,
		Logger:      slog.New(slog.DiscardHandler),
	})

	return harness
}

// get makes a same origin GET, as the editor's own pages do.
func (harness *editorHarness) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+testAddress+path, http.NoBody)
	recorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(recorder, request)

	return recorder
}

// post submits a form as the editor's own pages do: same origin, with the token.
func (harness *editorHarness) post(t *testing.T, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	form.Set("token", testToken)

	return harness.postRaw(t, path, form, func(request *http.Request) { request.Header.Set("Origin", testOrigin) })
}

func (harness *editorHarness) postRaw(
	t *testing.T,
	path string,
	form url.Values,
	adjust func(request *http.Request),
) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+testAddress+path,
		strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	adjust(request)

	recorder := httptest.NewRecorder()
	harness.handler.ServeHTTP(recorder, request)

	return recorder
}

// onDisk decodes the configuration file as saved.
func (harness *editorHarness) onDisk(t *testing.T) config.Configuration {
	t.Helper()

	configuration, diskError := harness.store.OnDisk()
	if diskError != nil {
		t.Fatalf("decoding the saved file: %v", diskError)
	}

	return configuration
}
