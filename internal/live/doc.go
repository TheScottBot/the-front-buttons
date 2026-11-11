// Package live runs every live brick, shares pollers between keys watching the same
// source, and caches the latest appearance per key.
//
// The cache lets a page switch show current state at once instead of waiting on a
// brick.
package live
