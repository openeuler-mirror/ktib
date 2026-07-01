# ktib project

## 子命令
- `default_config`
- `init`
- `build-rootfs`
- `clean-rootfs`
- `build`
- `make`

## 命令说明

### `default_config`
生成默认配置文件的模板。

**用法：**
```bash
ktib project default_config > config.yml
```

**可选参数：**
- `--timezone`：设置时区（默认：`Asia/Shanghai`）
- `--locale`：设置语言环境（默认：`C.UTF-8`）

**示例：**
```bash
# 生成默认配置示例
ktib project default_config > config.yml

# 指定时区
ktib project default_config --timezone "America/New_York" > config.yml

# 指定语言
ktib project default_config --locale "zh_CN.UTF-8" > config.yml

# 同时指定时区与语言
ktib project default_config --timezone "Europe/London" --locale "en_GB.UTF-8" > config.yml
```

**说明：**
- 输出一个默认的 YAML 配置模板到标准输出，通常配合重定向保存到文件
- 模板包含 rootfs 构建所需的基本配置项（packages、network、locale、timezone）

### `init`
初始化项目结构并可选写入默认配置。

**用法：**
```bash
ktib project init [--type <镜像类型>] [--config <配置文件路径>] [--timezone <时区>] [--locale <语言>] <项目路径>
```

**参数：**
- `--type`：镜像类型（可选，有效值：`micro|minimal|platform|init`）
- `--config`：生成并写入默认配置文件到指定路径（可选）
- `--timezone`：默认配置中的时区（可选，默认：`Asia/Shanghai`）
- `--locale`：默认配置中的语言（可选，默认：`C.UTF-8`）
- `<项目路径>`：要初始化项目的目录路径（必需）

**示例：**
```bash
# 创建项目骨架
ktib project init /path/to/project

# 指定类型并写入默认配置
ktib project init --type init --config /path/to/project/config.yml --timezone "Asia/Shanghai" --locale "C.UTF-8" /path/to/project
```

**说明：**
- 创建目录结构：`dockerfile/`、`rootfs/`、`files/`、`tests/`
- 生成模板文件：`dockerfile/Dockerfile`、`README.md`、`files/removeminimallist`、`files/unmaskService`
- 当类型为 `init` 时，`Dockerfile` 默认使用 `CMD ["/sbin/init"]`，其他类型使用 `CMD ["/bin/bash"]`
- 如果提供 `--config`，初始化后会写入默认配置文件（可用 `build-rootfs` 使用）

### `build-rootfs`
构建项目的 rootfs。

**用法：**
```bash
ktib project build-rootfs --config <配置文件路径> <项目路径>
```

**参数：**
- `--config`：配置文件路径（必需）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
ktib project build-rootfs --config /path/to/project/config.yml /path/to/project

# 先生成默认配置再构建
ktib project default_config > /path/to/project/config.yml
ktib project build-rootfs --config /path/to/project/config.yml /path/to/project
```

**说明：**
- 按配置安装软件包到 `rootfs/`（`yum/dnf`、`nodocs`、禁用弱依赖、`group_package_types=mandatory`）
- 写入网络、`/etc/dnf/vars/infra=container`、语言与 `/etc/locale.conf`、时区软链 `/etc/localtime` 与 `/etc/timezone`
- 初始化空的 `/etc/machine-id`，复制 `bash` skeleton

### `clean-rootfs`
清理 rootfs 中不必要的文件和软件包，并执行类型化优化。

**用法：**
```bash
ktib project clean-rootfs [--type <镜像类型>] <项目路径>
```

**参数：**
- `--type`：镜像类型（可选，有效值：`micro|minimal|platform|init`）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
# 使用默认清理
ktib project clean-rootfs /path/to/project

# 指定镜像类型进行清理（minimal 会根据 removeminimallist 移除包）
ktib project clean-rootfs --type minimal /path/to/project

# init/platform 类型的额外清理（pip 与 __pycache__）
ktib project clean-rootfs --type init /path/to/project
```

**说明：**
- 删除 locales、docs、icons、i18n、`var/cache/yum`、`var/cache/ldconfig`、日志与临时目录等
- `minimal` 类型：根据 `files/removeminimallist` 在 chroot 中批量移除包
- 解除服务屏蔽：按 `files/unmaskService` 在 chroot 中执行
- `platform`/`init` 类型：在 chroot 中安装 `pip`（`ensurepip`）并删除全局 `__pycache__`

### `build`
从 rootfs 构建容器镜像。

**用法：**
```bash
ktib project build [--name <镜像名称>] [--tag <标签>] <项目路径>
```

**参数：**
- `--name`：容器镜像名称（可选，默认：`ktib-image`）
- `--tag`：镜像标签（可选，默认：`latest`）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
# 使用默认名称和标签
ktib project build /path/to/project

# 指定镜像名称和标签
ktib project build --name myimage --tag v1.0 /path/to/project
```

**说明：**
- 打包 `rootfs/` 为 `rootfs.tar`，解析并优先使用项目 `dockerfile/Dockerfile`；如不存在则按类型生成最小 `Dockerfile`
- `init` 类型默认 `CMD ["/sbin/init"]`，其他类型默认 `CMD ["/bin/bash"]`
- 使用 buildah 接口构建镜像（支持 `name:tag` 标签）

### `make`
一键构建基础镜像：可选初始化项目、构建 rootfs、清理 rootfs、构建镜像。

**用法：**
```bash
ktib make [--init] [--config <配置文件>] [--type <镜像类型>] [--name <镜像名称>] [--tag <标签>] [--timezone <时区>] [--locale <语言>] <项目路径>
```

**参数：**
- `--init`：构建前初始化项目结构，若未提供 `--config` 则生成默认配置
- `--config`：配置文件路径（可选，与 `--init` 联动可自动生成）
- `--type`：镜像类型（可选，默认：`platform`；有效值：`micro|minimal|platform|init`）
- `--name`：容器镜像名称（可选，默认：`ktib-image`）
- `--tag`：镜像标签（可选，默认：`latest`）
- `--timezone`：默认配置中的时区（可选，默认：`Asia/Shanghai`）
- `--locale`：默认配置中的语言（可选，默认：`C.UTF-8`）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
# 初始化并一键构建（minimal 类型）
ktib make --init --type minimal --name myimage --tag latest /path/to/project

# 指定配置一键构建
ktib make --config /path/to/project/config.yml --type init --name myimage --tag latest /path/to/project

# 初始化并一键构建，带自定义时区与语言
ktib make --init --timezone "America/New_York" --locale "zh_CN.UTF-8" /path/to/project
```

**说明：**
- 当 `--init` 存在时：先初始化目录与模板，必要时生成默认配置
- 随后按配置构建 rootfs、执行类型化清理（含 `minimal` 移包、`init/platform` pip 与 `__pycache__` 清理、解屏蔽服务）
- 最后打包并构建容器镜像，默认标签 `name:latest`

## 典型工作流程
1. **初始化项目：**
   ```bash
   ktib project init --type platform /path/to/project
   ```

2. **生成配置：**
   ```bash
   ktib project default_config --type platform > config.yml
   ```

3. **构建 rootfs：**
   ```bash
   ktib project build-rootfs --config config.yml --type platform /path/to/project
   ```

4. **清理 rootfs：**
   ```bash
   ktib project clean-rootfs --type platform /path/to/project
   ```

5. **构建镜像：**
   ```bash
   ktib project build --name myimage --tag latest /path/to/project
   ```

6. **一键构建（可选）：**
   ```bash
   ktib make --init --type minimal --name myimage --tag latest /path/to/project
   ```

