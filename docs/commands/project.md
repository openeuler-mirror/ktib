# ktib project

## 子命令
- `default_config`
- `init`
- `build-rootfs`
- `clean-rootfs`
- `build`

## 命令说明

### `default_config`
生成默认配置文件的模板。

**用法：**
```bash
ktib project default_config > config.yml
```

**说明：**
- 该命令会输出一个默认的 YAML 配置模板到标准输出
- 通常需要重定向到文件（如 `config.yml`）来保存配置
- 配置模板包含 rootfs 构建所需的基本配置项

### `init`
初始化项目结构。

**用法：**
```bash
ktib project init <项目路径>
```

**参数：**
- `<项目路径>`：要初始化项目的目录路径（必需）

**说明：**
- 创建必要的项目目录结构和文件
- 为后续的 rootfs 构建和镜像构建做准备

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
ktib project build-rootfs --config config.yml /path/to/project
```

**说明：**
- 根据配置文件构建 rootfs 文件系统
- 配置文件指定了要安装的软件包和系统设置
- 构建完成后会提示运行 `clean-rootfs` 命令进行清理

### `clean-rootfs`
清理 rootfs 中不必要的文件和软件包。

**用法：**
```bash
ktib project clean-rootfs [--type <镜像类型>] <项目路径>
```

**参数：**
- `--type`：镜像类型（可选，有效值：micro、minimal、platform、init）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
# 使用默认清理
ktib project clean-rootfs /path/to/project

# 指定镜像类型进行清理
ktib project clean-rootfs --type minimal /path/to/project
```

**说明：**
- 移除 rootfs 中不必要的文件和软件包
- 执行额外的环境配置操作以优化镜像大小
- 支持不同类型的镜像优化策略

### `build`
从 rootfs 构建容器镜像。

**用法：**
```bash
ktib project build [--name <镜像名称>] [--tag <标签>] <项目路径>
```

**参数：**
- `--name`：容器镜像名称（可选，默认：ktib-image）
- `--tag`：镜像标签（可选，默认：latest）
- `<项目路径>`：项目目录路径（必需）

**示例：**
```bash
# 使用默认名称和标签
ktib project build /path/to/project

# 指定镜像名称和标签
ktib project build --name myimage --tag v1.0 /path/to/project
```

**说明：**
- 使用 rootfs 和 Dockerfile 构建容器镜像
- 将 rootfs 打包成可用的容器镜像
- 生成的镜像可用于容器运行时

## 典型工作流程
1. **初始化项目：**
   ```bash
   ktib project init /path/to/project
   ```

2. **生成配置：**
   ```bash
   ktib project default_config > config.yml
   ```

3. **构建 rootfs：**
   ```bash
   ktib project build-rootfs --config config.yml /path/to/project
   ```

4. **清理 rootfs：**
   ```bash
   ktib project clean-rootfs --type platform /path/to/project
   ```

5. **构建镜像：**
   ```bash
   ktib project build --name myimage --tag latest /path/to/project