/**
 * Server Management Constants
 */

// Tip badge options for server cards
export const TIP_BADGE_OPTIONS = [
  { value: '', label: 'Auto (from tag)' },
  { value: 'cn3-opt', label: '三网优化' },
  { value: 'cn3-gia', label: '三网 GIA' },
  { value: 'big-disk', label: '大盘鸡' },
  { value: 'perf', label: '性能机' },
  { value: 'landing', label: '落地机' },
  { value: 'dufu', label: '杜甫' },
];

// Currency options
export const CURRENCY_OPTIONS = [
  { value: 'USD', label: 'USD ($)' },
  { value: 'CNY', label: 'CNY (¥)' },
  { value: 'EUR', label: 'EUR (€)' },
  { value: 'GBP', label: 'GBP (£)' },
  { value: 'JPY', label: 'JPY (¥)' },
  { value: 'KRW', label: 'KRW (₩)' },
  { value: 'RUB', label: 'RUB (₽)' },
];

// Label color options for server labels
export interface LabelColorOption {
  value: string;
  label: string;
  bg: string;
  text: string;
  border: string;
}

export const LABEL_COLOR_OPTIONS: LabelColorOption[] = [
  { value: 'red', label: '红色', bg: 'bg-red-500/20', text: 'text-red-400', border: 'border-red-500/30' },
  { value: 'orange', label: '橙色', bg: 'bg-orange-500/20', text: 'text-orange-400', border: 'border-orange-500/30' },
  { value: 'amber', label: '琥珀', bg: 'bg-amber-500/20', text: 'text-amber-400', border: 'border-amber-500/30' },
  { value: 'yellow', label: '黄色', bg: 'bg-yellow-500/20', text: 'text-yellow-400', border: 'border-yellow-500/30' },
  { value: 'lime', label: '青柠', bg: 'bg-lime-500/20', text: 'text-lime-400', border: 'border-lime-500/30' },
  { value: 'green', label: '绿色', bg: 'bg-green-500/20', text: 'text-green-400', border: 'border-green-500/30' },
  { value: 'emerald', label: '翠绿', bg: 'bg-emerald-500/20', text: 'text-emerald-400', border: 'border-emerald-500/30' },
  { value: 'teal', label: '青色', bg: 'bg-teal-500/20', text: 'text-teal-400', border: 'border-teal-500/30' },
  { value: 'cyan', label: '青蓝', bg: 'bg-cyan-500/20', text: 'text-cyan-400', border: 'border-cyan-500/30' },
  { value: 'sky', label: '天蓝', bg: 'bg-sky-500/20', text: 'text-sky-400', border: 'border-sky-500/30' },
  { value: 'blue', label: '蓝色', bg: 'bg-blue-500/20', text: 'text-blue-400', border: 'border-blue-500/30' },
  { value: 'indigo', label: '靛蓝', bg: 'bg-indigo-500/20', text: 'text-indigo-400', border: 'border-indigo-500/30' },
  { value: 'violet', label: '紫罗兰', bg: 'bg-violet-500/20', text: 'text-violet-400', border: 'border-violet-500/30' },
  { value: 'purple', label: '紫色', bg: 'bg-purple-500/20', text: 'text-purple-400', border: 'border-purple-500/30' },
  { value: 'fuchsia', label: '洋红', bg: 'bg-fuchsia-500/20', text: 'text-fuchsia-400', border: 'border-fuchsia-500/30' },
  { value: 'pink', label: '粉色', bg: 'bg-pink-500/20', text: 'text-pink-400', border: 'border-pink-500/30' },
  { value: 'rose', label: '玫瑰', bg: 'bg-rose-500/20', text: 'text-rose-400', border: 'border-rose-500/30' },
];

// Get label color classes by color value
export const getLabelColorClasses = (color: string): LabelColorOption => {
  const option = LABEL_COLOR_OPTIONS.find(o => o.value === color);
  return option || LABEL_COLOR_OPTIONS[0];
};

