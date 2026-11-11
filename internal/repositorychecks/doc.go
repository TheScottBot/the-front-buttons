// Package repositorychecks holds the tree wide checks that continuous integration
// runs over the repository: the prose and typography scan, the operating system
// use check, and the README first line check.
//
// The decisions live in pure functions over a path and its bytes, so each check
// can be proven against a planted violation in a unit test (engineering datum 0.4
// and Part 3) before it is trusted on the real tree. Only ScanRepository touches
// the filesystem, and it does nothing but read files and hand them to those
// functions.
package repositorychecks
