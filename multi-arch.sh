#!/usr/bin/env bash
VERSION="$1"

rm rump-*
GOOS=darwin GOARCH=amd64 go build -o rump-$VERSION-darwin-amd64 rump.go
GOOS=linux GOARCH=amd64 go build -o rump-$VERSION-linux-amd64 rump.go
GOOS=linux GOARCH=arm go build -o rump-$VERSION-linux-arm rump.go
GOOS=windows GOARCH=amd64 go build -o rump-$VERSION-windows-amd64 rump.go
