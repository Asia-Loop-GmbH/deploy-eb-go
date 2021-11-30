#!/bin/sh

rm -rfv build/*
mkdir build/linux
mkdir build/windows
mkdir build/darwin

env GOOS=linux GOARCH=amd64 go build -o build/linux/
env GOOS=windows GOARCH=amd64 go build -o build/windows/
env GOOS=darwin GOARCH=amd64 go build -o build/darwin/
