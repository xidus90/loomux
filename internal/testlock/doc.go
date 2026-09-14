// Package testlock builds the one world several packages' tests need and no
// portable call provides: a manifest that exists as a regular file and cannot
// be read. It was a pair of test files in pkg/maintenance until the manifest
// reader and the visibility gates of pkg/mcp, pkg/search and cmd/brain needed
// the same world; test files cannot be imported across packages.
//
// Test support, not product code, and not held to the coverage floor: the
// branches it has fire only when the world cannot be built, and a test that
// provoked them would be a test of the operating system.
package testlock
