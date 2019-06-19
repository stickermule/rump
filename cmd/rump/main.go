package main

import (
	"github.com/stickermule/rump/pkg/config"
	"github.com/stickermule/rump/pkg/run"
)

func main() {
	// parse config flags, will exit in case of errors.
	cfg := config.Parse()

	run.Run(cfg)
}
