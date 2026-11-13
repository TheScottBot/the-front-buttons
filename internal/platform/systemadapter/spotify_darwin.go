package systemadapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg" // Spotify's album art is JPEG.
	"io"
	"net/http"
	"os/exec"
	"time"
)

// The bounds on reading album art: Spotify's art is a few hundred kilobytes
// at most, and a key needs it within a moment.
const (
	maximumArtworkBytes = 2 << 20
	artworkTimeout      = 5 * time.Second
	osascriptProgram    = "/usr/bin/osascript"
)

// errArtworkStatus reports album art the server would not give.
var errArtworkStatus = errors.New("album art refused")

// NewMediaPlayer returns this machine's MediaPlayer: Spotify, through
// osascript. The first command asks macOS for the Automation permission to
// control Spotify.
func NewMediaPlayer() *Spotify {
	client := &http.Client{Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: artworkTimeout}

	return NewSpotify(runAppleScript, func(fetchContext context.Context, address string) (image.Image, error) {
		return fetchArtwork(fetchContext, client, address)
	})
}

// runAppleScript runs one script through osascript, with no shell. Its error
// carries osascript's message, which names a refused permission's number.
func runAppleScript(runContext context.Context, script string) (string, error) {
	var output, failure bytes.Buffer

	// The program is fixed and the script is one of the adapter's own
	// constants, passed as one argument with no shell.
	command := exec.CommandContext(runContext, osascriptProgram, "-e", script) //nolint:gosec // see above
	command.Stdout, command.Stderr = &output, &failure

	runError := command.Run()
	if runError != nil {
		return "", fmt.Errorf("osascript: %s: %w", bytes.TrimSpace(failure.Bytes()), runError)
	}

	return output.String(), nil
}

// fetchArtwork reads album art from Spotify's image host into a picture the
// renderer can scale.
func fetchArtwork(fetchContext context.Context, client *http.Client, address string) (image.Image, error) {
	request, requestError := http.NewRequestWithContext(fetchContext, http.MethodGet, address, http.NoBody)
	if requestError != nil {
		return nil, fmt.Errorf("album art request: %w", requestError)
	}

	response, responseError := client.Do(request)
	if responseError != nil {
		return nil, fmt.Errorf("album art: %w", responseError)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errArtworkStatus, response.StatusCode)
	}

	decoded, _, decodeError := image.Decode(io.LimitReader(response.Body, maximumArtworkBytes))
	if decodeError != nil {
		return nil, fmt.Errorf("album art: %w", decodeError)
	}

	// A plain RGBA picture is always a pointer, so live states holding it
	// compare.
	picture := image.NewRGBA(decoded.Bounds())
	draw.Draw(picture, picture.Bounds(), decoded, decoded.Bounds().Min, draw.Src)

	return picture, nil
}
