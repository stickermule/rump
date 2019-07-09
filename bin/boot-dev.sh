#!/usr/bin/env sh

# get dependencies, build cmds
go build ./...

# autowatch and run tests
ls -d * */* */*/* | entr -r go test $(go list ./...)
