import { formatSpeed } from '../../hooks/useMetrics';

// Convert ISO 3166-1 alpha-2 country code to flag emoji
// Each letter becomes a regional indicator symbol (A=🇦, B=🇧, etc.)
export const getFlag = (code: string | undefined): string | null => {
  if (!code || code.length !== 2) return null;
  const upper = code.toUpperCase();
  const offset = 0x1F1E6 - 65; // 65 is char code for 'A'
  try {
    return String.fromCodePoint(
      upper.charCodeAt(0) + offset,
      upper.charCodeAt(1) + offset
    );
  } catch {
    return null;
  }
};

// Extract currency symbol from price string (e.g., "$89.99" -> "$", "¥199" -> "¥")
export const extractCurrency = (amount: string): string => {
  const match = amount.match(/^[^\d]+/);
  return match ? match[0] : '$';
};

// Format price display consistently
export const formatPrice = (amount: string): string => {
  const currency = extractCurrency(amount);
  const numMatch = amount.match(/[\d.]+/);
  if (!numMatch) return amount;
  const num = parseFloat(numMatch[0]);
  // Always show 2 decimal places for price consistency
  return `${currency}${num.toFixed(2)}`;
};

// Format purchase date to YYYY-MM-DD
export const formatPurchaseDate = (dateStr: string): string => {
  try {
    const date = new Date(dateStr);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  } catch {
    return dateStr;
  }
};

// Format latency to 1 decimal place
export const formatLatency = (ms: number | null): string => {
  if (ms === null) return 'N/A';
  return `${ms.toFixed(1)}ms`;
};

// Calculate expiry date from purchase date and billing period
export const calculateExpiryFromPurchase = (
  purchaseDate?: string,
  period?: 'month' | 'quarter' | 'year'
): string | null => {
  if (!purchaseDate || !period) return null;
  
  try {
    const purchase = new Date(purchaseDate);
    const now = new Date();
    
    if (isNaN(purchase.getTime())) return null;
    
    // Calculate the period length in months
    const periodMonths = period === 'year' ? 12 : period === 'quarter' ? 3 : 1;
    
    // Calculate how many complete periods have passed since purchase
    const monthsDiff = (now.getFullYear() - purchase.getFullYear()) * 12 + 
                       (now.getMonth() - purchase.getMonth());
    const periodsPassed = Math.floor(monthsDiff / periodMonths);
    
    // Calculate the next expiry date
    const expiry = new Date(purchase);
    expiry.setMonth(expiry.getMonth() + (periodsPassed + 1) * periodMonths);
    
    // If the expiry is in the past (edge case), add one more period
    if (expiry < now) {
      expiry.setMonth(expiry.getMonth() + periodMonths);
    }
    
    return expiry.toISOString().split('T')[0];
  } catch {
    return null;
  }
};

// Calculate days until expiry
export const calculateDaysUntilExpiry = (expiryDate?: string): number | null => {
  if (!expiryDate) return null;
  
  try {
    const expiry = new Date(expiryDate);
    const now = new Date();
    const diffTime = expiry.getTime() - now.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    return diffDays;
  } catch {
    return null;
  }
};

// Get expiry status class based on days left
export const getExpiryStatusClass = (daysLeft: number | null): string => {
  if (daysLeft === null) return '';
  if (daysLeft < 0) return 'text-red-500';
  if (daysLeft <= 7) return 'text-red-400';
  if (daysLeft <= 30) return 'text-amber-400';
  return 'text-gray-400';
};

// Format expiry display
export const formatExpiryDisplay = (daysLeft: number | null, autoRenew?: boolean): string => {
  if (daysLeft === null) return '';
  if (daysLeft < 0) return autoRenew ? `已过期 ${Math.abs(daysLeft)}天` : `过期 ${Math.abs(daysLeft)}天`;
  if (daysLeft === 0) return '今天到期';
  if (daysLeft === 1) return '明天到期';
  if (daysLeft <= 7) return `${daysLeft}天后到期`;
  if (daysLeft <= 30) return `${daysLeft}天`;
  return `${daysLeft}天`;
};

