// Package cliutil provides typed errors and their exit-code mapping.
package cliutil

import "errors"

var (
	// ErrUsage indicates user-facing flag/config errors. Exit 2.
	ErrUsage = errors.New("usage error")
	// ErrAbort indicates ferry refused to act for safety reasons. Exit 3.
	ErrAbort = errors.New("aborted")
	// ErrKey indicates a problem with key material. Exit 4.
	ErrKey = errors.New("key error")
	// ErrIntegrity indicates a manifest or snapshot integrity failure. Exit 5.
	ErrIntegrity = errors.New("integrity error")
	// ErrDrift indicates ferry diff found drift. Exit 1 (script-friendly).
	ErrDrift = errors.New("drift")
)

// ExitCodeFor maps an error to ferry's exit code conventions.
//
//	nil          → 0
//	ErrDrift     → 1
//	ErrUsage     → 2
//	ErrAbort     → 3
//	ErrKey       → 4
//	ErrIntegrity → 5
//	other        → 1
func ExitCodeFor(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrDrift):
		return 1
	case errors.Is(err, ErrUsage):
		return 2
	case errors.Is(err, ErrAbort):
		return 3
	case errors.Is(err, ErrKey):
		return 4
	case errors.Is(err, ErrIntegrity):
		return 5
	default:
		return 1
	}
}
