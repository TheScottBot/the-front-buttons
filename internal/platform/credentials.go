package platform

import (
	"errors"
	"fmt"
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
