/**
 * Dimensions Section
 * Manages group dimensions for server categorization
 */

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { showToast } from '../../components/Toast';
import type { GroupDimension } from '../../types';
import type { RemoteServer } from './types';

export interface DimensionsSectionProps {
  token: string | null;
  servers: RemoteServer[];
}

export function DimensionsSection({ token, servers }: DimensionsSectionProps) {
  const { t } = useTranslation();
  const [dimensions, setDimensions] = useState<GroupDimension[]>([]);
  const [showDimensionsSection, setShowDimensionsSection] = useState(false);
  const [expandedDimension, setExpandedDimension] = useState<string | null>(null);
  const [newOptionName, setNewOptionName] = useState<Record<string, string>>({});
  const [addingOption, setAddingOption] = useState<Record<string, boolean>>({});
  const [editingOption, setEditingOption] = useState<{ dimId: string; optId: string } | null>(null);
  const [editOptionName, setEditOptionName] = useState('');

  // Add/Edit dimension
  const [showAddDimensionForm, setShowAddDimensionForm] = useState(false);
  const [newDimension, setNewDimension] = useState({ name: '', key: '', enabled: true });
  const [addingDimension, setAddingDimension] = useState(false);
  const [editingDimension, setEditingDimension] = useState<string | null>(null);
  const [editDimensionName, setEditDimensionName] = useState('');
  const [deletingDimension, setDeletingDimension] = useState<string | null>(null);

  useEffect(() => {
    fetchDimensions();
  }, [token]);

  const fetchDimensions = async () => {
    try {
      const res = await fetch('/api/dimensions', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (res.ok) {
        const data = await res.json();
        setDimensions(data || []);
      }
    } catch (e) {
      console.error('Failed to fetch dimensions', e);
    }
  };

  const toggleDimensionEnabled = async (dimId: string, enabled: boolean) => {
    try {
      const res = await fetch(`/api/dimensions/${dimId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ enabled }),
      });

      if (res.ok) {
        const updated = await res.json();
        setDimensions(dimensions.map((d) => (d.id === dimId ? updated : d)));
        showToast(enabled ? t('common.enabled') : t('common.disabled'), 'success');
      } else {
        showToast(t('settings.saveFailed'), 'error');
      }
    } catch (e) {
      console.error('Failed to update dimension', e);
      showToast(t('settings.saveFailed'), 'error');
    }
  };

  const addOption = async (dimId: string) => {
    const name = newOptionName[dimId]?.trim();
    if (!name) return;

    setAddingOption({ ...addingOption, [dimId]: true });
    try {
      const dim = dimensions.find((d) => d.id === dimId);
      const res = await fetch(`/api/dimensions/${dimId}/options`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          name,
          sort_order: dim?.options?.length || 0,
        }),
      });

      if (res.ok) {
        const option = await res.json();
        setDimensions(
          dimensions.map((d) => (d.id === dimId ? { ...d, options: [...(d.options || []), option] } : d))
        );
        setNewOptionName({ ...newOptionName, [dimId]: '' });
        showToast(t('settings.saved'), 'success');
      } else {
        showToast(t('settings.saveFailed'), 'error');
      }
    } catch (e) {
      console.error('Failed to add option', e);
      showToast(t('settings.saveFailed'), 'error');
    }
    setAddingOption({ ...addingOption, [dimId]: false });
  };

  const updateOption = async (dimId: string, optId: string) => {
    if (!editOptionName.trim()) return;

    try {
      const res = await fetch(`/api/dimensions/${dimId}/options/${optId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: editOptionName.trim() }),
      });

      if (res.ok) {
        const updated = await res.json();
        setDimensions(
          dimensions.map((d) =>
            d.id === dimId ? { ...d, options: d.options.map((o) => (o.id === optId ? updated : o)) } : d
          )
        );
        setEditingOption(null);
        setEditOptionName('');
        showToast('选项已更新', 'success');
      } else {
        showToast('更新失败', 'error');
      }
    } catch (e) {
      console.error('Failed to update option', e);
      showToast('更新失败', 'error');
    }
  };

  const deleteOption = async (dimId: string, optId: string) => {
    if (!confirm('确定要删除此选项吗？使用此选项的服务器将变为未分配状态。')) {
      return;
    }

    try {
      const res = await fetch(`/api/dimensions/${dimId}/options/${optId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      if (res.ok) {
        setDimensions(
          dimensions.map((d) =>
            d.id === dimId ? { ...d, options: d.options.filter((o) => o.id !== optId) } : d
          )
        );
        showToast('选项已删除', 'success');
      } else {
        showToast('删除失败', 'error');
      }
    } catch (e) {
      console.error('Failed to delete option', e);
      showToast('删除失败', 'error');
    }
  };

  const getOptionServerCount = (dimId: string, optId: string) => {
    return servers.filter((s) => s.group_values?.[dimId] === optId).length;
  };

  const addDimension = async () => {
    if (!newDimension.name.trim() || !newDimension.key.trim()) return;

    setAddingDimension(true);
    try {
      const res = await fetch('/api/dimensions', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          name: newDimension.name.trim(),
          key: newDimension.key.trim().toLowerCase().replace(/\s+/g, '_'),
          enabled: newDimension.enabled,
          sort_order: dimensions.length,
        }),
      });

      if (res.ok) {
        const dimension = await res.json();
        setDimensions([...dimensions, dimension]);
        setNewDimension({ name: '', key: '', enabled: true });
        setShowAddDimensionForm(false);
        showToast('维度已添加', 'success');
      } else {
        const data = await res.json();
        showToast(data.error || '添加失败', 'error');
      }
    } catch (e) {
      console.error('Failed to add dimension', e);
      showToast('添加失败', 'error');
    }
    setAddingDimension(false);
  };

  const updateDimensionName = async (dimId: string) => {
    if (!editDimensionName.trim()) return;

    try {
      const res = await fetch(`/api/dimensions/${dimId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({ name: editDimensionName.trim() }),
      });

      if (res.ok) {
        const updated = await res.json();
        setDimensions(dimensions.map((d) => (d.id === dimId ? updated : d)));
        setEditingDimension(null);
        setEditDimensionName('');
        showToast('维度已更新', 'success');
      } else {
        showToast('更新失败', 'error');
      }
    } catch (e) {
      console.error('Failed to update dimension', e);
      showToast('更新失败', 'error');
    }
  };

  const deleteDimension = async (dimId: string) => {
    const dimension = dimensions.find((d) => d.id === dimId);
    if (!dimension) return;

    const serversUsingDimension = servers.filter((s) => s.group_values?.[dimId]);
    const confirmMessage =
      serversUsingDimension.length > 0
        ? `确定要删除维度"${dimension.name}"吗？\n${serversUsingDimension.length} 台服务器正在使用此维度，删除后将清除其分组设置。`
        : `确定要删除维度"${dimension.name}"吗？`;

    if (!confirm(confirmMessage)) {
      return;
    }

    setDeletingDimension(dimId);
    try {
      const res = await fetch(`/api/dimensions/${dimId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
      });

      if (res.ok) {
        setDimensions(dimensions.filter((d) => d.id !== dimId));
        showToast('维度已删除', 'success');
      } else {
        showToast('删除失败', 'error');
      }
    } catch (e) {
      console.error('Failed to delete dimension', e);
      showToast('删除失败', 'error');
    }
    setDeletingDimension(null);
  };

  return (
    <div className="nezha-card p-6 mb-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold text-white flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-orange-500"></span>
          分组维度
        </h2>
        <div className="flex items-center gap-2">
          {showDimensionsSection && (
            <button
              onClick={() => setShowAddDimensionForm(true)}
              className="px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-sm font-medium transition-colors"
            >
              添加维度
            </button>
          )}
          <button
            onClick={() => setShowDimensionsSection(!showDimensionsSection)}
            className="px-4 py-2 rounded-lg bg-orange-500/10 hover:bg-orange-500/20 text-orange-400 text-sm font-medium transition-colors"
          >
            {showDimensionsSection ? '收起' : '管理'}
          </button>
        </div>
      </div>

      <p className="text-gray-400 text-sm mb-4">
        管理服务器分组维度。启用的维度会显示在 Dashboard 上供筛选分组。
      </p>

      {showDimensionsSection && (
        <div className="space-y-4">
          {/* Add Dimension Form */}
          {showAddDimensionForm && (
            <div className="rounded-lg bg-emerald-500/5 border border-emerald-500/20 p-4 space-y-3">
              <div className="flex items-center gap-2 text-emerald-400 text-sm font-medium">
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
                添加新维度
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">维度名称</label>
                  <input
                    type="text"
                    value={newDimension.name}
                    onChange={(e) => setNewDimension({ ...newDimension, name: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                    placeholder="例如：地区、用途、类型..."
                  />
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">唯一标识 (key)</label>
                  <input
                    type="text"
                    value={newDimension.key}
                    onChange={(e) =>
                      setNewDimension({ ...newDimension, key: e.target.value.toLowerCase().replace(/\s+/g, '_') })
                    }
                    className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-emerald-500/50"
                    placeholder="例如：region、purpose..."
                  />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={newDimension.enabled}
                    onChange={(e) => setNewDimension({ ...newDimension, enabled: e.target.checked })}
                    className="w-4 h-4 rounded border-white/10 bg-white/5 text-emerald-500 focus:ring-emerald-500/20"
                  />
                  <span className="text-sm text-gray-400">创建后立即启用</span>
                </label>
              </div>
              <div className="flex items-center gap-2 pt-2">
                <button
                  onClick={addDimension}
                  disabled={addingDimension || !newDimension.name.trim() || !newDimension.key.trim()}
                  className="px-4 py-2 rounded-lg bg-emerald-500 hover:bg-emerald-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
                >
                  {addingDimension ? '添加中...' : '确认添加'}
                </button>
                <button
                  onClick={() => {
                    setShowAddDimensionForm(false);
                    setNewDimension({ name: '', key: '', enabled: true });
                  }}
                  className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm font-medium transition-colors"
                >
                  取消
                </button>
              </div>
            </div>
          )}

          {dimensions.length === 0 && !showAddDimensionForm ? (
            <div className="text-gray-600 text-sm text-center py-4 border border-dashed border-white/10 rounded-lg">
              暂无分组维度，点击上方"添加维度"创建
            </div>
          ) : (
            <div className="space-y-3">
              {dimensions
                .sort((a, b) => a.sort_order - b.sort_order)
                .map((dimension) => (
                  <div key={dimension.id} className="rounded-lg bg-white/[0.02] border border-white/5 overflow-hidden">
                    {/* Dimension Header */}
                    <div className="flex items-center justify-between p-3">
                      <div className="flex items-center gap-3">
                        <button
                          onClick={() =>
                            setExpandedDimension(expandedDimension === dimension.id ? null : dimension.id)
                          }
                          className="p-1 hover:bg-white/5 rounded transition-colors"
                        >
                          <svg
                            className={`w-4 h-4 text-gray-400 transition-transform ${
                              expandedDimension === dimension.id ? 'rotate-90' : ''
                            }`}
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                          </svg>
                        </button>
                        <div className="w-8 h-8 rounded-lg bg-orange-500/10 border border-orange-500/20 flex items-center justify-center">
                          <svg
                            className="w-4 h-4 text-orange-400"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
                            />
                          </svg>
                        </div>
                        <div className="flex-1">
                          {editingDimension === dimension.id ? (
                            <div className="flex items-center gap-2">
                              <input
                                type="text"
                                value={editDimensionName}
                                onChange={(e) => setEditDimensionName(e.target.value)}
                                className="px-2 py-1 rounded bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-orange-500/50"
                                autoFocus
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter') updateDimensionName(dimension.id);
                                  if (e.key === 'Escape') {
                                    setEditingDimension(null);
                                    setEditDimensionName('');
                                  }
                                }}
                              />
                              <button
                                onClick={() => updateDimensionName(dimension.id)}
                                className="px-2 py-1 rounded bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-xs transition-colors"
                              >
                                保存
                              </button>
                              <button
                                onClick={() => {
                                  setEditingDimension(null);
                                  setEditDimensionName('');
                                }}
                                className="px-2 py-1 rounded bg-white/5 hover:bg-white/10 text-gray-400 text-xs transition-colors"
                              >
                                取消
                              </button>
                            </div>
                          ) : (
                            <>
                              <div className="flex items-center gap-2">
                                <span className="text-white font-medium">{dimension.name}</span>
                                <button
                                  onClick={() => {
                                    setEditingDimension(dimension.id);
                                    setEditDimensionName(dimension.name);
                                  }}
                                  className="p-1 rounded hover:bg-white/5 text-gray-500 hover:text-gray-300 transition-colors"
                                  title="编辑名称"
                                >
                                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                    <path
                                      strokeLinecap="round"
                                      strokeLinejoin="round"
                                      strokeWidth={2}
                                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                                    />
                                  </svg>
                                </button>
                              </div>
                              <div className="text-xs text-gray-500">
                                <span className="text-gray-600">{dimension.key}</span>
                                <span className="mx-2">·</span>
                                {dimension.options?.length || 0} 个选项
                              </div>
                            </>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-3">
                        <label className="flex items-center gap-2 cursor-pointer">
                          <span className={`text-xs ${dimension.enabled ? 'text-emerald-400' : 'text-gray-500'}`}>
                            {dimension.enabled ? '已启用' : '已禁用'}
                          </span>
                          <button
                            onClick={() => toggleDimensionEnabled(dimension.id, !dimension.enabled)}
                            className={`relative w-10 h-5 rounded-full transition-colors ${
                              dimension.enabled ? 'bg-emerald-500' : 'bg-gray-700'
                            }`}
                          >
                            <div
                              className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
                                dimension.enabled ? 'translate-x-5' : ''
                              }`}
                            />
                          </button>
                        </label>
                        <button
                          onClick={() => deleteDimension(dimension.id)}
                          disabled={deletingDimension === dimension.id}
                          className="p-1.5 rounded hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors disabled:opacity-50"
                          title="删除维度"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                          </svg>
                        </button>
                      </div>
                    </div>

                    {/* Dimension Options (Expanded) */}
                    {expandedDimension === dimension.id && (
                      <div className="border-t border-white/5 p-3 bg-black/20">
                        <div className="space-y-2">
                          {/* Add New Option */}
                          <div className="flex items-center gap-2 mb-3">
                            <input
                              type="text"
                              value={newOptionName[dimension.id] || ''}
                              onChange={(e) => setNewOptionName({ ...newOptionName, [dimension.id]: e.target.value })}
                              className="flex-1 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-orange-500/50"
                              placeholder="新选项名称..."
                              onKeyDown={(e) => e.key === 'Enter' && addOption(dimension.id)}
                            />
                            <button
                              onClick={() => addOption(dimension.id)}
                              disabled={addingOption[dimension.id] || !newOptionName[dimension.id]?.trim()}
                              className="px-3 py-1.5 rounded-lg bg-orange-500 hover:bg-orange-600 text-white text-xs font-medium transition-colors disabled:opacity-50"
                            >
                              {addingOption[dimension.id] ? '添加中...' : '添加'}
                            </button>
                          </div>

                          {/* Options List */}
                          {!dimension.options || dimension.options.length === 0 ? (
                            <div className="text-gray-600 text-xs text-center py-3">暂无选项，请添加</div>
                          ) : (
                            <div className="space-y-1">
                              {dimension.options
                                .sort((a, b) => a.sort_order - b.sort_order)
                                .map((option) => (
                                  <div
                                    key={option.id}
                                    className="flex items-center justify-between p-2 rounded-lg bg-white/[0.02] hover:bg-white/[0.04] transition-colors"
                                  >
                                    {editingOption?.dimId === dimension.id && editingOption?.optId === option.id ? (
                                      <div className="flex items-center gap-2 flex-1">
                                        <input
                                          type="text"
                                          value={editOptionName}
                                          onChange={(e) => setEditOptionName(e.target.value)}
                                          className="flex-1 px-2 py-1 rounded bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-orange-500/50"
                                          autoFocus
                                          onKeyDown={(e) => {
                                            if (e.key === 'Enter') updateOption(dimension.id, option.id);
                                            if (e.key === 'Escape') {
                                              setEditingOption(null);
                                              setEditOptionName('');
                                            }
                                          }}
                                        />
                                        <button
                                          onClick={() => updateOption(dimension.id, option.id)}
                                          className="px-2 py-1 rounded bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 text-xs transition-colors"
                                        >
                                          保存
                                        </button>
                                        <button
                                          onClick={() => {
                                            setEditingOption(null);
                                            setEditOptionName('');
                                          }}
                                          className="px-2 py-1 rounded bg-white/5 hover:bg-white/10 text-gray-400 text-xs transition-colors"
                                        >
                                          取消
                                        </button>
                                      </div>
                                    ) : (
                                      <>
                                        <div className="flex items-center gap-2">
                                          <span className="text-sm text-gray-300">{option.name}</span>
                                          <span className="text-xs text-gray-600">
                                            ({getOptionServerCount(dimension.id, option.id)} 台)
                                          </span>
                                        </div>
                                        <div className="flex items-center gap-1">
                                          <button
                                            onClick={() => {
                                              setEditingOption({ dimId: dimension.id, optId: option.id });
                                              setEditOptionName(option.name);
                                            }}
                                            className="p-1.5 rounded hover:bg-blue-500/10 text-gray-500 hover:text-blue-400 transition-colors"
                                            title="编辑"
                                          >
                                            <svg
                                              className="w-3.5 h-3.5"
                                              fill="none"
                                              viewBox="0 0 24 24"
                                              stroke="currentColor"
                                            >
                                              <path
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                strokeWidth={2}
                                                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                                              />
                                            </svg>
                                          </button>
                                          <button
                                            onClick={() => deleteOption(dimension.id, option.id)}
                                            className="p-1.5 rounded hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
                                            title="删除"
                                          >
                                            <svg
                                              className="w-3.5 h-3.5"
                                              fill="none"
                                              viewBox="0 0 24 24"
                                              stroke="currentColor"
                                            >
                                              <path
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                strokeWidth={2}
                                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                                              />
                                            </svg>
                                          </button>
                                        </div>
                                      </>
                                    )}
                                  </div>
                                ))}
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </div>
                ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

