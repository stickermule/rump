// +build windows

package signals

// Init initializes the signal watcher with the handler function that will be invoked on USR1 signal
func Init(handler func()) {
	// NOOP: Windows doesn't have signals equivalent to the Unix world.
}
