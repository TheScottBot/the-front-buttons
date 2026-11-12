package platform_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
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
