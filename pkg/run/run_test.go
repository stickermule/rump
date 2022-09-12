// Example to test the command line output.
// Expected Redis monitor output: redis-cli -h redis monitor
// OK
// "SELECT" "9"
// "SELECT" "10"
// "SET" "key1" "value1"
// "SELECT" "9"
// "SELECT" "10"
// "SCAN" "0"
// "DUMP" "key1"
//  "RESTORE" "key1" "0" "..." "REPLACE"
// "FLUSHDB"
//  "FLUSHDB"
package run_test

import (
	"fmt"
	"net/url"
	"os"

	"github.com/mediocregopher/radix/v3"

	"github.com/stickermule/rump/pkg/config"
	"github.com/stickermule/rump/pkg/run"
)

var db1 *radix.Pool
var db2 *radix.Pool
var path string

func setup() {
	db1, _ = radix.NewPool("tcp", "redis://redis:6379/9", 1)
	db2, _ = radix.NewPool("tcp", "redis://redis:6379/10", 1)
	path = "/app/dump.rump"

	// generate source test data on db1
	for i := 1; i <= 1; i++ {
		k := fmt.Sprintf("key%v", i)
		v := fmt.Sprintf("value%v", i)
		db1.Do(radix.Cmd(nil, "SET", k, v))
		db1.Do(radix.Cmd(nil, "PEXPIRE", k, "10000"))
	}
}

func teardown() {
	// Reset test dbs
	db1.Do(radix.Cmd(nil, "FLUSHDB"))
	db2.Do(radix.Cmd(nil, "FLUSHDB"))
	// Delete dump file
	os.Remove(path)
}

func ExampleRun_redisToRedis() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("redis://redis:6379/10")

	cfg := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
		Silent: false,
	}

	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
}

func ExampleRun_redisToRedisTTL() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("redis://redis:6379/10")

	cfg := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
		Silent: false,
		TTL:    true,
	}

	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
}

func ExampleRun_redisToRedisSilent() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("redis://redis:6379/10")

	cfg := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
		Silent: true,
	}

	run.Run(cfg)
	// Output:
	// signal: exit
	// done
}

func ExampleRun_redisToFile() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("/app/dump.rump")

	cfg := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
	}

	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
}

func ExampleRun_redisToFileTTL() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("/app/dump.rump")

	cfg := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
		TTL:    true,
	}

	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
}

func ExampleRun_fileToRedis() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("/app/dump.rump")

	cfgFileDump := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
	}
	run.Run(cfgFileDump)

	cfg := config.Config{
		Source: config.Resource{*to},
		Target: config.Resource{*from},
	}
	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
	// rw
	// signal: exit
	// done
}

func ExampleRun_fileToRedisTTL() {
	setup()
	defer teardown()

	from, _ := url.Parse("redis://redis:6379/9")
	to, _ := url.Parse("/app/dump.rump")

	cfgFileDump := config.Config{
		Source: config.Resource{*from},
		Target: config.Resource{*to},
	}
	run.Run(cfgFileDump)

	cfg := config.Config{
		Source: config.Resource{*to},
		Target: config.Resource{*from},
		TTL:    true,
	}
	run.Run(cfg)
	// Output:
	// rw
	// signal: exit
	// done
	// rw
	// signal: exit
	// done
}
