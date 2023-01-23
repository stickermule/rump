// Package signal handles OS signals and gracefully exits.
// It's used in an ErrGroup to signal exit to other goroutines.
package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Run will be run in an ErrGroup supervisor.
func Run(ctx context.Context, cancel context.CancelFunc) error {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signalChannel:
		fmt.Println("signal: ", sig)
		cancel()
	case <-ctx.Done():
		fmt.Println("signal: done")
		return ctx.Err()
	}

	return nil
}
