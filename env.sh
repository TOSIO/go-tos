#!/bin/sh
export GOROOT=/usr/local/go
export GOPATH=$PWD
cd src/github.com/tos
ln -s ../../../ go-tos
echo "go get github.com/mattn/go-colorable"
go get github.com/mattn/go-colorable
echo "go get github.com/aristanetworks/goarista/monotime"
go get github.com/aristanetworks/goarista/monotime
echo "go get github.com/go-stack/stack"
go get github.com/go-stack/stack
echo "go get is over."

