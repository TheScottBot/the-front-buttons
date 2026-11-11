package config

import "go.yaml.in/yaml/v4"

// migration rewrites a file's node tree from one schema version to the next.
// Migrations run on the parsed tree before the strict decode, so a migrated file
// is validated exactly as a file written at the current version would be.
type migration struct {
	fromVersion int64
	migrate     func(root *yaml.Node) error
}

// productionMigrations is this build's chain of migrations. Version 1 is the
// first schema, so there are none yet; each breaking change to the contract adds
// one here with its migration note (datum 0.11).
func productionMigrations() []migration {
	return nil
}

func (decoder Decoder) migrationFrom(version int64) (migration, bool) {
	for _, step := range decoder.migrations {
		if step.fromVersion == version {
			return step, true
		}
	}

	return migration{fromVersion: 0, migrate: nil}, false
}
