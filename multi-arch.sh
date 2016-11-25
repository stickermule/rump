#!/usr/bin/env bash
VERSION="$1"

GOOS=darwin GOARCH=amd64 go build -o rump-$VERSION-darwin-amd64 rump.go 
GOOS=linux GOARCH=amd64 go build -o rump-$VERSION-linux-amd64 rump.go 
