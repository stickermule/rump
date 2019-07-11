// Package redis allows reading/writing from/to a Redis DB.
package redis

import (
	"context"
	"fmt"

	"github.com/mediocregopher/radix/v3"

	"github.com/stickermule/rump/pkg/message"
)

var (
	batchSize = 100
)

// Redis holds references to a DB pool and a shared message bus.
// Silent disables verbose mode.
// TTL enables TTL sync.
type Redis struct {
	Pool   *radix.Pool
	Bus    message.Bus
	Silent bool
	TTL    bool
}

// New creates the Redis struct, used to read/write.
func New(source *radix.Pool, bus message.Bus, silent, ttl bool) *Redis {
	return &Redis{
		Pool:   source,
		Bus:    bus,
		Silent: silent,
		TTL:    ttl,
	}
}

// maybeLog may log, depending on the Silent flag
func (r *Redis) maybeLog(s string) {
	if r.Silent {
		return
	}
	fmt.Print(s)
}

// Read gently scans an entire Redis DB for keys, then dumps
// the key/value pair (Payload) on the message Bus channel.
// It leverages implicit pipelining to speedup large DB reads.
// To be used in an ErrGroup.
func (r *Redis) Read(ctx context.Context) error {
	defer close(r.Bus)

	scanner := radix.NewScanner(r.Pool, radix.ScanAllKeys)

	var key string

	// Scan and push to bus until no keys are left.
	// If context Done, exit early.
	keys := make([]string, 0, batchSize)

	for scanner.Next(&key) {
		keys = append(keys, key)
		// check if we have enough keys
		if len(keys) == batchSize {
			err := r.ProcessKeys(ctx, keys)
			if err != nil {
				return err
			}

			// clear keys
			keys = keys[:0]
		}
	}

	// process remaining keys
	if len(keys) > 0 {
		err := r.ProcessKeys(ctx, keys)
		if err != nil {
			return err
		}
	}

	return scanner.Close()
}

// ProcessKeys ...
func (r *Redis) ProcessKeys(ctx context.Context, keys []string) error {
	batch, err := r.GenerateBatch(keys)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		fmt.Println("")
		fmt.Println("redis read: exit")
		return ctx.Err()
	case r.Bus <- batch:
		r.maybeLog("r")
	}

	return nil
}

// GenerateBatch ...
func (r *Redis) GenerateBatch(keys []string) ([]*message.Payload, error) {
	batch := make([]*message.Payload, 0, batchSize)
	valueActions := make([]radix.CmdAction, 0, 100)
	ttlActions := make([]radix.CmdAction, 0, 100)

	for _, k := range keys {
		p := &message.Payload{Key: k, TTL: "0"}
		batch = append(batch, p)
		valueActions = append(valueActions, radix.Cmd(&p.Value, "DUMP", p.Key))
		if r.TTL {
			ttlActions = append(ttlActions, radix.Cmd(&p.TTL, "PTTL", p.Key))
		}
	}

	// fetch values
	err := r.Pool.Do(radix.Pipeline(valueActions...))
	if err != nil {
		return batch, err
	}

	if r.TTL {
		// fetch TTL
		err = r.Pool.Do(radix.Pipeline(ttlActions...))
		if err != nil {
			return batch, err
		}
	}

	return batch, nil
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
		case batch, ok := <-r.Bus:
			// if channel closed, set to nil, break loop
			if !ok {
				r.Bus = nil
				continue
			}

			actions := make([]radix.CmdAction, 0, batchSize)
			for _, p := range batch {
				// When key has no expire PTTL returns "-1".
				// We set it to 0, default for no expiration time.
				if p.TTL == "-1" {
					p.TTL = "0"
				}
				actions = append(actions,
					radix.Cmd(nil, "RESTORE", p.Key, p.TTL, p.Value, "REPLACE"),
				)
			}

			err := r.Pool.Do(radix.Pipeline(actions...))
			if err != nil {
				return err
			}
			r.maybeLog("w")
		}
	}

	return nil
}
