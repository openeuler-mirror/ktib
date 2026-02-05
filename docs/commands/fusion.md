# ktib fusion

## 描述
`ktib fusion` 是一个用于镜像深度融合与裁剪的工具。通过依赖分析和 RPM 数据库重构，它可以将臃肿的镜像转化为仅包含运行时必要组件的精简镜像。

该命令通过解析镜像的依赖关系（RPM 包依赖和文件级依赖），生成一份“保留列表”，然后基于该列表重建 RPM 数据库并合成新的 Rootfs，从而实现极致的镜像瘦身。

## 工作流
1. **生成配置**：使用 `ktib fusion --dump-config fusion.yaml` 生成包含详细注释的默认配置文件。
2. **编辑策略**：根据镜像特点，编辑 `fusion.yaml`，指定需要保留的包 (`keep_packages`) 或文件 (`keep_files`)，以及需要移除的包 (`drop_packages`)。
3. **执行融合**：运行 `ktib fusion <image> --config fusion.yaml --tag <new-tag>` 生成精简镜像。
4. **验证结果**：使用 `ktib analyze` 对比新旧镜像，确认体积缩减效果。

## 用法
```bash
ktib fusion [image] [flags]
```

## 选项
| 选项 | 简写 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `--config` | `-c` | 融合配置文件路径 (YAML) | "" |
| `--output-dir` | `-o` | 融合后 Rootfs 的输出目录（可选；不设置则使用临时目录并在成功后自动清理） | "" |
| `--tag` | `-t` | 生成新镜像的 Tag（必填） | "" |
| `--dump-config` |  | 导出默认融合配置到文件（传 `-` 输出到 stdout；不带参数时写入 `fusion.yaml`） | "" |
| `--save-data` |  | 保存分析数据到 JSON 文件（便于后续复用） | "" |
| `--from-data` |  | 从 JSON 文件加载分析数据以跳过镜像扫描（可不传 image 参数） | "" |
| `--lang` |  | 输出语言 (en, zh) | "en" |

## 配置文件示例 (fusion.yaml)
生成的默认配置文件包含详细注释，帮助您理解每个字段的作用：

```yaml
# Fusion Configuration File
# This file controls how ktib fusion optimizes the image.

fusion:
  # Packages to explicitly keep in the final image.
  # Dependencies of these packages will be automatically resolved and kept.
  keep_packages:
    - bash
    - coreutils
    - systemd
    # - nginx
    # - openssl

  # Files to explicitly keep (absolute paths).
  # Use this for files not owned by any RPM package (e.g. app binaries, config files).
  keep_files:
    # - /app/my-app
    # - /etc/my-app/config.json

  # Packages to explicitly remove.
  drop_packages:
    # - vim
    # - curl

  behavior:
    # Whether to retain documentation files (man pages, /usr/share/doc, etc.)
    retain_docs: false

    # Whether to retain weak dependencies (Recommends/Suggests)
    retain_weak_deps: false

    # Whether to attempt automatic recovery if broken shared libraries are detected
    auto_heal_libs: true

    # Whether to keep files not owned by any RPM package (default: true)
    # Set to false to remove all unowned files unless specified in keep_files
    retain_unowned: true
```

## 示例

### 1. 基本融合
使用默认策略对镜像进行融合并生成新镜像（不保留 Rootfs 输出）：
```bash
ktib fusion myimage:latest --tag myimage:slim
```

### 2. 指定输出目录
```bash
ktib fusion myimage:latest --tag myimage:slim --output-dir ./slim-rootfs
```

### 3. 使用配置文件进行精细控制
```bash
ktib fusion myimage:latest --config my-policy.yaml --tag myimage:slim --output-dir ./output
```

### 4. 生成默认配置模板
写入当前目录的 `fusion.yaml`：
```bash
ktib fusion --dump-config
```
输出到 stdout（便于重定向/管道处理）：
```bash
ktib fusion --dump-config=-
```

### 5. 一行命令生成新镜像
```bash
ktib fusion myimage:latest --config fusion.yaml --tag myimage:slim
```
说明：`--tag` 会在融合完成后将 Rootfs 打包并构建 `FROM scratch` 的新镜像（不依赖外部 buildah 命令）。

### 6. 复用 analyze 的 JSON 数据
先保存一次分析数据（推荐加 `--fast`）：
```bash
ktib analyze myimage:latest --fast --save-data analysis.json
```
再使用该数据进行 fusion（可省略 image 参数，自动读取 `image_info.ref`）：
```bash
ktib fusion --from-data analysis.json --tag myimage:slim --output-dir ./output
```
限制：由于 analyze 的 JSON 不包含包级文件列表，`auto_heal_libs` 在 `--from-data` 模式下会自动跳过。
