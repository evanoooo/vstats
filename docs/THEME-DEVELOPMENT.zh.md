# VStats 主题开发指南

本指南介绍如何为 VStats 创建可从 GitHub 或其他来源安装的自定义主题。

## 目录

- [快速开始](#快速开始)
- [主题结构](#主题结构)
- [主题清单 (theme.json)](#主题清单-themejson)
- [主题样式 (theme.css)](#主题样式-themecss)
- [CSS 变量参考](#css-变量参考)
- [组件样式](#组件样式)
- [发布主题](#发布主题)
- [安装方式](#安装方式)

## 快速开始

1. 在 GitHub 上创建一个新仓库
2. 添加 `theme.json` 清单文件
3. 添加 `theme.css` 样式文件
4. 用户可通过 `你的用户名/仓库名` 安装

## 主题结构

VStats 主题至少包含两个文件：

```
my-theme/
├── theme.json      # 主题清单 (必需)
├── theme.css       # 主题样式 (必需)
├── preview.png     # 预览图 (推荐)
└── README.md       # 说明文档 (推荐)
```

## 主题清单 (theme.json)

### 必填字段

```json
{
  "id": "my-awesome-theme",
  "name": "My Awesome Theme",
  "nameZh": "我的主题",
  "version": "1.0.0",
  "author": "你的用户名",
  "description": "A beautiful theme",
  "descriptionZh": "一个漂亮的主题",
  "isDark": true,
  "style": "glass",
  "preview": {
    "primary": "#0a0a0f",
    "secondary": "#1a1a2e",
    "accent": "#ff6b6b",
    "background": "#0a0a0f"
  },
  "fonts": {
    "heading": "\"Inter\", sans-serif",
    "body": "\"Inter\", system-ui, sans-serif",
    "mono": "\"JetBrains Mono\", monospace"
  },
  "borderRadius": "12px",
  "cardStyle": "glass"
}
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 唯一标识符（小写字母、数字、连字符）|
| `name` | string | 显示名称（英文）|
| `nameZh` | string | 显示名称（中文）|
| `version` | string | 语义化版本号（如 `1.0.0`）|
| `author` | string | 作者名称 |
| `isDark` | boolean | 是否为深色主题 |
| `style` | string | UI 风格：`flat`、`glass`、`neumorphic`、`brutalist`、`minimal` |

## 主题样式 (theme.css)

### 基本结构

```css
/* 主题类 - 所有样式必须限定在此类下 */
.theme-my-awesome-theme {
  /* CSS 变量 */
  --theme-accent: #ff6b6b;
  --theme-accent-soft: rgba(255, 107, 107, 0.15);
  --bg-primary: #0a0a0f;
  --bg-card: rgba(26, 26, 46, 0.7);
  --text-primary: #f8fafc;
  --border-primary: rgba(255, 107, 107, 0.2);
  
  color-scheme: dark;
}
```

### 重要：使用属性选择器

由于 VStats 会将主题 ID 添加到组件类名中（如 `vps-overview-card--online-midnight`），
请使用**属性选择器**来匹配这些动态类名：

```css
/* ✅ 正确：使用属性选择器匹配任意主题后缀 */
.theme-my-awesome-theme [class*="vps-overview-card--online"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
}

/* ❌ 错误：硬编码主题 ID */
.theme-my-awesome-theme .vps-overview-card--online-my-awesome-theme {
  /* 可以工作但不够灵活 */
}
```

## CSS 变量参考

```css
/* 强调色 */
--theme-accent           /* 主强调色 */
--theme-accent-hover     /* 悬停状态 */
--theme-accent-soft      /* 低透明度 */
--theme-glow             /* 发光效果 */

/* 背景色 */
--bg-primary             /* 页面背景 */
--bg-secondary           /* 次级背景 */
--bg-card                /* 卡片背景 */
--bg-input               /* 输入框背景 */

/* 文字颜色 */
--text-primary           /* 主要文字 */
--text-secondary         /* 次要文字 */
--text-muted             /* 禁用文字 */

/* 边框 */
--border-primary         /* 默认边框 */
--border-secondary       /* 强调边框 */
```

## 组件样式

### 概览卡片

```css
/* 使用属性选择器匹配动态类名 */
.theme-my-theme [class*="vps-overview-card--online"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
  box-shadow: 0 4px 24px rgba(16, 185, 129, 0.2);
}

.theme-my-theme [class*="vps-overview-card--offline"] {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.28), rgba(248, 113, 113, 0.12));
  border: 1px solid rgba(239, 68, 68, 0.5);
}

.theme-my-theme [class*="vps-overview-card--download"] {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.28), rgba(96, 165, 250, 0.12));
  border: 1px solid rgba(59, 130, 246, 0.5);
}

.theme-my-theme [class*="vps-overview-card--upload"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
}

/* 概览值 */
.theme-my-theme [class*="vps-overview-value--"] {
  color: var(--text-primary);
  font-weight: 600;
}
```

### 服务器卡片

```css
.theme-my-theme .vps-card {
  background: var(--bg-card);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-primary);
}

.theme-my-theme .vps-card:hover {
  border-color: var(--border-secondary);
  transform: translateY(-4px);
}
```

### 紧凑表格视图

```css
.theme-my-theme [class*="vps-compact-header--"] {
  background: rgba(255, 255, 255, 0.03);
}

.theme-my-theme [class*="vps-compact-row--"] {
  background: var(--bg-card);
}

.theme-my-theme [class*="vps-compact-row--"]:hover {
  background: var(--bg-secondary-hover);
}
```

### 进度条

```css
.theme-my-theme .vps-resource-bar-track {
  background: rgba(255, 255, 255, 0.1);
}

.theme-my-theme .vps-resource-bar-fill {
  background: linear-gradient(90deg, var(--theme-accent), #818cf8);
}
```

## 发布主题

### 版本管理

使用语义化版本：

```bash
git tag v1.0.0
git push origin v1.0.0
```

用户可以安装指定版本：

```
你的用户名/仓库名@v1.0.0
```

## 安装方式

| 格式 | 示例 |
|------|------|
| 仓库根目录 | `username/repo` |
| 子目录 | `username/repo/themes/my-theme` |
| 指定版本 | `username/repo@v1.0.0` |
| 指定分支 | `username/repo@develop` |

## 提示

1. **使用属性选择器** - 如 `[class*="vps-overview-card--online"]` 匹配动态类名

2. **充分测试** - 测试不同状态（在线/离线服务器，高/低资源使用率）

3. **使用 CSS 变量** - 方便自定义

4. **考虑无障碍** - 确保足够的颜色对比度

---

祝你主题制作愉快！🎨

