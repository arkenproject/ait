#!/bin/bash

platforms=("linux/amd64" "linux/arm" "linux/arm64" "darwin/amd64")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags "-X github.com/arken/ait/apis/github.clientID=$1 -X github.com/arken/ait/cli.appVersion=$2" -o ait-$2-${GOOS}-${GOARCH} .

done