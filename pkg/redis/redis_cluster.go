// Package redis allows reading/writing from/to a Redis DB.
package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/pkg/errors"

	"github.com/stickermule/rump/pkg/config"
	"github.com/stickermule/rump/pkg/message"
)

// Redis holds references to a DB pool and a shared message bus.
// Silent disables verbose mode.
// TTL enables TTL sync.
type RedisCluster struct {
	Cluster *radix.Cluster
	Bus     message.Bus
	Silent  bool
	TTL     bool
}

// New creates the Redis struct, used to read/write.
func NewCluster(addr string, cfg config.Config, bus message.Bus) (*RedisCluster, error) {

	// this cluster will use the ClientFunc to create a pool to each node in the
	// cluster.
	poolFunc := func(network, saddr string) (radix.Client, error) {
		return radix.NewPool(network, addr, 100, radix.PoolConnFunc(authConn(cfg.Source)))
	}

	vanillaCluster, err := radix.NewCluster([]string{addr}, radix.ClusterPoolFunc(poolFunc))

	if err != nil {
		log.Fatalf("Error preparing for benchmark, while creating new connection. error = %v", err)
	}

	// do a PING and make sure you are connected
	var pong string
	if err := vanillaCluster.Do(radix.Cmd(&pong, "PING")); err != nil {
		return nil, errors.Wrap(err, "error in checking source connectivity")
	}

	// Issue CLUSTER SLOTS command
	err = vanillaCluster.Sync()
	if err != nil {
		log.Fatalf("Error preparing for benchmark, while issuing CLUSTER SLOTS. error = %v", err)
	}

	return &RedisCluster{
		Cluster: vanillaCluster,
		Bus:     bus,
		Silent:  cfg.Silent,
		TTL:     cfg.TTL,
	}, nil
}

// maybeLog may log, depending on the Silent flag
func (r *RedisCluster) maybeLog(s string) {
	if r.Silent {
		return
	}
	fmt.Print(s)
}

// maybeTTL may sync the TTL, depending on the TTL flag
func (r *RedisCluster) maybeTTL(key string) (string, error) {
	// noop if TTL is disabled, speeds up sync process
	if !r.TTL {
		return "0", nil
	}

	var ttl string
	var err error

	// Try getting key TTL.
	err = r.Cluster.Do(radix.Cmd(&ttl, "PTTL", key))

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
func (r *RedisCluster) Read(ctx context.Context) error {
	defer close(r.Bus)

	scanner := r.Cluster.NewScanner(radix.ScanAllKeys)

	var key string
	var value string
	var ttl string

	// Scan and push to bus until no keys are left.
	// If context Done, exit early.
	for scanner.Next(&key) {

		err := r.Cluster.Do(radix.Cmd(&value, "DUMP", key))
		if err != nil {
			return err
		}

		ttl, err = r.maybeTTL(key)
		if err != nil {
			return err
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

	return scanner.Close()
}

// Write restores keys on the db as they come on the message bus.
func (r *RedisCluster) Write(ctx context.Context) error {
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
			err := r.Cluster.Do(radix.Cmd(nil, "RESTORE", p.Key, p.TTL, p.Value, "REPLACE"))
			if err != nil {
				return err
			}
			r.maybeLog("w")
		}
	}

	return nil
}

func authConn(u config.Resource) radix.ConnFunc {

	p, _ := u.User.Password()

	if u.IsSecure() {
		return func(network, address string) (radix.Conn, error) {
			return radix.Dial(network, address,
				radix.DialTimeout(2*time.Minute),
				radix.DialAuthPass(p),
				radix.DialUseTLS(&tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				}),
			)
		}
	}

	return func(network, address string) (radix.Conn, error) {
		return radix.Dial(network, address,
			radix.DialTimeout(1*time.Minute),
			radix.DialAuthPass(p),
		)
	}
}
