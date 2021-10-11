// Package config parse and validates command flags.
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Resource can be either Redis (isRedis) or file.
// URI is either a Redis URI or a file path.
type Resource struct {
	URI     string
	IsRedis bool
	Auth    string
}

// Config represents the current source and target config.
// Source and target are Resources.
// Silent disables verbose mode.
// TTL enables keys TTL sync.
type Config struct {
	Source     Resource
	Target     Resource
	Silent     bool
	TTL        bool
	DefaultTTL int
	Count      int
	Pattern    string
}

// exit will exit and print the usage.
// Used in case of errors during flags parse/validate.
func exit(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	os.Exit(1)
}

// getRedisOptions makes sure provided string is Redis URI,
// and return is redis or not and auth string if it's specified.
func getRedisOptions(conn string) (bool, string, string) {
	var redisPrefix string = "redis://"
	var redisTlsPrefix string = "rediss://"
	var authSeparator string = "@"
	var isRedis bool
	var auth string
	var uri string = conn

	if strings.HasPrefix(conn, redisPrefix) {
		isRedis = true
	} else if strings.HasPrefix(conn, redisTlsPrefix) {
		isRedis = true

		if strings.Contains(conn, authSeparator) {
			auth = strings.Split(strings.TrimPrefix(conn, redisTlsPrefix), authSeparator)[0]
			uri = redisPrefix + strings.Split(conn, authSeparator)[1]
		}
	}

	return isRedis, uri, auth
}

// validate makes sure from and to are Redis URIs or file paths,
// and generates the final Config.
func validate(from, to string, silent, ttl bool, defaultTTL int, count int, pattern string) (Config, error) {
	cfg := Config{
		Source:     Resource{},
		Target:     Resource{},
		Silent:     silent,
		TTL:        ttl,
		DefaultTTL: defaultTTL,
		Count:      count,
	}

	cfg.Source.IsRedis, cfg.Source.URI, cfg.Source.Auth = getRedisOptions(from)
	cfg.Target.IsRedis, cfg.Target.URI, cfg.Target.Auth = getRedisOptions(to)

	// Guard from incorrect usage.
	switch {
	case cfg.Source.URI == "":
		return cfg, fmt.Errorf("from is required")
	case cfg.Target.URI == "":
		return cfg, fmt.Errorf("to is required")
	case !cfg.Source.IsRedis && !cfg.Target.IsRedis:
		return cfg, fmt.Errorf("file-only operations not supported")
	case cfg.TTL && cfg.DefaultTTL != 0:
		return cfg, fmt.Errorf("default ttl can only set if ttl mode is disabled")
	}

	return cfg, nil
}

// Parse parses the command line flags and returns a Config.
func Parse() Config {
	example := "example: redis://127.0.0.1:6379/0 or /tmp/dump.rump"
	from := flag.String("from", "", example)
	to := flag.String("to", "", example)
	silent := flag.Bool("silent", false, "optional, no verbose output")
	ttl := flag.Bool("ttl", false, "optional, enable ttl sync")
	defaultTTL := flag.Int("default-ttl", 0, "optional, default ttl if ttl sync mode is disabled")
	count := flag.Int("count", 10, "optional, number of record per seconds")
	pattern := flag.String("pattern", "*", "optional, pattern to scan")

	flag.Parse()

	cfg, err := validate(*from, *to, *silent, *ttl, *defaultTTL, *count, *pattern)
	if err != nil {
		// we exit here instead of returning so that we can show
		// the usage examples in case of an error.
		exit(err)
	}

	return cfg
}
