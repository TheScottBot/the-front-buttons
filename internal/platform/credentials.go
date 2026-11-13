package platform

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrCredentialMissing reports a credential name with nothing stored under it.
var ErrCredentialMissing = errors.New("credential missing")

// ErrCredentialRefused reports a credential name or secret refused before the
// store was touched.
var ErrCredentialRefused = errors.New("credential refused")

// CredentialStore reads and writes named secrets in the operating system's
// store: the Keychain on macOS, Credential Manager on Windows. Secrets never
// enter configuration, logs or the editor's responses; configuration names them
// (specification section 2).
type CredentialStore interface {
	ReadCredential(credentialName string) (string, error)
	WriteCredential(credentialName string, secret string) error
}

// The credential bounds. Windows Credential Manager refuses secrets over 2,560
// bytes (evaluation 6.3), so 2,048 leaves headroom for go-keyring's encoding; a
// GitHub fine grained token is under a hundred characters. A name is bounded as
// configuration bounds a credential field.
const (
	MaximumSecretBytes          = 2048
	MaximumCredentialNameLength = 64
)

// CheckCredential is the one rule every credential adapter applies before
// touching the store. Its errors name the problem, never the secret.
func CheckCredential(credentialName string, secret string) error {
	switch {
	case credentialName == "" || len(credentialName) > MaximumCredentialNameLength:
		return fmt.Errorf("%w: a name must be 1 to %d bytes", ErrCredentialRefused, MaximumCredentialNameLength)
	case containsControlCharacter(credentialName):
		return fmt.Errorf("%w: the name holds a control character", ErrCredentialRefused)
	case secret == "" || len(secret) > MaximumSecretBytes:
		return fmt.Errorf("%w: a secret must be 1 to %d bytes", ErrCredentialRefused, MaximumSecretBytes)
	case containsControlCharacter(secret):
		return fmt.Errorf("%w: the secret holds a control character, often a pasted line ending", ErrCredentialRefused)
	default:
		return nil
	}
}

func containsControlCharacter(text string) bool {
	for _, character := range text {
		if character <= lastControlCharacter || character == deleteCharacter {
			return true
		}
	}

	return false
}

// A token is sent only to the host it was stored for: api.github.com unless a
// person binds it to another, such as a GitHub Enterprise host, when storing
// it. The binding lives beside the token in the credential store, under the
// token's name with credentialHostSuffix, never in configuration, so a
// configuration from elsewhere cannot send the token to another server.
const (
	DefaultCredentialHost = "api.github.com" //nolint:gosec // a host's name, not a credential
	credentialHostSuffix  = ".host"
	// MaximumBindableNameLength is the longest name whose binding still fits
	// the store's own name bound.
	MaximumBindableNameLength = MaximumCredentialNameLength - len(credentialHostSuffix)
	maximumHostLength         = 253
)

// ErrCredentialHost reports a request to a host the credential was not stored
// for.
var ErrCredentialHost = errors.New("credential not stored for this host")

// CredentialHost is the host a credential may be sent to: its binding, or
// DefaultCredentialHost for a credential stored without one.
func CredentialHost(store CredentialStore, credentialName string) (string, error) {
	host, readError := store.ReadCredential(credentialName + credentialHostSuffix)

	switch {
	case errors.Is(readError, ErrCredentialMissing):
		return DefaultCredentialHost, nil
	case readError != nil:
		return "", fmt.Errorf("reading the host for %s: %w", credentialName, readError)
	default:
		return host, nil
	}
}

// BindCredentialHost records the host a credential may be sent to.
func BindCredentialHost(store CredentialStore, credentialName string, host string) error {
	hostError := CheckCredentialHost(host)
	if hostError != nil {
		return hostError
	}

	if len(credentialName) > MaximumBindableNameLength {
		return fmt.Errorf("%w: a name must be at most %d bytes", ErrCredentialRefused, MaximumBindableNameLength)
	}

	// The store's own errors name the entry, never the secret.
	return store.WriteCredential(credentialName+credentialHostSuffix, host) //nolint:wrapcheck // see above
}

// CheckCredentialHost accepts a plain host name, lower case, with an optional
// port: no scheme, path, user or space.
func CheckCredentialHost(host string) error {
	name, port, hasPort := strings.Cut(host, ":")

	switch {
	case !isHostName(name) || len(host) > maximumHostLength:
		return fmt.Errorf("%w: a host is a plain name such as github.example.com", ErrCredentialRefused)
	case hasPort && !isPort(port):
		return fmt.Errorf("%w: a port is digits after a colon", ErrCredentialRefused)
	default:
		return nil
	}
}

// minimumHostLabels is a host's least labels, a name and a domain.
const minimumHostLabels = 2

// isHostName reports dotted labels of lower case letters, digits and inner
// hyphens.
func isHostName(name string) bool {
	labels := strings.Split(name, ".")

	return len(labels) >= minimumHostLabels && !slices.ContainsFunc(labels, func(label string) bool {
		return !isHostLabel(label)
	})
}

func isHostLabel(label string) bool {
	if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
		return false
	}

	return !strings.ContainsFunc(label, func(character rune) bool {
		return (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-'
	})
}

const maximumPortDigits = 5

func isPort(port string) bool {
	if port == "" || len(port) > maximumPortDigits {
		return false
	}

	for _, character := range port {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}
