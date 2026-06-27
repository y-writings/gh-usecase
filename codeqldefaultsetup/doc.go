// Package codeqldefaultsetup reconciles GitHub CodeQL default setup for a repository.
//
// It reads the current default setup, normalizes the desired language list, and
// patches GitHub only when the current configuration differs from the desired
// configuration.
package codeqldefaultsetup
