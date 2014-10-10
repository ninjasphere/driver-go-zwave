#!/usr/bin/env bash
set -ex

OWNER=ninjasphere
BIN_NAME=driver-go-zwave
PROJECT_NAME=driver-go-zwave


# Get the parent directory of where this script is.
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

GIT_COMMIT="$(git rev-parse HEAD)"
GIT_DIRTY="$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)"
VERSION="$(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' )"

# remove working build
# rm -rf .gopath
if [ ! -d ".gopath" ]; then
	mkdir -p .gopath/src/github.com/${OWNER}
	ln -sf ../../../.. .gopath/src/github.com/${OWNER}/${PROJECT_NAME}
fi

export GOPATH="$(pwd)/.gopath"

if [ ! -d $GOPATH/src/github.com/ninjasphere/go-ninja ]; then
	# Clone our internal commons package
	git clone git@github.com:ninjasphere/go-ninja.git $GOPATH/src/github.com/ninjasphere/go-ninja
fi

if [ ! -d $GOPATH/src/github.com/ninjasphere/go-openzwave ]; then
	git clone git@github.com:ninjasphere/go-openzwave.git $GOPATH/src/github.com/ninjasphere/go-openzwave
fi
( cd $GOPATH/src/github.com/ninjasphere/go-openzwave; make deps )

# move the working path and build
cd .gopath/src/github.com/${OWNER}/${PROJECT_NAME} &&
go get -d -v ./... &&
make deps && (
if [ "$BUILDBOX_BRANCH" = "master" ]; then
	go build -ldflags "-X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY} -extldflags -L${GOPATH}/src/github.com/${OWNER}/go-openzwave/openzwave" -tags release -o ./bin/${BIN_NAME}
else
	go build -ldflags "-X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY} -extldflags -L${GOPATH}/src/github.com/${OWNER}/go-openzwave/openzwave" -o ./bin/${BIN_NAME}
fi
) 

# building the master branch on ci
