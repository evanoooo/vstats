/**
 * Settings Page Constants
 */

import type { BackgroundOption, UnsplashPreset, PlatformOption } from './types';
import type { BackgroundType } from '../../context/ThemeContext';

// Social platform options
export const PLATFORM_OPTIONS: PlatformOption[] = [
  { value: 'github', label: 'GitHub' },
  { value: 'twitter', label: 'Twitter/X' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'discord', label: 'Discord' },
  { value: 'email', label: 'Email' },
  { value: 'website', label: 'Website' },
];

// Background type options
export const BACKGROUND_OPTIONS: BackgroundOption[] = [
  { type: 'gradient' as BackgroundType, name: 'Theme Gradient', nameZh: '主题渐变', icon: '🎨' },
  { type: 'bing' as BackgroundType, name: 'Bing Daily', nameZh: 'Bing 每日壁纸', icon: '🖼️' },
  { type: 'unsplash' as BackgroundType, name: 'Unsplash', nameZh: 'Unsplash 随机', icon: '📷' },
  { type: 'custom' as BackgroundType, name: 'Custom URL', nameZh: '自定义图片', icon: '🔗' },
  { type: 'solid' as BackgroundType, name: 'Solid Color', nameZh: '纯色背景', icon: '🎯' },
];

// Unsplash keyword presets
export const UNSPLASH_PRESETS: UnsplashPreset[] = [
  { query: 'nature,landscape', label: 'Nature', labelZh: '自然风光' },
  { query: 'city,night', label: 'City Night', labelZh: '城市夜景' },
  { query: 'mountains,snow', label: 'Mountains', labelZh: '雪山' },
  { query: 'ocean,beach', label: 'Ocean', labelZh: '海洋' },
  { query: 'forest,trees', label: 'Forest', labelZh: '森林' },
  { query: 'space,galaxy', label: 'Space', labelZh: '宇宙' },
  { query: 'abstract,gradient', label: 'Abstract', labelZh: '抽象' },
  { query: 'minimal,architecture', label: 'Minimal', labelZh: '极简' },
];

// Audit log categories
export const AUDIT_CATEGORIES = [
  { value: '', label: 'All', labelZh: '全部' },
  { value: 'auth', label: 'Auth', labelZh: '认证' },
  { value: 'server', label: 'Server', labelZh: '服务器' },
  { value: 'settings', label: 'Settings', labelZh: '设置' },
  { value: 'alert', label: 'Alert', labelZh: '告警' },
  { value: 'system', label: 'System', labelZh: '系统' },
];

// Get action labels based on language
export const getActionLabels = (isZh: boolean): Record<string, string> => ({
  login: isZh ? '登录' : 'Login',
  login_failed: isZh ? '登录失败' : 'Login Failed',
  logout: isZh ? '登出' : 'Logout',
  password_change: isZh ? '修改密码' : 'Password Change',
  oauth_login: isZh ? 'OAuth 登录' : 'OAuth Login',
  oauth_login_failed: isZh ? 'OAuth 登录失败' : 'OAuth Login Failed',
  server_create: isZh ? '创建服务器' : 'Server Created',
  server_update: isZh ? '更新服务器' : 'Server Updated',
  server_delete: isZh ? '删除服务器' : 'Server Deleted',
  server_upgrade: isZh ? '升级服务器' : 'Server Upgrade',
  agent_register: isZh ? 'Agent 注册' : 'Agent Register',
  agent_connect: isZh ? 'Agent 连接' : 'Agent Connect',
  agent_disconnect: isZh ? 'Agent 断开' : 'Agent Disconnect',
  settings_update: isZh ? '更新设置' : 'Settings Update',
  site_settings_update: isZh ? '更新站点设置' : 'Site Settings Update',
  probe_settings_update: isZh ? '更新探测设置' : 'Probe Settings Update',
  oauth_settings_update: isZh ? '更新 OAuth 设置' : 'OAuth Settings Update',
  local_node_update: isZh ? '更新本地节点' : 'Local Node Update',
  alert_config_update: isZh ? '更新告警配置' : 'Alert Config Update',
  channel_create: isZh ? '创建通知渠道' : 'Channel Created',
  channel_update: isZh ? '更新通知渠道' : 'Channel Updated',
  channel_delete: isZh ? '删除通知渠道' : 'Channel Deleted',
  channel_test: isZh ? '测试通知渠道' : 'Channel Test',
  alert_mute: isZh ? '静音告警' : 'Alert Muted',
  rule_update: isZh ? '更新规则' : 'Rule Updated',
  template_update: isZh ? '更新模板' : 'Template Updated',
  group_create: isZh ? '创建分组' : 'Group Created',
  group_update: isZh ? '更新分组' : 'Group Updated',
  group_delete: isZh ? '删除分组' : 'Group Deleted',
  dimension_create: isZh ? '创建维度' : 'Dimension Created',
  dimension_update: isZh ? '更新维度' : 'Dimension Updated',
  dimension_delete: isZh ? '删除维度' : 'Dimension Deleted',
  option_create: isZh ? '创建选项' : 'Option Created',
  option_update: isZh ? '更新选项' : 'Option Updated',
  option_delete: isZh ? '删除选项' : 'Option Deleted',
  theme_install: isZh ? '安装主题' : 'Theme Install',
  theme_uninstall: isZh ? '卸载主题' : 'Theme Uninstall',
});

