#!/bin/bash

# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e
set -o pipefail

if [ -z "${OS:-$(go env GOOS)}" ]; then
    echo "OS must be set"
    exit 1
fi
if [ -z "${ARCH:-$(go env GOARCH)}" ]; then
    echo "ARCH must be set"
    exit 1
fi

VERSION="$(grep "Version = " pkg/version/version.go | \
  cut -d '=' -f 2                                   | \
  tr -d '"'                                         | \
  tr -d ' ')"
if [ -z "${VERSION:-}" ]; then
    echo "VERSION must be set"
    exit 1
fi

export CGO_ENABLED=0
export GOARCH="${ARCH}"
export GOOS="${OS}"
export GO111MODULE=on
export GOFLAGS="-mod=vendor"

# Create a file called `i0xen` in the current build directory.

go mod tidy
go mod vendor

gofmt -w cmd pkg

GOBIN="$(pwd)" go install   \
    -installsuffix "static" \
    -ldflags "-s -w"        \
    ./...

# Compress the binary if possible.

if hash upx &>/dev/null; then
  if [[ "${UPX_ENABLE}" == [yY]* ]]; then
    upx --best i0xen
  fi
fi
