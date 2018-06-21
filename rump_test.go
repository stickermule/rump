package main

import (
	"testing"
	"github.com/ory/dockertest"
	"fmt"
	"log"
	"github.com/gomodule/redigo/redis"
	"os"
	"github.com/stretchr/testify/assert"
)

var testRedises = []string{}

func getRedisURL(serverPort string) string {
	return fmt.Sprintf("redis://%s", serverPort)
}

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker service: %s", err)
	}
	var resources []*dockertest.Resource
	// pulls an image, creates a container based on it and runs it
	for i := 1; i <= 2; i++ {
		resource, err := pool.Run("redis", "3.2", []string{})
		resources = append(resources, resource)
		if err != nil {
			log.Fatalf("Could not start redis resource: %s", err)
		}

		// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
		if err := pool.Retry(func() error {
			var err error
			hostPort := fmt.Sprintf("localhost:%s", resource.GetPort("6379/tcp"))
			c, err := redis.Dial("tcp", hostPort)
			if err != nil {
				return err
			}
			_, err = redis.String(c.Do("PING"))
			handle(err)
			testRedises = append(testRedises, hostPort)
			return nil
		}); err != nil {
			log.Fatalf("Could not connect to docker: %s", err)
		}
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	for _, resource := range resources {

		if err := pool.Purge(resource); err != nil {
			log.Fatalf("Could not purge resource: %s", err)
		}
	}
	os.Exit(code)
}

func cleanRedis(redisURL string) {
	c, _ := redis.DialURL(redisURL)
	_, _ = c.Do("FLUSHALL")
	c.Close()
}

func TestConnection(t *testing.T) {

	// Optionally set some keys your code expects:
	cleanRedis(getRedisURL(testRedises[0]))
	key1, value1 := "foo", "bar"

	// Run your code and see if it behaves.
	c, _ := redis.Dial("tcp", testRedises[0])
	_, _ = c.Do("SET", "foo", "bar")
	got, _ := redis.String(c.Do("GET", key1))
	assert.Equal(t, value1, got, key1+"has the wrong value: exp")

	_, _ = redis.String(c.Do("EXPIRE", key1, "0"))
	got, _ = redis.String(c.Do("GET", key1))
	assert.NotEqual(t, value1, got, key1+"has the wrong value: exp")
}

// Sync functionality test
func TestSync(t *testing.T) {
	var (
		err error
		got string
	)

	fromAddr := getRedisURL(testRedises[0])
	toAddr := getRedisURL(testRedises[1])
	cleanRedis(fromAddr)
	cleanRedis(toAddr)

	// Verify values at source server
	fromConn, _ := redis.DialURL(fromAddr)
	defer fromConn.Close()

	toConn, _ := redis.DialURL(toAddr)
	defer toConn.Close()

	// Set values at server
	key1, value1 := "foo", "bar"
	_, _ = redis.String(fromConn.Do("SET", key1, value1))

	key2, hKey20, hValue20 := "some", "other", "value"
	hKey21, hValue21 := "another", "value"
	_, _ = redis.String(fromConn.Do("HSET", key2, hKey20, hValue20))
	_, _ = redis.String(fromConn.Do("HSET", key2, hKey21, hValue21))

	// Check dump command
	got, err = redis.String(fromConn.Do("DUMP", key2))
	handle(err)
	assert.NotEmpty(t, got, "Dump output shouldn't be null")

	// Verify values at server

	got, _ = redis.String(fromConn.Do("GET", key1))
	assert.Equal(t, value1, got, key1+"has the wrong value: exp")

	got, _ = redis.String(fromConn.Do("HGET", key2, hKey20))
	assert.Equal(t, hValue20, got, hValue20+"has the wrong value: exp")

	// Verify values are not existing in the new server, before syncing
	got, _ = redis.String(toConn.Do("GET", key1))
	assert.NotEqual(t, value1, got, key1+"has the wrong value: exp")

	got, _ = redis.String(toConn.Do("HGET", key2, hKey20))
	assert.NotEqual(t, hValue20, got, hValue20+"has the wrong value: exp")

	// Verify values are existing in the new server, after syncing
	Sync(fromAddr, toAddr)
	got, _ = redis.String(toConn.Do("GET", key1))
	assert.Equal(t, value1, got, key1+"has the wrong value: exp")

	got, _ = redis.String(toConn.Do("HGET", key2, hKey20))
	assert.Equal(t, hValue20, got, hValue20+"has the wrong value: exp")

}
