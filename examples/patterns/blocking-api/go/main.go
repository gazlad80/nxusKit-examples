// Example: Synchronous (Blocking) API
//
// ## nxusKit Features Demonstrated
// NOTE: This pattern is Rust-specific. Go's concurrency model is fundamentally
// different - goroutines provide cooperative multitasking without requiring
// explicit async/await syntax.
//
// ## Why This Pattern Matters (Rust context)
// In Rust, async code requires a runtime (tokio). The BlockingProvider wrapper
// allows sync code to call async providers. Go doesn't need this - all nxuskit
// APIs are already synchronous from the caller's perspective (goroutine-safe).
//
// See ../rust/src/main.rs for the Rust reference implementation.
// See basic-chat/go for the equivalent Go functionality.
package main

import (
	"fmt"
)

func main() {
	fmt.Println("blocking-api example")
	fmt.Println("")
	fmt.Println("This pattern is Rust-specific. In Rust, async code requires an")
	fmt.Println("async runtime (tokio). The blocking-api feature provides sync wrappers.")
	fmt.Println("")
	fmt.Println("In Go, nxuskit already provides synchronous APIs natively.")
	fmt.Println("Use the basic-chat example for equivalent functionality.")
}
