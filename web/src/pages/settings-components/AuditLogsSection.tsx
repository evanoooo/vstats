/**
 * Audit Logs Section Component
 * Displays and manages audit logs with filtering and export capabilities
 */

import { useState, useEffect, useCallback } from 'react';
import { showToast } from '../../components/Toast';
import { AUDIT_CATEGORIES, getActionLabels } from './constants';
import { formatTimestamp, getCategoryColor, getStatusColor } from './utils';
import type { AuditLog, AuditLogStats, AuditLogsSectionProps } from './types';

export function AuditLogsSection({ token, isZh }: AuditLogsSectionProps) {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<AuditLogStats | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit] = useState(20);
  const [categoryFilter, setCategoryFilter] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [exporting, setExporting] = useState(false);

  const categories = AUDIT_CATEGORIES.map(cat => ({
    value: cat.value,
    label: isZh ? cat.labelZh : cat.label
  }));

  const actionLabels = getActionLabels(isZh);

  const fetchLogs = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });
      if (categoryFilter) params.append('category', categoryFilter);
      if (searchQuery) params.append('search', searchQuery);

      const res = await fetch(`/api/audit-logs?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setLogs(data.logs || []);
        setTotal(data.total || 0);
      }
    } catch (e) {
      console.error('Failed to fetch audit logs:', e);
    }
    setLoading(false);
  }, [token, page, limit, categoryFilter, searchQuery]);

  const fetchStats = useCallback(async () => {
    if (!token) return;
    try {
      const res = await fetch('/api/audit-logs/stats', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setStats(data);
      }
    } catch (e) {
      console.error('Failed to fetch audit stats:', e);
    }
  }, [token]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const handleExport = async (format: 'json' | 'csv') => {
    if (!token) return;
    setExporting(true);
    try {
      const params = new URLSearchParams({ format });
      if (categoryFilter) params.append('category', categoryFilter);
      
      const res = await fetch(`/api/audit-logs/export?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      
      if (res.ok) {
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.${format}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        showToast(isZh ? '导出成功' : 'Export successful', 'success');
      }
    } catch (e) {
      console.error('Failed to export:', e);
      showToast(isZh ? '导出失败' : 'Export failed', 'error');
    }
    setExporting(false);
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <div data-section="audit-logs" className="nezha-card p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-slate-500"></span>
          {isZh ? '审计日志' : 'Audit Logs'}
        </h2>
        <div className="flex items-center gap-2">
          <button
            onClick={() => handleExport('csv')}
            disabled={exporting}
            className="px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-xs transition-colors disabled:opacity-50"
          >
            {isZh ? '导出 CSV' : 'Export CSV'}
          </button>
          <button
            onClick={() => handleExport('json')}
            disabled={exporting}
            className="px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-xs transition-colors disabled:opacity-50"
          >
            {isZh ? '导出 JSON' : 'Export JSON'}
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-white/5 rounded-lg p-4 border border-white/10">
            <div className="text-2xl font-bold text-white">{stats.total}</div>
            <div className="text-xs text-gray-500">{isZh ? '总记录' : 'Total Records'}</div>
          </div>
          <div className="bg-white/5 rounded-lg p-4 border border-white/10">
            <div className="text-2xl font-bold text-blue-400">{stats.today}</div>
            <div className="text-xs text-gray-500">{isZh ? '今日记录' : 'Today'}</div>
          </div>
          <div className="bg-white/5 rounded-lg p-4 border border-white/10">
            <div className="text-2xl font-bold text-red-400">{stats.errors}</div>
            <div className="text-xs text-gray-500">{isZh ? '错误' : 'Errors'}</div>
          </div>
          <div className="bg-white/5 rounded-lg p-4 border border-white/10">
            <div className="text-2xl font-bold text-emerald-400">
              {Object.keys(stats.by_category).length}
            </div>
            <div className="text-xs text-gray-500">{isZh ? '分类' : 'Categories'}</div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap gap-4 mb-6">
        <select
          value={categoryFilter}
          onChange={(e) => { setCategoryFilter(e.target.value); setPage(1); }}
          className="px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-slate-500/50"
        >
          {categories.map(cat => (
            <option key={cat.value} value={cat.value} className="bg-gray-900">{cat.label}</option>
          ))}
        </select>
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { setPage(1); fetchLogs(); } }}
          placeholder={isZh ? '搜索 IP、目标名称...' : 'Search IP, target name...'}
          className="flex-1 min-w-[200px] px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-slate-500/50"
        />
        <button
          onClick={() => { setPage(1); fetchLogs(); }}
          className="px-4 py-2 rounded-lg bg-slate-500/20 hover:bg-slate-500/30 text-slate-400 text-sm transition-colors"
        >
          {isZh ? '搜索' : 'Search'}
        </button>
      </div>

      {/* Logs Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <svg className="animate-spin h-8 w-8 text-slate-400" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        </div>
      ) : logs.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          {isZh ? '暂无审计日志' : 'No audit logs found'}
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-white/10">
                <th className="pb-3 font-medium">{isZh ? '时间' : 'Time'}</th>
                <th className="pb-3 font-medium">{isZh ? '操作' : 'Action'}</th>
                <th className="pb-3 font-medium">{isZh ? '分类' : 'Category'}</th>
                <th className="pb-3 font-medium">{isZh ? '目标' : 'Target'}</th>
                <th className="pb-3 font-medium">IP</th>
                <th className="pb-3 font-medium">{isZh ? '状态' : 'Status'}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {logs.map((log) => (
                <tr key={log.id} className="hover:bg-white/5 transition-colors">
                  <td className="py-3 text-gray-400 whitespace-nowrap">
                    {formatTimestamp(log.timestamp)}
                  </td>
                  <td className="py-3 text-white">
                    {actionLabels[log.action] || log.action}
                  </td>
                  <td className="py-3">
                    <span className={`px-2 py-1 rounded text-xs border ${getCategoryColor(log.category)}`}>
                      {log.category}
                    </span>
                  </td>
                  <td className="py-3 text-gray-300">
                    {log.target_name || log.target_id || '-'}
                  </td>
                  <td className="py-3 text-gray-500 font-mono text-xs">
                    {log.user_ip}
                  </td>
                  <td className="py-3">
                    <span className={`font-medium ${getStatusColor(log.status)}`}>
                      {log.status === 'success' ? '✓' : '✗'}
                    </span>
                    {log.error_message && (
                      <span className="ml-2 text-red-400 text-xs">{log.error_message}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between mt-6 pt-4 border-t border-white/10">
          <div className="text-sm text-gray-500">
            {isZh ? `共 ${total} 条记录` : `${total} records total`}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(Math.max(1, page - 1))}
              disabled={page === 1}
              className="px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm transition-colors disabled:opacity-50"
            >
              {isZh ? '上一页' : 'Prev'}
            </button>
            <span className="text-gray-400 text-sm">
              {page} / {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages, page + 1))}
              disabled={page === totalPages}
              className="px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm transition-colors disabled:opacity-50"
            >
              {isZh ? '下一页' : 'Next'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default AuditLogsSection;

