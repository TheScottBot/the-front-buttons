package platform_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The credential name and a placeholder secret the tests use, named once.
const (
	workCredential = "github-work"
	sampleSecret   = "secret"
)

func TestCheckCredential_AcceptsANamedSecretWithinTheBounds(t *testing.T) {
	t.Parallel()

	checkError := platform.CheckCredential(workCredential, strings.Repeat("x", platform.MaximumSecretBytes))
	if checkError != nil {
		t.Fatalf("want accepted, got %v", checkError)
	}
}

func TestCheckCredential_RefusesBadNamesAndSecrets(t *testing.T) {
	t.Parallel()

	refused := map[string][2]string{
		"empty name":        {"", sampleSecret},
		"long name":         {strings.Repeat("n", platform.MaximumCredentialNameLength+1), sampleSecret},
		"control in name":   {"github\nwork", sampleSecret},
		"empty secret":      {workCredential, ""},
		"oversized secret":  {workCredential, strings.Repeat("x", platform.MaximumSecretBytes+1)},
		"newline in secret": {workCredential, "ghp_abc\n"},
	}

	for caseName, credential := range refused {
		checkError := platform.CheckCredential(credential[0], credential[1])
		if !errors.Is(checkError, platform.ErrCredentialRefused) {
			t.Errorf("%s: want ErrCredentialRefused, got %v", caseName, checkError)
		}

		if credential[1] != "" && strings.Contains(checkError.Error(), credential[1]) {
			t.Errorf("%s: the error must never contain the secret", caseName)
		}
	}
}

// TestCredentialHost_DefaultsToGitHubAndReadsABinding: a token goes only to
// the host it was stored for, kept beside it in the credential store rather
// than in configuration, so a configuration from elsewhere cannot send it to
// another server. A token stored before bindings is GitHub's.
func TestCredentialHost_DefaultsToGitHubAndReadsABinding(t *testing.T) {
	t.Parallel()

	store := testsupport.NewCredentialStoreTestDouble()

	host, hostError := platform.CredentialHost(store, workCredential)

	if platform.DefaultCredentialHost != "api.github.com" {
		t.Fatalf("want GitHub's API host as the default, got %q", platform.DefaultCredentialHost)
	}

	if hostError != nil || host != platform.DefaultCredentialHost {
		t.Fatalf("want api.github.com for an unbound credential, got %q (%v)", host, hostError)
	}

	writeError := platform.BindCredentialHost(store, workCredential, "github.example.test")
	if writeError != nil {
		t.Fatal(writeError)
	}

	host, _ = platform.CredentialHost(store, workCredential)
	if host != "github.example.test" {
		t.Errorf("want the bound host, got %q", host)
	}

	for _, refused := range []string{"https://github.example.test", "github.example.test/api", "", "a b", "UPPER.test"} {
		if platform.BindCredentialHost(store, workCredential, refused) == nil {
			t.Errorf("want %q refused as a host", refused)
		}
	}

	if platform.BindCredentialHost(store, strings.Repeat("n", platform.MaximumCredentialNameLength), "a.test") == nil {
		t.Error("want a name too long to hold its binding refused")
	}
}
