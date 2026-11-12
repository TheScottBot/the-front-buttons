package testsupport

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// CredentialStoreTestDouble keeps secrets in memory.
type CredentialStoreTestDouble struct {
	mutex   sync.Mutex
	secrets map[string]string
}

// NewCredentialStoreTestDouble returns an empty store.
func NewCredentialStoreTestDouble() *CredentialStoreTestDouble {
	return &CredentialStoreTestDouble{mutex: sync.Mutex{}, secrets: map[string]string{}}
}

// ReadCredential returns a stored secret.
func (store *CredentialStoreTestDouble) ReadCredential(credentialName string) (string, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	secret, stored := store.secrets[credentialName]
	if !stored {
		return "", fmt.Errorf("fake store: %w", platform.ErrCredentialMissing)
	}

	return secret, nil
}

// WriteCredential stores a secret, applying the shared rule first.
func (store *CredentialStoreTestDouble) WriteCredential(credentialName string, secret string) error {
	checkError := platform.CheckCredential(credentialName, secret)
	if checkError != nil {
		return fmt.Errorf("fake store: %w", checkError)
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.secrets[credentialName] = secret

	return nil
}

// RunCredentialStoreContract is the one contract suite for the CredentialStore
// port. credentialName must be one the suite may write and overwrite; the
// caller removes it afterwards when the store is real.
func RunCredentialStoreContract(t *testing.T, store platform.CredentialStore, credentialName string) {
	t.Helper()

	const firstSecret, secondSecret = "contract-secret-one", "contract-secret-two"

	t.Run("reads back what was written, and the latest write wins", func(t *testing.T) {
		for _, secret := range []string{firstSecret, secondSecret} {
			writeError := store.WriteCredential(credentialName, secret)
			if writeError != nil {
				t.Fatalf("writing: %v", writeError)
			}

			read, readError := store.ReadCredential(credentialName)
			if readError != nil || read != secret {
				t.Fatalf("want the secret just written, got an error %v", readError)
			}
		}
	})

	t.Run("reports a name with nothing stored as missing", func(t *testing.T) {
		_, readError := store.ReadCredential(credentialName + "-never-written")
		if !errors.Is(readError, platform.ErrCredentialMissing) {
			t.Fatalf("want ErrCredentialMissing, got %v", readError)
		}
	})

	t.Run("refuses an oversized secret without echoing it", func(t *testing.T) {
		oversized := strings.Repeat("s", platform.MaximumSecretBytes+1)

		writeError := store.WriteCredential(credentialName, oversized)
		if !errors.Is(writeError, platform.ErrCredentialRefused) || strings.Contains(writeError.Error(), oversized) {
			t.Fatalf("want ErrCredentialRefused without the secret in it, got %v", writeError)
		}
	})
}
