// Package run manages Read, Write and Signal goroutines.
package run

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/pkg/errors"
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
	if cfg.Source.IsRedis() {
		var db *radix.Cluster
		var err error

		if cfg.Source.IsSecure() {

			// p, _ := cfg.Source.User.Password()
			// err := db.Do(radix.Cmd(nil, "AUTH", p))
			// if err != nil {
			// 	log.Printf("redis error returned: %s", "wrong pass")
			// }

			poolFunc := func(network, addr string) (radix.Client, error) {
				return radix.NewPool(network, addr, 1, radix.PoolConnFunc(authConn(cfg.Source)))
			}

			db, err = radix.NewCluster([]string{cfg.Source.FormattedString()}, radix.ClusterPoolFunc(poolFunc))
		}

		if err != nil {
			exit(errors.Wrap(err, "error connecting to source"))
		}

		// do a PING and make sure you are connected
		if err := db.Do(radix.Cmd(nil, "PING")); err != nil {
			exit(err)
		}

		source := redis.New(db, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			if err := source.Read(gctx); err != nil {
				log.Fatal(err)
			}

			return err
		})
	} else {
		source := file.New(cfg.Source.String(), ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			return source.Read(gctx)
		})
	}

	// Create and run either a Redis or File Target writer.
	if cfg.Target.IsRedis() {
		var db *radix.Cluster
		var err error

		poolFunc := func(network, addr string) (radix.Client, error) {
			return radix.NewPool(network, addr, 1, radix.PoolConnFunc(authConn(cfg.Source)))
		}

		db, err = radix.NewCluster([]string{cfg.Target.FormattedString()}, radix.ClusterPoolFunc(poolFunc))

		if err != nil {
			exit(err)
		}

		target := redis.New(db, ch, cfg.Silent, cfg.TTL)

		g.Go(func() error {
			defer cancel()
			return target.Write(gctx)
		})
	} else {
		target := file.New(cfg.Target.String(), ch, cfg.Silent, cfg.TTL)

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

func authConn(u config.Resource) radix.ConnFunc {

	p, _ := u.User.Password()

	if u.IsSecure() {
		return func(network, address string) (radix.Conn, error) {
			return radix.Dial(network, address,
				radix.DialTimeout(1*time.Minute),
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
