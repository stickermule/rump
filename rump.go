package main

import (
	"os"
	"fmt"
	"flag"
	"github.com/garyburd/redigo/redis"
)

// Report all errors to stdout.
func handle(err error) {
	if err != nil && err != redis.ErrNil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Scan and queue source keys.
func get(conn redis.Conn, queue chan<- map[string]string, size int64) {
	var (
		cursor int64
		keys []string
	)

	for {
		// Scan a batch of keys.
		values, err := redis.Values(conn.Do("SCAN", cursor, "COUNT", size))
		handle(err)
		values, err = redis.Scan(values, &cursor, &keys)
		handle(err)

		fmt.Printf("scaned keys %d\n", len(keys))
		// Get pipelined dumps.
		for _, key := range keys {
			conn.Send("DUMP", key)
		}
		dumps, err := redis.Strings(conn.Do(""))
		handle(err)

		// Build batch map.
		batch := make(map[string]string)
		for i, _ := range keys {
			batch[keys[i]] = dumps[i]
		}

		// Last iteration of scan.
		if cursor == 0 {
			// queue last batch.
			select {
			case queue <- batch:
			}
			close(queue)
			break
		}

		//fmt.Printf(">")
		// queue current batch.
		queue <- batch
	}
}

// Restore a batch of keys on destination.
func put(conn redis.Conn, queue <-chan map[string]string) {
	for batch := range queue {
		for key, value := range batch {
			conn.Send("RESTORE", key, "0", value)
		}
		_, err := conn.Do("")
		handle(err)

		//fmt.Printf(".")
	}
}

func main() {
	from := flag.String("from", "", "example: redis://127.0.0.1:6379/0")
	fromPwd := flag.String("fromPwd", "", "from redis password")
	to := flag.String("to", "", "example: redis://127.0.0.1:6379/1")
	toPwd := flag.String("toPwd", "", "to redis password")
	size := flag.Int64("size", 10, "scan size")
	flag.Parse()

	source, err := redis.DialURL(*from, redis.DialPassword(*fromPwd))
	handle(err)
	destination, err := redis.DialURL(*to, redis.DialPassword(*toPwd))
	handle(err)
	defer source.Close()
	defer destination.Close()

	// Channel where batches of keys will pass.
	queue := make(chan map[string]string, 100)

	// Scan and send to queue.
	go get(source, queue, *size)

	// Restore keys as they come into queue.
	put(destination, queue)

	fmt.Println("Sync done.")
}
