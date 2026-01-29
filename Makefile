#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-s -w -X=main.version=$(VERSION)"
BUILDTAGS := seccomp containers_image_openpgp exclude_graphdriver_devicemapper exclude_graphdriver_btrfs

GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOSRC := $(GOPATH)/src

u := $(if $(update),-u)

.PHONY: deps
deps:
	go mod tidy

.PHONY: build
build:
	CGO_ENABLED=0 go build $(LDFLAGS) -tags "$(BUILDTAGS)" ./cmd/ktib

.PHONY: install
install:
	go install $(LDFLAGS) ./cmd/ktib

.PHONY: test
test:
	CGO_ENABLED=0 go test -v -tags "containers_image_openpgp exclude_graphdriver_devicemapper exclude_graphdriver_btrfs" ./...

.PHONY: clean
clean:
	rm -f ktib
	go clean