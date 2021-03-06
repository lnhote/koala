#!/bin/bash

if [ ! -d /tmp/build-golang/src/github.com/v2pro ]; then
    mkdir -p /tmp/build-golang/src/github.com/v2pro
    ln -s $PWD /tmp/build-golang/src/github.com/v2pro/koala
fi
export GOPATH=/tmp/build-golang
rm -rf output
mkdir output
echo "compile output/koala-replayer.so"
go get github.com/Masterminds/glide
cd /tmp/build-golang/src/github.com/v2pro/koala
/tmp/build-golang/bin/glide i
go build -tags="koala_replayer" -buildmode=c-shared -o output/koala-replayer.so github.com/v2pro/koala/cmd/replayer
