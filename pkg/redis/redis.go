// Package redis allows reading/writing from/to a Redis DB.
package redis

import (
	"fmt"
	"github.com/stickermule/rump/pkg/message"
	"github.com/mediocregopher/radix/v3"
)

// Redis holds references to a DB pool and a shared message bus.
type Redis struct {
	Pool *radix.Pool
	Bus message.Bus
}

// New creates the Redis struct, used to read/write.
func New(source *radix.Pool, bus message.Bus) *Redis {
	return &Redis{
		Pool: source,
		Bus: bus,
	}
}

// Read gently scans an entire Redis DB for keys, then dumps
// the key/value pair (Payload) on the message Bus channel.
// It leverages implicit pipelining to speedup large DB reads.
// To be used in an ErrGroup.
func (r *Redis) Read() error {
	scanner := radix.NewScanner(r.Pool, radix.ScanAllKeys)

	var key string
	var value string

	// Scan and push to bus until until no keys are left.
	for scanner.Next(&key) {
		err := r.Pool.Do(radix.Cmd(&value, "DUMP", key))
		if err != nil {
			return err
		}

		r.Bus <- message.Payload{Key: key, Value: value}
		fmt.Printf("r")
	}

	// Scan completed, close channel.
	close(r.Bus)

	return scanner.Close()
}

// Write restores keys on the db as they come on the message bus.
func (r *Redis) Write() error {
	for p := range r.Bus {
		err := r.Pool.Do(radix.Cmd(nil, "RESTORE", p.Key, "0", p.Value, "REPLACE"))
		if err != nil {
			return err
		}

		fmt.Printf("w")
	}
	return nil
}
