//go:build !darwin || !cgo

package notify

// nativePost is only implemented on macOS; elsewhere beeep is used.
func nativePost(_, _, _, _ string) bool { return false }

// Init is a no-op off macOS.
func Init() {}
