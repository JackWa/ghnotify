//go:build !darwin

package tray

// reopenMenu is a no-op off macOS.
func reopenMenu() {}
