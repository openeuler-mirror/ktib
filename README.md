# ktib

Kylin Trusted Image Builder（ktib）是一款面向基础镜像构建与合规性的命令行工具，提供：项目初始化、rootfs 构建与清理、镜像构建、通过 make 子命令一站式构建基础镜像，以及 Dockerfile 静态合规审计、镜像与构建器操作等能力。

## 特性
- 项目管理：`ktib project` 初始化项目、生成默认配置、构建/清理 rootfs、构建镜像
- 一站式构建：`ktib make` 完成初始化 → 生成配置 → 构建/清理 rootfs → 构建镜像
- 合规扫描：`ktib scan dockerfile-audit` 进行 Dockerfile 静态合规审计
- 镜像操作：`ktib images` list/login/logout/pull/push/save/load/tag/inspect/manifest
- 构建器操作：`ktib builders` add/build/copy/commit/from/label/list/mount/run/rm/umount

## 支持平台
- `x86_64`
- `aarch64`

## 安装
使用 Make：

```bash
make build
make install
```

或从源码构建（示例）：

```bash
go build -ldflags "-s -w -X=main.version=vX.Y.Z" -tags "seccomp" ./cmd/ktib
```

## 快速开始
初始化与构建：

```bash
# 初始化项目骨架
ktib project init --type minimal /path/to/project

# 生成默认配置
ktib project default_config --type minimal > /path/to/project/config.yml

# 构建与清理 rootfs
ktib project build-rootfs --config /path/to/project/config.yml /path/to/project
ktib project clean-rootfs --type minimal /path/to/project

# 构建镜像
ktib project build --name myimage --tag v1 /path/to/project

# 一站式构建
ktib make --init --type minimal --name myimage --tag v1 /path/to/project
```

审计与镜像操作：

```bash
# Dockerfile 合规审计
ktib scan dockerfile-audit --dockerfile ./Dockerfile --output audit.json

# 常用镜像操作
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

构建器操作示例：

```bash
# 从基础镜像创建构建器
ktib builders from --name mybuilder docker.io/library/alpine:latest

# 在构建器内执行构建
ktib builders build -f Dockerfile -t myimage:v1
```

## 命令文档
- 项目管理：`docs/commands/project.md`
- 镜像操作：`docs/commands/images.md`
- 构建器操作：`docs/commands/builders.md`
- 扫描功能：`docs/commands/scan.md`

## 自动补全
为常用 Shell 生成自动补全：

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

## 日志与版本
- 全局日志：`--log-level`（trace|debug|info|warn|error|fatal|panic）、`--log-format`（text|json）
- 版本信息：`ktib version`

## 许可证
本项目采用 [Mulan PSL v2](http://license.coscl.org.cn/MulanPSL2) 许可证。

## 参与贡献
1. Fork 本仓库
2. 新建分支（如 `feat/xxx`）
3. 提交代码并通过 CI
4. 创建 Pull Request 并完善说明
