package systemadapter_test

import (
	"os"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// credentialTestsVariable enables the real store's contract run. It is off by
// default because it writes to the user's Keychain or Credential Manager; the
// author runs it as part of the TE8 gate.
const credentialTestsVariable = "THE_FRONT_BUTTONS_CREDENTIAL_TESTS"

// contractCredentialName is the one entry the real contract run writes and then
// removes, so it can never collide with a credential a person configured.
const contractCredentialName = "contract-test-entry"

func TestCredentialStoreTestDouble_PassesTheCredentialStoreContract(t *testing.T) {
	t.Parallel()

	testsupport.RunCredentialStoreContract(t, testsupport.NewCredentialStoreTestDouble(), contractCredentialName)
}

func TestKeyringCredentialStore_PassesTheCredentialStoreContract_WithConsent(t *testing.T) {
	t.Parallel()

	if os.Getenv(credentialTestsVariable) != "1" {
		t.Skipf("set %s=1 to run the contract against the real credential store (TE8 author gate)",
			credentialTestsVariable)
	}

	store := systemadapter.NewKeyringCredentialStore()

	t.Cleanup(func() {
		removeError := store.RemoveCredential(contractCredentialName)
		if removeError != nil {
			t.Errorf("removing the contract entry: %v", removeError)
		}
	})

	testsupport.RunCredentialStoreContract(t, store, contractCredentialName)
}
