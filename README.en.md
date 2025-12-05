# ktib

Kylin Trusted Image Builder (ktib) is a CLI tool for base image building and compliance. It supports project initialization, rootfs build/cleanup, image build, one-shot base image build via the make subcommand, Dockerfile static auditing, and image/builder operations.

## Features
- Project management: `ktib project` to init project, generate default config, build/clean rootfs, and build images
- One-shot build: `ktib make` to init → generate config → build/clean rootfs → build image
- Compliance scanning: `ktib scan dockerfile-audit` for Dockerfile static compliance audit
- Image operations: `ktib images` list/login/logout/pull/push/save/load/tag/inspect/manifest
- Builder operations: `ktib builders` add/build/copy/commit/from/label/list/mount/run/rm/umount

## Supported Architectures
- `x86_64`
- `aarch64`

## Requirements
- Linux with CGO enabled (Windows native build is not supported)
- Container toolchain: `buildah`/`podman`, `containers-common`
- `seccomp` support is recommended (build with `-tags "seccomp"`)
- Default policy for Dockerfile audit: `/etc/ktib/policy.yaml`

## Installation
Using Make:

```bash
make build
make install
```

From source:

```bash
go build -ldflags "-s -w -X=main.version=vX.Y.Z" -tags "seccomp" ./cmd/ktib
```

## Quick Start
Initialize and build:

```bash
# Init project skeleton
ktib project init --type minimal /path/to/project

# Generate default config
ktib project default_config --type minimal > /path/to/project/config.yml

# Build and clean rootfs
ktib project build-rootfs --config /path/to/project/config.yml /path/to/project
ktib project clean-rootfs --type minimal /path/to/project

# Build image
ktib project build --name myimage --tag v1 /path/to/project

# One-shot build
ktib make --init --type minimal --name myimage --tag v1 /path/to/project
```

Audit and image operations:

```bash
# Dockerfile audit
ktib scan dockerfile-audit --dockerfile ./Dockerfile --output audit.json

# Common image operations
ktib images list --json
ktib images pull docker.io/library/alpine:latest
ktib images login docker.io --username myuser --password-stdin <<<"mypassword"
ktib images push myimage:v1
ktib images save myimage:v1 -o myimage.tar
ktib images load -i myimage.tar
ktib images tag myimage:v1 myimage:latest
ktib images inspect myimage:latest
ktib images manifest myimage:latest
```

Builder operations examples:

```bash
# Create a builder from base image
ktib builders from --name mybuilder docker.io/library/alpine:latest

# Build inside the builder
ktib builders build -f Dockerfile -t myimage:v1
```

## Command Docs
- Project: `docs/commands/project.md`
- Images: `docs/commands/images.md`
- Builders: `docs/commands/builders.md`
- Scan: `docs/commands/scan.md`

## Shell Completion
Generate completion for popular shells:

```bash
# Bash
ktib completion bash > /etc/bash_completion.d/ktib

# Zsh
ktib completion zsh > "${fpath[1]}/_ktib"

# Fish
ktib completion fish > ~/.config/fish/completions/ktib.fish

# PowerShell
ktib completion powershell > ktib.ps1
```

## Logging and Version
- Global flags: `--log-level` (trace|debug|info|warn|error|fatal|panic), `--log-format` (text|json)
- Version: `ktib version`

## License
[Mulan PSL v2](http://license.coscl.org.cn/MulanPSL2)

## Contribution
1. Fork this repository
2. Create a topic branch (e.g., `feat/xxx`)
3. Commit with tests when applicable
4. Open a Pull Request with detailed description
