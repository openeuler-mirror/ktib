# ktib analyze

## 描述
`analyze` 命令用于分析镜像，识别空间浪费、列出已安装的软件包（RPM 和 Python），并提供优化建议。
该命令可以帮助用户了解镜像的组成，发现潜在的冗余文件，并生成包含软件包版本和校验和（Digest）的详细清单。

主要功能包括：
- **层级分析**：识别每一层增加、删除的文件大小。
- **文件系统统计**：统计大目录、文件类型分布。
- **软件包扫描**：
  - **RPM**：扫描 `/var/lib/rpm` 数据库，列出包名、版本、大小、License 及签名摘要 (Digest)。
  - **Python**：扫描常见路径下的 Python 包（`.dist-info`, `.egg-info`），列出元数据及文件摘要 (Digest)。
- **浪费检测**：识别跨层重复文件（Duplicate Files）。
- **优化建议**：根据分析结果给出缩减镜像体积的建议。

## 用法
```bash
ktib analyze <image> [flags]
```

## 示例

### 基本分析（输出摘要）
```bash
ktib analyze myimage:latest
```

### 输出 JSON 格式
```bash
ktib analyze myimage:latest --output json
```

### 将报告保存到文件
```bash
ktib analyze myimage:latest --file report.json
```

### 快速模式（跳过校验和计算）
```bash
ktib analyze myimage:latest --fast
```

## 选项

| 选项 | 简写 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `--output` | `-o` | 输出格式 (summary, json) | `summary` |
| `--file` | `-f` | 将分析报告输出到指定文件 | `""` |
| `--fast` | | 启用快速模式（跳过文件校验和计算与深度检查） | `false` |

## 输出字段说明 (JSON)
JSON 报告包含以下主要字段：
- `image_info`: 镜像基本信息（Ref, Size, OS, Created, Architecture）。
- `analysis`:
  - `layers`: 层级详细信息（Digest, Size, Command, Added/Deleted/Modified Files）。
  - `packages`:
    - `rpm`: RPM 包列表（Name, Version, Size, License, **Digest**）。
    - `python`: Python 包列表（Name, Version, License, **Digest**）。
  - `filesystem`: 文件系统统计（TopDirectories, FileTypes）。
  - `waste_detection`: 浪费检测（Duplicates, Caches）。
- `recommendations`: 优化建议列表。

> **注意**：`digest` 字段为新增功能，用于提供软件包或元数据的完整性校验值（如 RPM SigMD5 或 Python Metadata SHA256）。
