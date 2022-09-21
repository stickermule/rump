// Package config parse and validates command flags.
package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

// Resource can be either Redis (isRedis) or file.
// URI is either a Redis URI or a file path.
type Resource struct {
	url.URL
}

func (r Resource) IsRedis() bool {
	return contains([]string{"redis", "rediss", "credis", "crediss"}, r.Scheme)
}

func (r Resource) IsSecure() bool {
	return r.Scheme == "rediss" || r.Scheme == "crediss"
}

func (r Resource) IsCluster() bool {
	return r.Scheme == "credis" || r.Scheme == "crediss"
}

func (r Resource) FormattedString() string {
	return fmt.Sprintf("redis://%v", r.Host)
}

// Config represents the current source and target config.
// Source and target are Resources.
// Silent disables verbose mode.
// TTL enables keys TTL sync.
type Config struct {
	Source Resource
	Target Resource
	Silent bool
	TTL    bool
}

// exit will exit and print the usage.
// Used in case of errors during flags parse/validate.
func exit(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	os.Exit(1)
}

// https://play.golang.org/p/Qg_uv_inCek
// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// validate makes sure from and to are Redis URIs or file paths,
// and generates the final Config.
func validate(from, to string, silent, ttl bool) (Config, error) {

	source, err := url.Parse(from)
	if err != nil {
		return Config{}, err
	}

	target, err := url.Parse(to)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Source: Resource{
			*source,
		},
		Target: Resource{
			*target,
		},
		Silent: silent,
		TTL:    ttl,
	}

	switch {
	case cfg.Source.String() == "":
		return cfg, fmt.Errorf("source not valid redis url")
	case cfg.Target.String() == "":
		return cfg, fmt.Errorf("target not valid redis url")
	case !cfg.Source.IsRedis() && !cfg.Target.IsRedis():
		return cfg, fmt.Errorf("file-only operations not supported")
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

	flag.Parse()

	cfg, err := validate(*from, *to, *silent, *ttl)
	if err != nil {
		// we exit here instead of returning so that we can show
		// the usage examples in case of an error.
		exit(err)
	}

	return cfg
}
