# 脚本安装

安装脚本提供了一种无需 Docker 直接在系统上安装 vStats 的简便方式。

## 前提条件

- Linux（Debian、Ubuntu、CentOS、RHEL、Fedora、Arch 等）或 macOS
- Root/sudo 权限
- curl 或 wget

## 安装服务端

### 使用 curl

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

### 使用 wget

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash
```

## 脚本做了什么

1. 检测操作系统和架构
2. 下载对应的二进制文件
3. 创建安装目录（Linux 为 `/opt/vstats`，macOS 为 `~/.vstats`）
4. 设置 systemd 服务（Linux）或 launchd（macOS）
5. 启动服务
6. 生成管理员密码

## 安装路径

| 操作系统 | 二进制位置 | 数据目录 |
|----------|------------|----------|
| Linux | `/opt/vstats/vstats-server` | `/opt/vstats/data` |
| macOS | `~/.vstats/vstats-server` | `~/.vstats/data` |

## 获取管理员密码

安装后：

```bash
# Linux
journalctl -u vstats | grep -i password

# macOS
tail -20 ~/.vstats/data/vstats.log | grep -i password
```

## 重置密码

```bash
# Linux
/opt/vstats/vstats-server --reset-password

# macOS
~/.vstats/vstats-server --reset-password
```

## 升级

升级到最新版本：

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

或：

```bash
wget -qO- https://vstats.zsoft.cc/install.sh | sudo bash -s -- upgrade
```

## 卸载

完全移除 vStats：

```bash
curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash -s -- uninstall
```

::: warning 警告
这将移除二进制文件和服务，但默认保留数据。添加 `--purge` 同时删除数据。
:::

## 自定义安装

可以使用环境变量自定义安装：

```bash
# 自定义端口
PORT=8080 curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash

# 自定义安装目录（仅限 Linux）
INSTALL_DIR=/usr/local/vstats curl -fsSL https://vstats.zsoft.cc/install.sh | sudo bash
```

## 验证安装

```bash
# 检查服务状态
systemctl status vstats  # Linux
launchctl list | grep vstats  # macOS

# 检查服务器是否响应
curl http://localhost:3001/health
```

## 故障排除

### 安装失败

检查错误信息。常见问题：

- **Permission denied**：使用 `sudo` 运行
- **Architecture not supported**：仅支持 amd64 和 arm64
- **Network error**：检查网络连接

### 服务无法启动

```bash
# 检查日志
journalctl -u vstats -f  # Linux
tail -f ~/.vstats/data/vstats.log  # macOS

# 检查服务状态
systemctl status vstats
```

### 端口已被占用

编辑配置文件更改端口：

```bash
# Linux
vim /opt/vstats/vstats-config.json

# macOS
vim ~/.vstats/vstats-config.json
```

然后重启服务：

```bash
systemctl restart vstats  # Linux
```

