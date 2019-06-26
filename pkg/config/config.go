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
// Silent mode will disable read/write logs.
type Resource struct {
	URI     string
	IsRedis bool
	Silent bool
}

// Config represents the current source and target config.
// Source and target are Resources.
type Config struct {
	Source Resource
	Target Resource
}

// exit will exit and print the usage.
// Used in case of errors during flags parse/validate.
func exit(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	os.Exit(1)
}

// validate makes sure from and to are Redis URIs or file paths,
// and generates the final Config.
func validate(from, to string, silent bool) (Config, error) {
	cfg := Config{
		Source: Resource{
			URI: from,
			Silent: silent,
		},
		Target: Resource{
			URI: to,
			Silent: silent,
		},
	}

	if strings.HasPrefix(from, "redis://") {
		cfg.Source.IsRedis = true
	}

	if strings.HasPrefix(to, "redis://") {
		cfg.Target.IsRedis = true
	}

	// Guard from incorrect usage.
	switch {
	case cfg.Source.URI == "":
		return cfg, fmt.Errorf("from is required")
	case cfg.Target.URI == "":
		return cfg, fmt.Errorf("to is required")
	case !cfg.Source.IsRedis && !cfg.Target.IsRedis:
		return cfg, fmt.Errorf("file-only operations not supported")
	}

	return cfg, nil
}

// Parse parses the command line flags and returns a Config.
func Parse() Config {
	example := "example: redis://127.0.0.1:6379/0 or /tmp/dump.rump"
	from := flag.String("from", "", example)
	to := flag.String("to", "", example)
	silent := flag.Bool("silent", false, "example: -silent")

	flag.Parse()

	cfg, err := validate(*from, *to, *silent)
	if err != nil {
		// we exit here instead of returning so that we can show
		// the usage examples in case of an error.
		exit(err)
	}

	return cfg
}
