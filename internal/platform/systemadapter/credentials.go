package systemadapter

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// KeyringCredentialStore is the CredentialStore adapter for both operating
// systems (TD12): go-keyring uses the Keychain through /usr/bin/security on
// macOS and Credential Manager on Windows. Every secret is filed under the
// product's service name, so this program's entries are easy to find and remove.
type KeyringCredentialStore struct {
	service string
}

// NewKeyringCredentialStore returns the store for this product.
func NewKeyringCredentialStore() KeyringCredentialStore {
	return KeyringCredentialStore{service: platform.ProductIdentifier}
}

// ReadCredential returns the secret stored under a name.
func (store KeyringCredentialStore) ReadCredential(credentialName string) (string, error) {
	secret, readError := keyring.Get(store.service, credentialName)

	switch {
	case errors.Is(readError, keyring.ErrNotFound):
		return "", fmt.Errorf("%w: %s", platform.ErrCredentialMissing, credentialName)
	case readError != nil:
		return "", fmt.Errorf("reading credential %s: %w", credentialName, readError)
	default:
		return secret, nil
	}
}

// WriteCredential stores a secret under a name, replacing any earlier one.
func (store KeyringCredentialStore) WriteCredential(credentialName string, secret string) error {
	checkError := platform.CheckCredential(credentialName, secret)
	if checkError != nil {
		return fmt.Errorf("storing credential %s: %w", credentialName, checkError)
	}

	// go-keyring's own errors never include the secret; only the name is added.
	writeError := keyring.Set(store.service, credentialName, secret)
	if writeError != nil {
		return fmt.Errorf("storing credential %s: %w", credentialName, writeError)
	}

	return nil
}

// RemoveCredential deletes the secret stored under a name. It exists for the
// contract test's clean up; removing a credential a person stored is theirs to
// do in their operating system's credential manager.
func (store KeyringCredentialStore) RemoveCredential(credentialName string) error {
	removeError := keyring.Delete(store.service, credentialName)
	if removeError != nil && !errors.Is(removeError, keyring.ErrNotFound) {
		return fmt.Errorf("removing credential %s: %w", credentialName, removeError)
	}

	return nil
}