// Calculate remaining value based on price and purchase date
export const calculateRemainingValue = (price?: { amount: string; period: 'month' | 'quarter' | 'year' }, purchaseDate?: string): string | null => {
  if (!price || !purchaseDate) return null;
  
  try {
    // Extract currency symbol and numeric value
    const currency = extractCurrency(price.amount);
    const priceMatch = price.amount.match(/[\d.]+/);
    if (!priceMatch) return null;
    
    const priceValue = parseFloat(priceMatch[0]);
    if (isNaN(priceValue)) return null;
    
    const purchase = new Date(purchaseDate);
    const now = new Date();
    
    if (purchase > now) return null; // Invalid date
    
    // Calculate days elapsed
    const daysElapsed = Math.floor((now.getTime() - purchase.getTime()) / (1000 * 60 * 60 * 24));
    
    if (price.period === 'month') {
      // Monthly billing: calculate based on days in month
      const daysInMonth = 30; // Approximate
      const monthsElapsed = daysElapsed / daysInMonth;
      const remainingMonths = Math.max(0, 1 - monthsElapsed);
      const remainingValue = priceValue * remainingMonths;
      
      if (remainingValue <= 0) return null;
      return `${currency}${remainingValue.toFixed(2)}`;
    } else if (price.period === 'quarter') {
      // Quarterly billing: calculate based on days in quarter (90 days)
      const daysInQuarter = 90; // Approximate (3 months)
      const quartersElapsed = daysElapsed / daysInQuarter;
      const remainingQuarters = Math.max(0, 1 - quartersElapsed);
      const remainingValue = priceValue * remainingQuarters;
      
      if (remainingValue <= 0) return null;
      return `${currency}${remainingValue.toFixed(2)}`;
    } else if (price.period === 'year') {
      // Yearly billing: calculate based on days in year
      const daysInYear = 365;
      const yearsElapsed = daysElapsed / daysInYear;
      const remainingYears = Math.max(0, 1 - yearsElapsed);
      const remainingValue = priceValue * remainingYears;
      
      if (remainingValue <= 0) return null;
      return `${currency}${remainingValue.toFixed(2)}`;
    }
  } catch (e) {
    console.error('Failed to calculate remaining value', e);
  }
  
  return null;
};

export const getShortCpuBrand = (brand: string) => {
  return brand
    .replace(/\(R\)|\(TM\)|CPU|Processor|@.*$/gi, '')
    .replace(/Intel Core |AMD Ryzen |AMD EPYC |Intel Xeon /gi, '')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 24);
};

export const formatDiskSize = (bytes: number) => {
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  const tb = gb * 1024;
  
  if (bytes >= tb) return `${(bytes / tb).toFixed(0)}T`;
  if (bytes >= gb) return `${(bytes / gb).toFixed(0)}G`;
  if (bytes >= mb) return `${(bytes / mb).toFixed(0)}M`;
  return `${(bytes / kb).toFixed(0)}K`;
};

export const getResourceState = (value: number, thresholds: [number, number]): 'ok' | 'warn' | 'bad' => {
  if (value > thresholds[1]) return 'bad';
  if (value > thresholds[0]) return 'warn';
  return 'ok';
};

// Format traffic (total bytes transferred)
export const formatTraffic = (bytes: number): string => {
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  const tb = gb * 1024;
  
  if (bytes >= tb) return `${(bytes / tb).toFixed(2)}T`;
  if (bytes >= gb) return `${(bytes / gb).toFixed(2)}G`;
  if (bytes >= mb) return `${(bytes / mb).toFixed(0)}M`;
  if (bytes >= kb) return `${(bytes / kb).toFixed(0)}K`;
  return `${bytes}B`;
};

// Format uptime as days
export const formatUptimeDays = (seconds: number, t: (key: string) => string): string => {
  const days = Math.floor(seconds / 86400);
  return `${days} ${t('dashboard.days')}`;
};

