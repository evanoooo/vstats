/**
 * Settings Page Utility Functions
 */

/**
 * Universal copy to clipboard function that works in all contexts
 * Handles both secure (HTTPS) and non-secure (HTTP/localhost) contexts
 */
export const copyTextToClipboard = async (text: string): Promise<boolean> => {
  try {
    // Try modern clipboard API first (requires secure context)
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
    // Fallback for non-secure contexts (http://localhost, etc.)
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-999999px';
    textArea.style.top = '-999999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    const success = document.execCommand('copy');
    document.body.removeChild(textArea);
    return success;
  } catch (e) {
    console.error('Failed to copy', e);
    // Last resort fallback
    try {
      const textArea = document.createElement('textarea');
      textArea.value = text;
      textArea.style.position = 'fixed';
      textArea.style.left = '-999999px';
      textArea.style.top = '-999999px';
      document.body.appendChild(textArea);
      textArea.focus();
      textArea.select();
      const success = document.execCommand('copy');
      document.body.removeChild(textArea);
      return success;
    } catch {
      return false;
    }
  }
};

/**
 * Format timestamp to locale string
 */
export const formatTimestamp = (ts: string): string => {
  const date = new Date(ts);
  return date.toLocaleString();
};

/**
 * Get category color for audit logs
 */
export const getCategoryColor = (category: string): string => {
  switch (category) {
    case 'auth': return 'bg-blue-500/20 text-blue-400';
    case 'server': return 'bg-emerald-500/20 text-emerald-400';
    case 'settings': return 'bg-purple-500/20 text-purple-400';
    case 'alert': return 'bg-orange-500/20 text-orange-400';
    case 'system': return 'bg-gray-500/20 text-gray-400';
    default: return 'bg-gray-500/20 text-gray-400';
  }
};

/**
 * Get status color for audit logs
 */
export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'success': return 'text-emerald-400';
    case 'error': return 'text-red-400';
    case 'warning': return 'text-amber-400';
    default: return 'text-gray-400';
  }
};

