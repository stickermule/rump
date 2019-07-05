#!/usr/bin/env sh
VERSION="$1"

rm rump-*
GOOS=darwin GOARCH=amd64 go build -o rump-$VERSION-darwin-amd64 cmd/rump/main.go
GOOS=linux GOARCH=amd64 go build -o rump-$VERSION-linux-amd64 cmd/rump/main.go
GOOS=linux GOARCH=arm go build -o rump-$VERSION-linux-arm cmd/rump/main.go
GOOS=windows GOARCH=amd64 go build -o rump-$VERSION-windows-amd64 cmd/rump/main.go
