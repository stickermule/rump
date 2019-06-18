package redis_test

import (
	"testing"
	"reflect"
	"os"
	"fmt"
	"github.com/mediocregopher/radix/v3"
	"github.com/stickermule/rump/pkg/message"
	"github.com/stickermule/rump/pkg/redis"
)

var db1 *radix.Pool
var db2 *radix.Pool
var ch message.Bus
var expected map[string]string

func setup() {
	db1, _ = radix.NewPool("tcp", "redis://redis:6379/3", 1)
	db2, _ = radix.NewPool("tcp", "redis://redis:6379/4", 1)
	ch = make(message.Bus, 100)
	expected = make(map[string]string)

	// generate source test data on db1
	for i := 1; i <= 20; i++ {
		k := fmt.Sprintf("key%v", i)
		v := fmt.Sprintf("value%v", i)
		db1.Do(radix.Cmd(nil, "SET", k, v))
		expected[k] = v
	}
}

func teardown() {
	// Reset test dbs
	db1.Do(radix.Cmd(nil, "FLUSHDB"))
	db2.Do(radix.Cmd(nil, "FLUSHDB"))
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

// Test reading all keys from db1 and then writing them to db2
func TestRead(t *testing.T) {
	source := redis.New(db1, ch)
	target := redis.New(db2, ch)

	// Read all keys from db1, push to shared message bus
	if err := source.Read(); err != nil {
		t.Error("error: ", err)
	}

	// Write all keys from message bus to db2
	if err := target.Write(); err != nil {
		t.Error("error: ", err)
	}

	// Get all db2 keys
	result := map[string]string{}
	var v string
	for k := range expected {
		db2.Do(radix.Cmd(&v, "GET", k))
		result[k] = v
	}

	// Compare db1 keys with db2 keys
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("expected: %v, result: %v", expected, result)
	}
}
