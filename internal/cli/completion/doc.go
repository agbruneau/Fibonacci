// Package completion provides shell completion script generation for fibcalc
// (bash, zsh, fish and PowerShell).
//
// All shell completion functions are generated from a single source of truth:
// flagRegistry. Adding a new flag only requires appending to that registry.
//
// Generators emit identifiers into shell scripts: any addition must keep the
// escaping of flag names and values correct for each target shell (a latent
// injection risk documented in the project guidelines).
package completion