// Get short OS name
export const getShortOsName = (osName: string): string => {
  const name = osName.toLowerCase();
  if (name.includes('ubuntu')) return 'Ubuntu';
  if (name.includes('debian')) return 'Debian';
  if (name.includes('centos')) return 'CentOS';
  if (name.includes('rocky')) return 'Rocky';
  if (name.includes('alma')) return 'AlmaLinux';
  if (name.includes('fedora')) return 'Fedora';
  if (name.includes('arch')) return 'Arch';
  if (name.includes('windows')) return 'Windows';
  if (name.includes('macos') || name.includes('darwin')) return 'macOS';
  return osName.split(' ')[0] || 'Linux';
};

// Tip badge mapping
export const getTipBadgeClass = (tag?: string) => {
  if (!tag) return null;
  const tagLower = tag.toLowerCase();
  if (tagLower.includes('cn3-opt') || tagLower.includes('三网优化')) return 'cn3-opt';
  if (tagLower.includes('cn3-gia') || tagLower.includes('三网gia')) return 'cn3-gia';
  if (tagLower.includes('big-disk') || tagLower.includes('大盘')) return 'big-disk';
  if (tagLower.includes('perf') || tagLower.includes('性能')) return 'perf';
  if (tagLower.includes('landing') || tagLower.includes('落地')) return 'landing';
  if (tagLower.includes('dufu') || tagLower.includes('杜甫')) return 'dufu';
  return null;
};

export const getTipBadgeLabel = (tag: string | undefined, t: (key: string) => string) => {
  if (!tag) return null;
  const badgeClass = getTipBadgeClass(tag);
  if (badgeClass) return t(`dashboard.tipBadge.${badgeClass}`);
  return null;
};

// Label color classes for displaying labels on dashboard
export const LABEL_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  red: { bg: 'bg-red-500/20', text: 'text-red-400', border: 'border-red-500/30' },
  orange: { bg: 'bg-orange-500/20', text: 'text-orange-400', border: 'border-orange-500/30' },
  amber: { bg: 'bg-amber-500/20', text: 'text-amber-400', border: 'border-amber-500/30' },
  yellow: { bg: 'bg-yellow-500/20', text: 'text-yellow-400', border: 'border-yellow-500/30' },
  lime: { bg: 'bg-lime-500/20', text: 'text-lime-400', border: 'border-lime-500/30' },
  green: { bg: 'bg-green-500/20', text: 'text-green-400', border: 'border-green-500/30' },
  emerald: { bg: 'bg-emerald-500/20', text: 'text-emerald-400', border: 'border-emerald-500/30' },
  teal: { bg: 'bg-teal-500/20', text: 'text-teal-400', border: 'border-teal-500/30' },
  cyan: { bg: 'bg-cyan-500/20', text: 'text-cyan-400', border: 'border-cyan-500/30' },
  sky: { bg: 'bg-sky-500/20', text: 'text-sky-400', border: 'border-sky-500/30' },
  blue: { bg: 'bg-blue-500/20', text: 'text-blue-400', border: 'border-blue-500/30' },
  indigo: { bg: 'bg-indigo-500/20', text: 'text-indigo-400', border: 'border-indigo-500/30' },
  violet: { bg: 'bg-violet-500/20', text: 'text-violet-400', border: 'border-violet-500/30' },
  purple: { bg: 'bg-purple-500/20', text: 'text-purple-400', border: 'border-purple-500/30' },
  fuchsia: { bg: 'bg-fuchsia-500/20', text: 'text-fuchsia-400', border: 'border-fuchsia-500/30' },
  pink: { bg: 'bg-pink-500/20', text: 'text-pink-400', border: 'border-pink-500/30' },
  rose: { bg: 'bg-rose-500/20', text: 'text-rose-400', border: 'border-rose-500/30' },
};

export const getLabelColorClasses = (color: string) => {
  return LABEL_COLORS[color] || LABEL_COLORS.blue;
};

// Re-export formatSpeed from hooks
export { formatSpeed };

