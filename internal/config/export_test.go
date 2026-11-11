package config

import (
	"errors"

	"go.yaml.in/yaml/v4"
)

// errMigrationShape reports a file the test migration cannot walk.
var errMigrationShape = errors.New("test migration found an unexpected shape")

// WithMigrationsForTest returns the decoder with the given migrations in place
// of this build's, so the migration hook can be proven before any real
// migration exists.
func WithMigrationsForTest(decoder Decoder, migrations ...migration) Decoder {
	decoder.migrations = migrations

	return decoder
}

// RenameKeyFieldMigrationForTest returns a migration from the given version that
// renames a field in every key, the kind of change a real migration makes.
func RenameKeyFieldMigrationForTest(fromVersion int64, oldName string, newName string) migration {
	return migration{
		fromVersion: fromVersion,
		migrate: func(root *yaml.Node) error {
			pagesNode := mappingValue(root, pagesName)
			if pagesNode == nil {
				return errMigrationShape
			}

			for _, pageNode := range pagesNode.Content {
				keysNode := mappingValue(pageNode, keysName)
				if keysNode == nil {
					return errMigrationShape
				}

				for _, keyNode := range keysNode.Content {
					for entryIndex := 0; entryIndex < len(keyNode.Content); entryIndex += 2 {
						if keyNode.Content[entryIndex].Value == oldName {
							keyNode.Content[entryIndex].Value = newName
						}
					}
				}
			}

			return nil
		},
	}
}
