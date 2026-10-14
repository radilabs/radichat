// Package discovery contains the Phase 5 adversarial investigation harness.
//
// Every executable file in this package is guarded by the mockdiscovery build
// tag so the default build, vet, and test suites never see it:
//
//	go test -tags mockdiscovery ./internal/discovery -count=1 -run 'P5' -v
//
// The harness is black-box. It builds the real RadiChat executable once into a
// temporary directory and drives that binary with piped stdin and per-case
// temporary config files against the accepted in-process mockllm server. It
// registers its own harness-local p5- scenarios from test code when the
// accepted Phase 4 catalog lacks a needed payload, and it never modifies
// product code, the accepted mock, or any accepted Phase 0-4 artifact.
//
// The binding oracle for every verdict is docs/protocol.md together with
// docs/context-accounting.md and docs/cli.md. Stderr wording is advisory
// unless those documents promise it.
package discovery
