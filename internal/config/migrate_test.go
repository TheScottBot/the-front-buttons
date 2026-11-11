package config_test

import (
	"strings"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/config"
)

func TestDecode_RunsMigrationChain_WhenFileIsOlderThanCurrent(t *testing.T) {
	t.Parallel()

	// A version 0 file named the parameters block "settings"; the test migration
	// renames it, proving the hook runs before the strict decode.
	oldFile := strings.Replace(validConfiguration, "schema_version: 1", "schema_version: 0", 1)
	oldFile = strings.ReplaceAll(oldFile, "parameters:", "settings:")

	renameMigration := config.RenameKeyFieldMigrationForTest(0, "settings", "parameters")
	decoder := config.WithMigrationsForTest(newTestDecoder(t), renameMigration)

	configuration, decodeError := decoder.Decode([]byte(oldFile))
	if decodeError != nil {
		t.Fatalf("decoding a migrated file: %v", decodeError)
	}

	if configuration.SchemaVersion != config.CurrentSchemaVersion {
		t.Fatalf("want the migrated file at the current version, got %d", configuration.SchemaVersion)
	}
}

func TestDecode_RejectsOlderVersion_WhenNoMigrationCoversIt(t *testing.T) {
	t.Parallel()

	oldFile := strings.Replace(validConfiguration, "schema_version: 1", "schema_version: 0", 1)

	requireProblem(t, oldFile, config.ReasonUnsupportedVersion)
}
