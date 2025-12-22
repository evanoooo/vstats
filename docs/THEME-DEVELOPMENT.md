# VStats Theme Development Guide

This guide explains how to create custom themes for VStats that can be installed from GitHub or other sources.

## Table of Contents

- [Quick Start](#quick-start)
- [Theme Structure](#theme-structure)
- [Theme Manifest (theme.json)](#theme-manifest-themejson)
- [Theme CSS (theme.css)](#theme-css-themecss)
- [CSS Variables Reference](#css-variables-reference)
- [Component Styling](#component-styling)
- [Publishing Your Theme](#publishing-your-theme)
- [Installation](#installation)
- [Examples](#examples)

## Quick Start

1. Create a new GitHub repository for your theme
2. Add `theme.json` manifest file
3. Add `theme.css` stylesheet
4. Users can install with: `your-username/your-repo`

## Theme Structure

A VStats theme consists of at minimum two files:

```
my-theme/
├── theme.json      # Theme manifest (required)
├── theme.css       # Theme stylesheet (required)
├── assets/         # Optional assets directory
│   ├── fonts/
│   └── images/
└── README.md       # Documentation (recommended)
```

## Theme Manifest (theme.json)

The `theme.json` file describes your theme and its properties.

### Required Fields

```json
{
  "id": "my-awesome-theme",
  "name": "My Awesome Theme",
  "version": "1.0.0",
  "author": "your-username",
  "description": "A beautiful theme for VStats",
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

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (lowercase, hyphens only, e.g., `my-theme`) |
| `name` | string | Display name |
| `version` | string | Semantic version (e.g., `1.0.0`) |
| `author` | string | Author name or GitHub username |
| `description` | string | Brief description of the theme |
| `isDark` | boolean | Whether this is a dark theme |
| `style` | string | UI style: `flat`, `glass`, `neumorphic`, `brutalist`, `minimal` |
| `preview` | object | Preview colors for theme selector |
| `fonts` | object | Font families for different text types |
| `borderRadius` | string | Default border radius |
| `cardStyle` | string | Card style identifier |

### Optional Fields

```json
{
  "nameZh": "我的主题",
  "descriptionZh": "一个漂亮的 VStats 主题",
  "license": "MIT",
  "homepage": "https://github.com/user/my-theme",
  "repository": "https://github.com/user/my-theme",
  "keywords": ["dark", "neon", "cyberpunk"],
  "minVersion": "1.0.0",
  "cssFile": "theme.css",
  "assetsDir": "assets/",
  "previewImage": "preview.png"
}
```

### Style Types

| Style | Description |
|-------|-------------|
| `flat` | Solid colors, minimal shadows |
| `glass` | Glassmorphism with blur effects |
| `neumorphic` | Soft shadows, embossed elements |
| `brutalist` | Bold, raw design with hard edges |
| `minimal` | Maximum whitespace, minimal elements |

## Theme CSS (theme.css)

Your CSS file should define styles scoped to your theme class.

### Basic Structure

```css
/* ========================================
   Theme: My Awesome Theme
   Author: your-username
   ======================================== */

/* Theme class - all styles must be scoped to this */
.theme-my-awesome-theme {
  /* CSS Variables */
  --theme-accent: #ff6b6b;
  --theme-accent-hover: #ff5252;
  --theme-accent-soft: rgba(255, 107, 107, 0.15);
  --theme-glow: rgba(255, 107, 107, 0.4);
  
  --bg-primary: #0a0a0f;
  --bg-secondary: rgba(26, 26, 46, 0.85);
  --bg-card: rgba(26, 26, 46, 0.7);
  --bg-input: rgba(255, 255, 255, 0.08);
  
  --text-primary: #f8fafc;
  --text-secondary: #cbd5e1;
  --text-tertiary: #94a3b8;
  --text-muted: #64748b;
  
  --border-primary: rgba(255, 107, 107, 0.2);
  --border-secondary: rgba(255, 107, 107, 0.3);
  
  --shadow-card: 0 8px 32px rgba(0, 0, 0, 0.4);
  
  color-scheme: dark;
}

/* Card styles */
.theme-my-awesome-theme .vps-card {
  background: var(--bg-card);
  backdrop-filter: blur(20px);
  border: 1px solid var(--border-primary);
  border-radius: 12px;
  box-shadow: var(--shadow-card);
}

.theme-my-awesome-theme .vps-card:hover {
  border-color: var(--border-secondary);
  transform: translateY(-4px);
}

/* More styles... */
```

### Important: Class Naming

**Always scope your styles to `.theme-{your-theme-id}`**

The theme ID in your CSS class must match the `id` field in your `theme.json`:

```json
{ "id": "my-awesome-theme" }
```
```css
.theme-my-awesome-theme { ... }
```

### Using Attribute Selectors for Flexibility

Since VStats appends the theme ID to component class names (e.g., `vps-overview-card--online-midnight`), 
use **attribute selectors** to match these dynamic classes:

```css
/* ✅ Good: Use attribute selector to match any theme suffix */
.theme-my-awesome-theme [class*="vps-overview-card--online"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
}

/* ❌ Bad: Hardcoded theme ID won't work with dynamic class names */
.theme-my-awesome-theme .vps-overview-card--online-my-awesome-theme {
  /* This works but is fragile */
}
```

## CSS Variables Reference

### Core Variables

```css
/* Accent Colors */
--theme-accent           /* Primary accent color */
--theme-accent-hover     /* Accent hover state */
--theme-accent-soft      /* Accent with low opacity */
--theme-glow             /* Glow/shadow color */

/* Backgrounds */
--bg-primary             /* Page background */
--bg-secondary           /* Secondary background */
--bg-secondary-hover     /* Hover state */
--bg-card                /* Card background */
--bg-input               /* Input field background */

/* Text Colors */
--text-primary           /* Main text */
--text-secondary         /* Secondary text */
--text-tertiary          /* Tertiary text */
--text-muted             /* Muted/disabled text */

/* Borders */
--border-primary         /* Default border */
--border-secondary       /* Stronger border */
--border-hover           /* Hover border */

/* Shadows */
--shadow-card            /* Card shadow */

/* Font Variables (set from manifest) */
--theme-font-heading     /* Heading font family */
--theme-font-body        /* Body text font family */
--theme-font-mono        /* Monospace font family */
--theme-border-radius    /* Default border radius */
```

## Component Styling

### Server Cards

```css
.theme-my-theme .vps-card {
  /* Main card container */
}

.theme-my-theme .vps-card:hover {
  /* Card hover state */
}

/* Use attribute selectors for themed variants */
.theme-my-theme [class*="vps-card--"] {
  /* Matches vps-card--dark, vps-card--light, etc. */
}
```

### Overview Cards (Dashboard Stats)

```css
/* Use attribute selectors for dynamic class names */
.theme-my-theme [class*="vps-overview-card--online"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
  box-shadow: 0 4px 24px rgba(16, 185, 129, 0.2);
}

.theme-my-theme [class*="vps-overview-card--offline"] {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.28), rgba(248, 113, 113, 0.12));
  border: 1px solid rgba(239, 68, 68, 0.5);
  box-shadow: 0 4px 24px rgba(239, 68, 68, 0.2);
}

.theme-my-theme [class*="vps-overview-card--download"] {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.28), rgba(96, 165, 250, 0.12));
  border: 1px solid rgba(59, 130, 246, 0.5);
  box-shadow: 0 4px 24px rgba(59, 130, 246, 0.2);
}

.theme-my-theme [class*="vps-overview-card--upload"] {
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.28), rgba(52, 211, 153, 0.12));
  border: 1px solid rgba(16, 185, 129, 0.5);
  box-shadow: 0 4px 24px rgba(16, 185, 129, 0.2);
}

/* Overview Labels */
.theme-my-theme .vps-overview-label--online { color: #34d399; }
.theme-my-theme .vps-overview-label--offline { color: #f87171; }
.theme-my-theme .vps-overview-label--download { color: #60a5fa; }
.theme-my-theme .vps-overview-label--upload { color: #34d399; }

/* Overview Values */
.theme-my-theme [class*="vps-overview-value--"] {
  color: var(--text-primary);
  font-weight: 600;
}
```

### Compact Table View

```css
.theme-my-theme [class*="vps-compact-header--"] {
  background: rgba(255, 255, 255, 0.03);
  color: var(--text-muted);
}

.theme-my-theme [class*="vps-compact-row--"] {
  background: var(--bg-card);
}

.theme-my-theme [class*="vps-compact-row--"]:hover {
  background: var(--bg-secondary-hover);
}

.theme-my-theme [class*="vps-compact-node-name--"] {
  color: var(--text-primary);
}
```

### Progress Bars

```css
.theme-my-theme .vps-resource-bar-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 8px;
}

.theme-my-theme .vps-resource-bar-fill {
  background: linear-gradient(90deg, var(--theme-accent), #818cf8);
  box-shadow: 0 0 10px var(--theme-accent-soft);
}
```

### Buttons

```css
.theme-my-theme .vps-btn,
.theme-my-theme button {
  background: var(--theme-accent-soft);
  border: 1px solid var(--border-secondary);
  color: var(--text-primary);
}

.theme-my-theme .vps-btn:hover,
.theme-my-theme button:hover {
  box-shadow: 0 0 20px var(--theme-accent-soft);
}
```

### Form Elements

```css
.theme-my-theme input,
.theme-my-theme select,
.theme-my-theme textarea {
  background: var(--bg-input);
  border: 1px solid var(--border-primary);
  color: var(--text-primary);
}

.theme-my-theme input:focus {
  border-color: var(--theme-accent);
  box-shadow: 0 0 0 3px var(--theme-accent-soft);
}
```

## Publishing Your Theme

### GitHub Repository Structure

```
your-theme-repo/
├── theme.json
├── theme.css
├── preview.png          # Screenshot for theme gallery
├── README.md
└── LICENSE
```

### Versioning

Use semantic versioning:

- `1.0.0` - Initial release
- `1.0.1` - Bug fixes
- `1.1.0` - New features (backwards compatible)
- `2.0.0` - Breaking changes

Create Git tags for releases:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Users can install specific versions:

```
your-username/your-repo@v1.0.0
```

## Installation

### From GitHub

| Format | Example |
|--------|---------|
| Repository root | `username/repo` |
| Subdirectory | `username/repo/themes/my-theme` |
| Specific version | `username/repo@v1.0.0` |
| Branch | `username/repo@develop` |

### From URL

Direct URL to `theme.json`:

```
https://example.com/themes/my-theme/theme.json
```

## Examples

### Minimal Dark Theme

```json
{
  "id": "midnight-minimal",
  "name": "Midnight Minimal",
  "version": "1.0.0",
  "author": "example",
  "description": "A minimal dark theme",
  "isDark": true,
  "style": "minimal",
  "preview": {
    "primary": "#0f0f0f",
    "secondary": "#1a1a1a",
    "accent": "#ffffff",
    "background": "#0f0f0f"
  },
  "fonts": {
    "heading": "\"Inter\", sans-serif",
    "body": "\"Inter\", sans-serif",
    "mono": "\"SF Mono\", monospace"
  },
  "borderRadius": "4px",
  "cardStyle": "minimal"
}
```

### Neon Cyberpunk Theme

```json
{
  "id": "neon-nights",
  "name": "Neon Nights",
  "nameZh": "霓虹之夜",
  "version": "1.0.0",
  "author": "example",
  "description": "Vibrant neon colors on dark background",
  "descriptionZh": "深色背景上的霓虹色彩",
  "isDark": true,
  "style": "brutalist",
  "preview": {
    "primary": "#0a0015",
    "secondary": "#1a0030",
    "accent": "#ff00ff",
    "background": "#0a0015"
  },
  "fonts": {
    "heading": "\"Orbitron\", sans-serif",
    "body": "\"Rajdhani\", sans-serif",
    "mono": "\"Share Tech Mono\", monospace"
  },
  "borderRadius": "0px",
  "cardStyle": "neon"
}
```

## Tips & Best Practices

1. **Use attribute selectors** - e.g., `[class*="vps-overview-card--online"]` to match dynamic class names

2. **Test thoroughly** - Test with different data states (online/offline servers, high/low resource usage)

3. **Use CSS variables** - This makes customization easier

4. **Consider accessibility** - Ensure sufficient color contrast

5. **Keep file sizes small** - Minimize CSS and optimize images

6. **Use semantic versioning** - Makes it easier for users to track updates

## Troubleshooting

### Theme not loading?

- Check that `theme.json` is valid JSON
- Ensure the `id` field matches your CSS class name
- Verify the repository is public

### Styles not applying?

- Make sure all CSS rules are scoped to `.theme-{your-id}`
- Use attribute selectors for dynamic component classes
- Check browser dev tools for CSS specificity issues

### Updates not showing?

- Clear browser cache
- Try reinstalling the theme
- Check that version number was updated in `theme.json`

---

Happy theming! 🎨

