#   Copyright (c) 2023 KylinSoft Co., Ltd.
#   Kylin trusted image builder(ktib) is licensed under Mulan PSL v2.
#   You can use this software according to the terms and conditions of the Mulan PSL v2.
#   You may obtain a copy of Mulan PSL v2 at:
#            http://license.coscl.org.cn/MulanPSL2
#   THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING
#   BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
#   See the Mulan PSL v2 for more details.

VERSION := $(shell git describe --tags --always)
LDFLAGS := -ldflags "-s -w -X=main.version=$(VERSION)"
BUILDTAGS := seccomp


GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOSRC := $(GOPATH)/src

u := $(if $(update),-u)

.PHONY: deps
deps:
	go get ${u} -d
	go mod tidy

.PHONY: build
build:
	go build $(LDFLAGS) -tags "$(BUILDTAGS)" ./cmd/ktib

.PHONY: install
install:
	go install $(LDFLAGS) ./cmd/ktib
