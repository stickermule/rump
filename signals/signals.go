// +build !windows

package signals

import (
	"os"
	"os/signal"
	"syscall"
)

var signals chan os.Signal

// Init initializes the signal watcher with the handler function that will be invoked on USR1 signal
func Init(handler func()) {
	signals = make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGUSR1)

	go handleSignals(handler)
}

// Invoke the given handler for the USR1 signal
func handleSignals(handler func()) {
	for {
		sig := <-signals
		if sig == syscall.SIGUSR1 {
			handler()
		}
	}
}
