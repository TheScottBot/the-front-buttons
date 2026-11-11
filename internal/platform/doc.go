// Package platform declares the operating system ports: opening URLs, launching
// applications, credential storage and configuration location.
//
// The ports live apart from their adapters so a brick can depend on a port without
// importing anything that touches the operating system.
package platform
