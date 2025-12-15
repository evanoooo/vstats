import { defineConfig } from 'vitepress'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const base = '/vp-docs/'

const __dirname = dirname(fileURLToPath(import.meta.url))
const pkg = JSON.parse(readFileSync(resolve(__dirname, '../../package.json'), 'utf-8')) as {
  version?: string
}
const version = `v${pkg.version ?? '0.0.0'}`
const mainSiteUrl = 'https://vstats.zsoft.cc/'

export default defineConfig({
  title: 'vStats Docs',
  description: 'Minimalist & Beautiful Server Monitoring Dashboard',
  
  // Base path for deployment under /vp-docs/
  base,
  
  // Output directory
  outDir: '../dist',
  
  head: [
    // Favicon / app icons
    ['link', { rel: 'icon', href: `${base}logo-server.svg` }],
    ['link', { rel: 'icon', type: 'image/svg+xml', href: `${base}logo-server.svg` }],
    ['link', { rel: 'shortcut icon', href: `${base}logo-server.svg` }],
    ['link', { rel: 'apple-touch-icon', href: `${base}logo-server.svg` }],
    ['meta', { name: 'theme-color', content: '#0ea5e9' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:title', content: 'vStats - Server Monitoring Dashboard' }],
    ['meta', { name: 'og:description', content: 'Minimalist & Beautiful Server Monitoring Dashboard. Go-powered, millisecond latency, one-click deployment.' }],
    ['meta', { name: 'og:image', content: 'https://vstats.zsoft.cc/theme/1.png' }],
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
  ],

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/' },
          {
            text: 'Server',
            items: [
              { text: 'Overview', link: '/server/' },
              { text: 'Guide', link: '/guide/getting-started' },
              { text: 'Reference', link: '/reference/api' },
            ],
          },
          { text: 'Cloud', link: '/cloud/' },
          { text: 'CLI', link: '/cli/' },
          { text: 'Demo', link: 'https://vps.zsoft.cc/' },
          {
            text: version,
            items: [
              { text: 'Changelog', link: '/changelog' },
              { text: 'GitHub Releases', link: 'https://github.com/zsai001/vstats/releases' }
            ]
          }
        ],
        sidebar: {
          '/server/': [
            {
              text: 'Server',
              items: [{ text: 'Overview', link: '/server/' }],
            },
          ],
          '/guide/': [
            {
              text: 'Introduction',
              items: [
                { text: 'What is vStats?', link: '/guide/what-is-vstats' },
                { text: 'Getting Started', link: '/guide/getting-started' },
              ]
            },
            {
              text: 'Installation',
              items: [
                { text: 'Docker', link: '/guide/installation/docker' },
                { text: 'Script', link: '/guide/installation/script' },
                { text: 'Manual', link: '/guide/installation/manual' },
              ]
            },
            {
              text: 'Agent',
              items: [
                { text: 'Install Agent', link: '/guide/agent/install' },
                { text: 'Configuration', link: '/guide/agent/config' },
                { text: 'Troubleshooting', link: '/guide/agent/troubleshooting' },
              ]
            },
            {
              text: 'Configuration',
              items: [
                { text: 'Server Config', link: '/guide/config/server' },
                { text: 'Environment Variables', link: '/guide/config/env' },
                { text: 'Reverse Proxy', link: '/guide/config/reverse-proxy' },
                { text: 'SSL/HTTPS', link: '/guide/config/ssl' },
              ]
            },
            {
              text: 'Administration',
              items: [
                { text: 'Service Management', link: '/guide/admin/service' },
                { text: 'Backup & Restore', link: '/guide/admin/backup' },
                { text: 'Upgrade', link: '/guide/admin/upgrade' },
                { text: 'Uninstall', link: '/guide/admin/uninstall' },
              ]
            }
          ],
          '/cloud/': [
            {
              text: 'Cloud',
              items: [{ text: 'Overview', link: '/cloud/' }],
            },
          ],
          '/cli/': [
            {
              text: 'CLI',
              items: [{ text: 'Overview', link: '/cli/' }],
            },
          ],
          '/reference/': [
            {
              text: 'Reference',
              items: [
                { text: 'REST API', link: '/reference/api' },
                { text: 'WebSocket', link: '/reference/websocket' },
                { text: 'Metrics Schema', link: '/reference/metrics' },
                { text: 'Architecture', link: '/reference/architecture' },
              ]
            }
          ]
        }
      }
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '首页', link: '/zh/' },
          {
            text: '自部署',
            items: [
              { text: '概览', link: '/zh/server/' },
              { text: '指南', link: '/zh/guide/getting-started' },
              { text: '参考', link: '/zh/reference/api' },
            ],
          },
          { text: '云端', link: '/zh/cloud/' },
          { text: '命令行', link: '/zh/cli/' },
          { text: '演示', link: 'https://vps.zsoft.cc/' },
          {
            text: version,
            items: [
              { text: '更新日志', link: '/zh/changelog' },
              { text: 'GitHub Releases', link: 'https://github.com/zsai001/vstats/releases' }
            ]
          }
        ],
        sidebar: {
          '/zh/server/': [
            {
              text: '自部署',
              items: [{ text: '概览', link: '/zh/server/' }],
            },
          ],
          '/zh/guide/': [
            {
              text: '简介',
              items: [
                { text: '什么是 vStats？', link: '/zh/guide/what-is-vstats' },
                { text: '快速开始', link: '/zh/guide/getting-started' },
              ]
            },
            {
              text: '安装',
              items: [
                { text: 'Docker 安装', link: '/zh/guide/installation/docker' },
                { text: '脚本安装', link: '/zh/guide/installation/script' },
                { text: '手动安装', link: '/zh/guide/installation/manual' },
              ]
            },
            {
              text: 'Agent 探针',
              items: [
                { text: '安装 Agent', link: '/zh/guide/agent/install' },
                { text: '配置选项', link: '/zh/guide/agent/config' },
                { text: '故障排除', link: '/zh/guide/agent/troubleshooting' },
              ]
            },
            {
              text: '配置',
              items: [
                { text: '服务器配置', link: '/zh/guide/config/server' },
                { text: '环境变量', link: '/zh/guide/config/env' },
                { text: '反向代理', link: '/zh/guide/config/reverse-proxy' },
                { text: 'SSL/HTTPS', link: '/zh/guide/config/ssl' },
              ]
            },
            {
              text: '管理',
              items: [
                { text: '服务管理', link: '/zh/guide/admin/service' },
                { text: '备份与恢复', link: '/zh/guide/admin/backup' },
                { text: '升级', link: '/zh/guide/admin/upgrade' },
                { text: '卸载', link: '/zh/guide/admin/uninstall' },
              ]
            }
          ],
          '/zh/cloud/': [
            {
              text: '云端',
              items: [{ text: '概览', link: '/zh/cloud/' }],
            },
          ],
          '/zh/cli/': [
            {
              text: '命令行',
              items: [{ text: '概览', link: '/zh/cli/' }],
            },
          ],
          '/zh/reference/': [
            {
              text: '参考',
              items: [
                { text: 'REST API', link: '/zh/reference/api' },
                { text: 'WebSocket', link: '/zh/reference/websocket' },
                { text: '指标结构', link: '/zh/reference/metrics' },
                { text: '架构设计', link: '/zh/reference/architecture' },
              ]
            }
          ]
        },
        outline: {
          label: '页面导航'
        },
        docFooter: {
          prev: '上一页',
          next: '下一页'
        },
        lastUpdated: {
          text: '最后更新于'
        },
        returnToTopLabel: '返回顶部',
        sidebarMenuLabel: '菜单',
        darkModeSwitchLabel: '主题',
        langMenuLabel: '语言'
      }
    }
  },

  themeConfig: {
    // Important: logo will be resolved via withBase() by VitePress theme.
    // Do NOT prefix it with base, otherwise it becomes /base/base/... and breaks.
    logo: '/logo-server.svg',
    siteTitle: 'vStats',

    // Custom fields used by our theme components
    vstats: {
      version,
      mainSiteUrl,
    },
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/zsai001/vstats' },
      { icon: { svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="currentColor" d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/></svg>' }, link: 'https://t.me/zsai010_group/10' },
    ],

    search: {
      provider: 'local',
      options: {
        locales: {
          zh: {
            translations: {
              button: {
                buttonText: '搜索文档',
                buttonAriaLabel: '搜索文档'
              },
              modal: {
                noResultsText: '无法找到相关结果',
                resetButtonTitle: '清除查询条件',
                footer: {
                  selectText: '选择',
                  navigateText: '切换'
                }
              }
            }
          }
        }
      }
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 vStats'
    },

    editLink: {
      pattern: 'https://github.com/zsai001/vstats/edit/main/site/vitepress/docs/:path',
      text: 'Edit this page on GitHub'
    }
  }
})

