// Package scripts provides a placeholder non-main package so that `go test ./...`
// succeeds without attempting to build command binaries named after the
// directory (which would conflict with the existing folder). All executable
// helper programs in this directory are individually marked with `//go:build ignore`
// so they can be invoked via `go run` but are skipped in standard builds & tests.
package scripts
