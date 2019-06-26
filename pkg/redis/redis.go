// Package redis allows reading/writing from/to a Redis DB.
package redis

import (
	"context"
	"fmt"

	"github.com/mediocregopher/radix/v3"

	"github.com/stickermule/rump/pkg/message"
)

// Redis holds references to a DB pool and a shared message bus.
type Redis struct {
	Pool *radix.Pool
	Bus  message.Bus
	Silent bool
}


// New creates the Redis struct, used to read/write.
func New(source *radix.Pool, bus message.Bus, silent bool) *Redis {
	return &Redis{
		Pool: source,
		Bus:  bus,
		Silent: silent,
	}
}

// Log read/write operations unless silent mode enabled
func (r *Redis) log(s string) {
	if r.Silent {
		return
	}
	fmt.Printf(s)
}

// Read gently scans an entire Redis DB for keys, then dumps
// the key/value pair (Payload) on the message Bus channel.
// It leverages implicit pipelining to speedup large DB reads.
// To be used in an ErrGroup.
func (r *Redis) Read(ctx context.Context) error {
	defer close(r.Bus)

	scanner := radix.NewScanner(r.Pool, radix.ScanAllKeys)

	var key string
	var value string

	// Scan and push to bus until no keys are left.
	// If context Done, exit early.
	for scanner.Next(&key) {
		err := r.Pool.Do(radix.Cmd(&value, "DUMP", key))
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("redis read: exit")
			return ctx.Err()
		case r.Bus <- message.Payload{Key: key, Value: value}:
			r.log("r")
		}
	}

	return scanner.Close()
}

// Write restores keys on the db as they come on the message bus.
func (r *Redis) Write(ctx context.Context) error {
	// Loop until channel is open
	for r.Bus != nil {
		select {
		// Exit early if context done.
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("redis write: exit")
			return ctx.Err()
		// Get Messages from Bus
		case p, ok := <-r.Bus:
			// if channel closed, set to nil, break loop
			if !ok {
				r.Bus = nil
				continue
			}
			err := r.Pool.Do(radix.Cmd(nil, "RESTORE", p.Key, "0", p.Value, "REPLACE"))
			if err != nil {
				return err
			}
			r.log("w")
		}
	}

	return nil
}
