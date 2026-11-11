// Package config loads, validates, migrates, watches and atomically writes the per
// machine configuration file.
//
// An invalid file never replaces the running configuration, so a typo cannot take
// the deck down.
package config
