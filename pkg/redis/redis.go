// Package redis allows reading/writing from/to a Redis DB.
package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mediocregopher/radix/v3"
	"golang.org/x/sync/errgroup"

	"github.com/stickermule/rump/pkg/message"
)

// Redis holds references to a DB pool and a shared message bus.
// Silent disables verbose mode.
// TTL enables TTL sync.
type Redis struct {
	Pool       *radix.Pool
	Bus        message.Bus
	ReadBus    message.Bus
	Silent     bool
	TTL        bool
	DefaultTTL int
	Count      int
	Pattern    string
	Parallel   int
}

// New creates the Redis struct, used to read/write.
func New(source *radix.Pool, bus message.Bus, silent, ttl bool, defaultTTL int, count int, pattern string, parallel int) *Redis {
	return &Redis{
		Pool:       source,
		Bus:        bus,
		Silent:     silent,
		TTL:        ttl,
		DefaultTTL: defaultTTL,
		Count:      count,
		Pattern:    pattern,
		Parallel:   parallel,
	}
}

// maybeLog may log, depending on the Silent flag
func (r *Redis) maybeLog(s string) {
	if r.Silent {
		return
	}
	fmt.Print(s)
}

// maybeTTL may sync the TTL, depending on the TTL flag
func (r *Redis) maybeTTL(key string) (string, error) {
	// noop if TTL is disabled, speeds up sync process
	if !r.TTL {
		return strconv.FormatInt(int64(r.DefaultTTL), 10), nil
	}

	var ttl string

	// Try getting key TTL.
	err := r.Pool.Do(radix.Cmd(&ttl, "PTTL", key))
	if err != nil {
		return ttl, err
	}

	// When key has no expire PTTL returns "-1".
	// We set it to 0, default for no expiration time.
	if ttl == "-1" {
		ttl = "0"
	}

	return ttl, nil
}

// Read gently scans an entire Redis DB for keys, then dumps
// the key/value pair (Payload) on the message Bus channel.
// It leverages implicit pipelining to speedup large DB reads.
// To be used in an ErrGroup.
func (r *Redis) Read(ctx context.Context) error {
	defer close(r.Bus)

	scanner := radix.NewScanner(r.Pool, r.scanOption())
	defer scanner.Close()

	var key string

	// Create read workers
	g, gctx := errgroup.WithContext(ctx)

	// Create shared message bus
	r.ReadBus = make(message.Bus, 100)

	for i := 0; i < r.Parallel; i++ {
		g.Go(func() error {
			return r.ReadKey(gctx)
		})
	}

	// Scan and push to bus until no keys are left.
	// If context Done, exit early.
	for scanner.Next(&key) {
		select {
		// Exit early if context done.
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("redis read master: exit")
			return ctx.Err()
		case r.ReadBus <- message.Payload{Key: key}:
			// Do we need log here?
		}
	}

	close(r.ReadBus)

	// Block and wait for goroutines
	err := g.Wait()
	if err != nil && err != context.Canceled {
		return err
	} else {
		fmt.Println("done read")
	}

	return nil
}

// ReadKey dump the key/value from redis and send to bus
func (r *Redis) ReadKey(ctx context.Context) error {
	for r.ReadBus != nil {
		select {
		// Exit early if context done.
		case <-ctx.Done():
			fmt.Println("")
			fmt.Println("redis worker: exit")
			return ctx.Err()
			// Get Messages from Bus
		case p, ok := <-r.ReadBus:
			// if channel closed, set to nil, break loop
			if !ok {
				r.ReadBus = nil
				break
			}

			var value string
			var ttl string
			key := p.Key

			err := r.Pool.Do(radix.Cmd(&value, "DUMP", key))
			if err != nil {
				return err
			}

			ttl, err = r.maybeTTL(key)
			if err != nil {
				return err
			}

			if ttl == "-2" {
				// expired key
				continue
			}

			select {
			case <-ctx.Done():
				fmt.Println("")
				fmt.Println("redis read: exit")
				return ctx.Err()
			case r.Bus <- message.Payload{Key: key, Value: value, TTL: ttl}:
				r.maybeLog("r")
			}
		}
	}

	return nil
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
				break
			}
			err := r.Pool.Do(radix.Cmd(nil, "RESTORE", p.Key, p.TTL, p.Value, "REPLACE"))
			if err != nil {
				return err
			}
			r.maybeLog("w")
		}
	}

	return nil
}

func (r *Redis) scanOption() radix.ScanOpts {
	return radix.ScanOpts{
		Command: "SCAN",
		Pattern: r.Pattern,
		Count:   r.Count,
	}
}
