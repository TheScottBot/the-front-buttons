// Package systemadapter holds the build tagged macOS and Windows adapters for the
// ports declared in package platform.
//
// Only the composition root may import it, which the import layering check
// enforces, so operating system behaviour never leaks into the core.
package systemadapter
