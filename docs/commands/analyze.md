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
- **优化建议**：根据预置或自定义规则，给出缩减镜像体积的建议（如删除缓存、文档、非必要软件包）。

## 用法
```bash
ktib analyze <image> [flags]
```

## 示例

### 基本分析（输出摘要）
```bash
ktib analyze myimage:latest
```

### 指定优化建议等级
只运行安全等级（SAFE）的规则，忽略可能影响功能的建议：
```bash
ktib analyze myimage:latest --level SAFE
```

### 使用自定义规则文件
加载用户定义的规则文件进行分析：
```bash
ktib analyze myimage:latest --rules ./my_rules.yaml
```

### 导出默认规则
将内置的默认规则导出到默认系统路径 `/etc/ktib/default_rules.yaml`（如果失败则导出到当前目录），以便查看或修改：
```bash
ktib analyze --default-rules
```

### 分离分析与建议（离线模式）
仅执行数据分析并保存原始数据（不生成建议）：
```bash
ktib analyze myimage:latest --save-data raw_data.json
```

加载原始数据并生成建议（无需镜像，支持离线）：
```bash
ktib analyze --from-data raw_data.json
```

### 输出 JSON 格式并保存
```bash
ktib analyze myimage:latest --output json --file report.json
```

### 快速模式（跳过校验和计算）
```bash
ktib analyze myimage:latest --fast
```

## 选项

| 选项 | 简写 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `--output` | `-o` | 输出格式 (summary, json) | `summary` |
| `--file` | `-f` | 将分析报告保存到指定文件 (例如 report.json) | `""` |
| `--fast` | | 启用快速模式（跳过文件校验和计算与深度检查） | `false` |
| `--rules` | | 指定自定义规则文件路径（YAML格式）。如果不指定且 `/etc/ktib/default_rules.yaml` 存在，则加载该文件；否则使用内置规则。 | `""` |
| `--level` | | 覆盖运行等级，多个等级用逗号分隔 (如 "SAFE,STANDARD") | `""` (使用规则文件定义) |
| `--default-rules` | | 导出内置默认规则到 `/etc/ktib/default_rules.yaml` (失败则回退到当前目录) | `false` |
| `--save-data` | | 仅执行数据分析，将原始分析数据保存到指定 JSON 文件（跳过建议生成） | `""` |
| `--from-data` | | 从指定 JSON 文件加载分析数据并生成建议（离线模式，跳过镜像扫描） | `""` |

## 输出字段说明 (JSON)
JSON 报告已进行精简优化，移除了冗余的依赖链信息，包含以下主要字段：
- `image_info`: 镜像基本信息（Ref, Size, OS, Created, Architecture）。
- `analysis`:
  - `layers`: 层级详细信息（Digest, Size, Command, Added/Deleted/Modified Files）。
  - `packages`:
    - `rpm`: RPM 包列表（Name, Version, Size, License, **Digest**）。
    - `python`: Python 包列表（Name, Version, License, **Digest**）。
  - `filesystem`: 文件系统统计（TopDirectories, FileTypes）。
  - `waste_detection`: 浪费检测（Duplicates, Caches）。
- `recommendations`: 优化建议列表（Level, Code, Message, Command, Saving）。

> **注意**：`digest` 字段为新增功能，用于提供软件包或元数据的完整性校验值（如 RPM SigMD5 或 Python Metadata SHA256）。
