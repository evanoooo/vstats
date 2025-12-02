# GitHub Actions 工作流说明

## Build and Release Go

这是 vStats 的主要构建和发布工作流，使用 Go 语言实现。**Rust 版本已弃用，所有构建现在都使用 Go。**

### 触发方式

1. **自动触发**：当推送以 `v` 开头的 tag 时（例如 `v0.1.0`）
2. **手动触发**：在 GitHub Actions 页面手动运行，需要提供版本号

### 支持的平台

#### Server (vstats-server)
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
- FreeBSD (amd64, arm64)

#### Agent (vstats-agent)
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
- FreeBSD (amd64, arm64)

### 使用方法

#### 自动发布（推荐）

1. 创建并推送 tag：
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. GitHub Actions 会自动：
   - 编译所有平台和架构的二进制文件
   - 创建 GitHub Release
   - 上传所有二进制文件和校验和文件

#### 手动发布

1. 前往 GitHub Actions 页面
2. 选择 "Build and Release Go" 工作流
3. 点击 "Run workflow"
4. 输入版本号（例如：`0.1.0`）
5. 点击 "Run workflow"

### 版本号格式

- 版本号应该遵循语义化版本（Semantic Versioning）
- Tag 格式：`v0.1.0`（带 `v` 前缀）
- 版本号会被注入到二进制文件中，可通过 `--version` 或 API 查询

### 输出文件

每个构建会生成：
- 二进制文件：`vstats-server-{os}-{arch}` 或 `vstats-server-{os}-{arch}.exe`
- 校验和文件：`checksums.txt`（包含所有文件的 SHA256 校验和）

### 构建参数

- Go 版本：1.22
- CGO：禁用（静态链接）
- 构建标志：`-trimpath -a -installsuffix cgo`

