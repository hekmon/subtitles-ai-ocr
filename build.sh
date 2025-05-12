#!/usr/bin/env bash

set -e

version=$(git describe --tags --always --dirty)

crosscompile () {
    if [ "$1" == "windows" ]; then
        name='pgs-ai-ocr.exe'
    else
        name='pgs-ai-ocr'
    fi
    GOOS="$1" GOARCH="$2" go build -ldflags="-s -w -X 'main.Version=${version}'" -o "$name"
    zip -9 "pgs-ai-ocr_${version}_${1}_${2}.zip" "$name"
}

echo '* Compiling for Windows'
crosscompile 'windows' 'amd64'
echo
echo '* Compiling for MacOS Intel'
crosscompile 'darwin' 'amd64'
echo
echo '* Compiling for MacOS Apple Silicon'
crosscompile 'darwin' 'arm64'
echo
echo '* Compiling for Linux'
crosscompile 'linux' 'amd64'
echo
echo '* Cleaning up'
test -f pgs-ai-ocr && rm pgs-ai-ocr
test -f pgs-ai-ocr.exe && rm pgs-ai-ocr.exe
