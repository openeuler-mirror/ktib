# ktib fusion

## 描述
`ktib fusion` 是一个用于镜像深度融合与裁剪的工具。通过依赖分析和 RPM 数据库重构，它可以将臃肿的镜像转化为仅包含运行时必要组件的精简镜像。

该命令通过解析镜像的依赖关系（RPM 包依赖和文件级依赖），生成一份“保留列表”，然后基于该列表重建 RPM 数据库并合成新的 Rootfs，从而实现极致的镜像瘦身。

## 用法
```bash
ktib fusion <image> [flags]
```

## 选项
| 选项 | 简写 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `--config` | `-c` | 融合配置文件路径 (YAML) | "" |
| `--output-dir` | `-o` | 融合后 Rootfs 的输出目录 | "fusion_output_<image_name>" |
| `--tag` | `-t` | (可选) 生成新镜像的 Tag | "" |

## 配置文件示例 (fusion.yaml)
```yaml
fusion:
  keep_packages:
    - nginx
    - openssl
  drop_packages:
    - vim
    - curl
  behavior:
    retain_docs: false
    retain_weak_deps: false
    auto_heal_libs: true
```

## 示例

### 1. 基本融合
使用默认策略对镜像进行融合，输出到默认目录：
```bash
ktib fusion myimage:latest
```

### 2. 指定输出目录
```bash
ktib fusion myimage:latest --output-dir ./slim-rootfs
```

### 3. 使用配置文件进行精细控制
```bash
ktib fusion myimage:latest --config my-policy.yaml --output-dir ./output
```
