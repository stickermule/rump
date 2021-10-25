// Package run manages Read, Write and Signal goroutines.
package run

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/mediocregopher/radix/v3"
	"golang.org/x/sync/errgroup"

	"github.com/stickermule/rump/pkg/config"
	"github.com/stickermule/rump/pkg/file"
	"github.com/stickermule/rump/pkg/message"
	"github.com/stickermule/rump/pkg/redis"
	"github.com/stickermule/rump/pkg/signal"
)

// Exit helper
func exit(e error) {
	fmt.Println(e)
	os.Exit(1)
}

func authConn(authPass string, useTls bool) radix.ConnFunc {
	if useTls {
		return func(network, address string) (radix.Conn, error) {
			return radix.Dial(network, address,
				radix.DialTimeout(1*time.Minute),
				radix.DialAuthPass(authPass),
				radix.DialUseTLS(&tls.Config{}),
			)
		}
	} else {
		return func(network, address string) (radix.Conn, error) {
			return radix.Dial(network, address,
				radix.DialTimeout(1*time.Minute),
				radix.DialAuthPass(authPass),
			)
		}
	}
}

// Run orchestrate the Reader, Writer and Signal handler.
func Run(cfg config.Config) {
	// create ErrGroup to manage goroutines
	ctx, cancel := context.WithCancel(context.Background())
	g, gctx := errgroup.WithContext(ctx)

	// Start signal handling goroutine
	g.Go(func() error {
		return signal.Run(gctx, cancel)
	})

	// Create shared message bus
	ch := make(message.Bus, 100)

	// Create and run either a Redis or File Source reader.
	if cfg.Source.IsRedis {
		var db *radix.Pool
		var err error

		if len(cfg.Source.Auth) > 0 {
			db, err = radix.NewPool("tcp", cfg.Source.URI, 1, radix.PoolConnFunc(authConn(cfg.Source.Auth, cfg.Source.TLS)))

			if err != nil {
				exit(err)
			}
		} else {
			db, err = radix.NewPool("tcp", cfg.Source.URI, 1)

			if err != nil {
				exit(err)
			}
		}

		source := redis.New(db, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			return source.Read(gctx)
		})
	} else {
		source := file.New(cfg.Source.URI, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			return source.Read(gctx)
		})
	}

	// Create and run either a Redis or File Target writer.
	if cfg.Target.IsRedis {
		var db *radix.Pool
		var err error

		if len(cfg.Target.Auth) > 0 {
			db, err = radix.NewPool("tcp", cfg.Target.URI, 1, radix.PoolConnFunc(authConn(cfg.Target.Auth, cfg.Target.TLS)))

			if err != nil {
				exit(err)
			}
		} else {
			db, err = radix.NewPool("tcp", cfg.Target.URI, 1)

			if err != nil {
				exit(err)
			}
		}

		target := redis.New(db, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			defer cancel()
			return target.Write(gctx)
		})
	} else {
		target := file.New(cfg.Target.URI, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			defer cancel()
			return target.Write(gctx)
		})
	}

	// Block and wait for goroutines
	err := g.Wait()
	if err != nil && err != context.Canceled {
		exit(err)
	} else {
		fmt.Println("done")
	}
}
