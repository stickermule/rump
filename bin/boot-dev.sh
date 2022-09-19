#!/usr/bin/env sh

# get dependencies, build cmds
go mod download
go build ./...

# autowatch and run tests
ls -d * */* */*/* | entr -n -r go test $(go list ./...)
