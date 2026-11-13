package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/editor"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The editor's session file, in the state folder: its address and token, one
// to a line, readable only by its owner. The daemon reuses the token across
// restarts, so an open editor keeps its session, and the -open-editor command
// reads both to ask for a login link (security sweep).
const (
	editorSessionFileName        = "editor-session"
	editorSessionFilePermissions = 0o600
	editorTokenHexLength         = 64
	loginLinkTimeout             = 5 * time.Second
	maximumLoginLinkBytes        = 1024
)

// errNoEditor reports no running editor to ask for a login link.
var errNoEditor = errors.New("the editor is not running; start the daemon first")

// errForeignLink reports an answer that is not a link to the editor asked.
var errForeignLink = errors.New("the editor's answer was not a login link to itself")

// editorSessionToken is the token saved in the state folder, or a new one.
func editorSessionToken(stateDirectory string) (string, error) {
	_, token, readError := readEditorSession(stateDirectory)
	if readError == nil {
		return token, nil
	}

	return editor.NewToken() //nolint:wrapcheck // NewToken says what failed
}

// writeEditorSession saves the editor's address and token for -open-editor.
func writeEditorSession(stateDirectory string, address string, token string) error {
	path := filepath.Join(stateDirectory, editorSessionFileName)

	writeError := os.WriteFile(path, []byte(address+"\n"+token+"\n"), editorSessionFilePermissions)
	if writeError != nil {
		return fmt.Errorf("saving the editor's session: %w", writeError)
	}

	// WriteFile keeps an existing file's mode, so it is set again.
	chmodError := os.Chmod(path, editorSessionFilePermissions)
	if chmodError != nil {
		return fmt.Errorf("saving the editor's session: %w", chmodError)
	}

	return nil
}

// readEditorSession reads the saved address and token, refusing anything but
// a loopback address and a 64 character hexadecimal token.
func readEditorSession(stateDirectory string) (string, string, error) {
	content, readError := os.ReadFile(filepath.Clean(filepath.Join(stateDirectory, editorSessionFileName)))
	if readError != nil {
		return "", "", fmt.Errorf("reading the editor's session: %w", readError)
	}

	address, token, _ := strings.Cut(strings.TrimSpace(string(content)), "\n")
	host, _, splitError := net.SplitHostPort(address)

	_, hexError := hex.DecodeString(token)
	if splitError != nil || host != "127.0.0.1" || len(token) != editorTokenHexLength || hexError != nil {
		return "", "", fmt.Errorf("%w: the session file is not one the daemon wrote", errNoEditor)
	}

	return address, token, nil
}

// requestLoginLink asks the editor at the address for a one time login link,
// sending the token as a header, never in a command line.
func requestLoginLink(linkContext context.Context, client *http.Client, address string, token string) (string, error) {
	request, requestError := http.NewRequestWithContext(linkContext, http.MethodPost,
		"http://"+address+"/session/code", http.NoBody)
	if requestError != nil {
		return "", fmt.Errorf("asking for a login link: %w", requestError)
	}

	request.Header.Set("Authorization", "Bearer "+token)

	response, sendError := client.Do(request)
	if sendError != nil {
		return "", fmt.Errorf("%w: %w", errNoEditor, sendError)
	}

	defer func() { _ = response.Body.Close() }()

	body, bodyError := io.ReadAll(io.LimitReader(response.Body, maximumLoginLinkBytes))
	if bodyError != nil || response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", errForeignLink, response.StatusCode)
	}

	link := strings.TrimSpace(string(body))
	if !strings.HasPrefix(link, "http://"+address+"/session?code=") {
		return "", errForeignLink
	}

	return link, nil
}

// openEditor is the -open-editor command: it opens the running editor in the
// browser with a fresh login link.
func openEditor() int {
	stateDirectory, locationError := systemadapter.UserConfigLocation{}.StateDirectory()
	if locationError != nil {
		printError(locationError.Error())

		return 1
	}

	address, token, readError := readEditorSession(stateDirectory)
	if readError != nil {
		printError(errNoEditor.Error())

		return 1
	}

	linkContext, cancel := context.WithTimeout(context.Background(), loginLinkTimeout)
	defer cancel()

	client := &http.Client{Transport: nil, CheckRedirect: nil, Jar: nil, Timeout: loginLinkTimeout}

	link, linkError := requestLoginLink(linkContext, client, address, token)
	if linkError != nil {
		printError(linkError.Error())

		return 1
	}

	openError := systemadapter.NewURLOpener().OpenURL(linkContext, link)
	if openError != nil {
		printError(openError.Error())

		return 1
	}

	return 0
}
