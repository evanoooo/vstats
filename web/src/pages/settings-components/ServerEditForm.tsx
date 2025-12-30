/**
 * Server Edit Form Component
 * Comprehensive form for editing server details
 */

import type { GroupDimension } from '../../types';
import type { EditForm } from './serverManagementTypes';
import { TIP_BADGE_OPTIONS, CURRENCY_OPTIONS } from './serverManagementConstants';
import { LabelEditor } from './LabelEditor';

export interface ServerEditFormProps {
  editForm: EditForm;
  editLoading: boolean;
  editSuccess: boolean;
  editError: string | null;
  dimensions: GroupDimension[];
  onChange: (form: EditForm) => void;
  onSave: () => void;
  onCancel: () => void;
}

export function ServerEditForm({ 
  editForm, 
  editLoading, 
  editSuccess, 
  editError, 
  dimensions, 
  onChange, 
  onSave, 
  onCancel 
}: ServerEditFormProps) {
  return (
    <div className="mt-4 pt-4 border-t border-white/5">
      {editSuccess && (
        <div className="mb-3 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-sm">
          ✓ Server information updated successfully!
        </div>
      )}
      {editError && (
        <div className="mb-3 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          ✗ {editError}
        </div>
      )}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-3">
        <div>
          <label className="block text-xs text-gray-500 mb-1">
            Name <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            value={editForm.name}
            onChange={(e) => onChange({ ...editForm, name: e.target.value })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            placeholder="Server name"
            required
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Location</label>
          <input
            type="text"
            value={editForm.location}
            onChange={(e) => onChange({ ...editForm, location: e.target.value })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            placeholder="e.g., US, CN, HK"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Provider</label>
          <input
            type="text"
            value={editForm.provider}
            onChange={(e) => onChange({ ...editForm, provider: e.target.value })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            placeholder="e.g., AWS, Vultr"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-500 mb-1">Tip Badge</label>
          <select
            value={editForm.tip_badge}
            onChange={(e) => onChange({ ...editForm, tip_badge: e.target.value })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
          >
            {TIP_BADGE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Extended Metadata Section */}
      <div className="pt-3 border-t border-white/5 mb-3">
        <div className="text-xs text-gray-500 uppercase tracking-wider mb-3">Extended Metadata</div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Price Amount</label>
            <input
              type="text"
              value={editForm.price_amount}
              onChange={(e) => onChange({ ...editForm, price_amount: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
              placeholder="e.g., 89.99"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Price Period</label>
            <select
              value={editForm.price_period}
              onChange={(e) => onChange({ ...editForm, price_period: e.target.value as 'month' | 'quarter' | 'year' })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            >
              <option value="month">Monthly / 月付</option>
              <option value="quarter">Quarterly / 季付</option>
              <option value="year">Yearly / 年付</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Currency</label>
            <select
              value={editForm.price_currency}
              onChange={(e) => onChange({ ...editForm, price_currency: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            >
              {CURRENCY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">Purchase Date</label>
            <input
              type="date"
              value={editForm.purchase_date}
              onChange={(e) => onChange({ ...editForm, purchase_date: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            />
          </div>
          <div className="flex items-center">
            <input
              type="checkbox"
              id="auto_renew"
              checked={editForm.auto_renew}
              onChange={(e) => onChange({ ...editForm, auto_renew: e.target.checked })}
              className="w-4 h-4 rounded bg-white/5 border border-white/10 text-blue-500 focus:ring-blue-500"
            />
            <label htmlFor="auto_renew" className="ml-2 text-sm text-gray-400">
              Auto Renew
            </label>
          </div>
        </div>
        <div className="mt-3">
          <label className="block text-xs text-gray-500 mb-1">Notes</label>
          <textarea
            value={editForm.notes}
            onChange={(e) => onChange({ ...editForm, notes: e.target.value })}
            className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50 resize-none"
            placeholder="Additional notes about this server..."
            rows={2}
          />
        </div>
      </div>

      {/* Labels Section */}
      <LabelEditor 
        labels={editForm.labels || []}
        onChange={(labels) => onChange({ ...editForm, labels })}
        borderColor="blue"
      />

      {/* Group Dimensions Selection */}
      {dimensions.length > 0 && (
        <div className="pt-3 border-t border-white/5 mb-3">
          <div className="text-xs text-gray-500 uppercase tracking-wider mb-3">分组标签</div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {dimensions
              .filter((dim) => dim.enabled)
              .sort((a, b) => a.sort_order - b.sort_order)
              .map((dimension) => (
                <div key={dimension.id}>
                  <label className="block text-xs text-gray-400 mb-1">{dimension.name}</label>
                  <select
                    value={editForm.group_values[dimension.id] || ''}
                    onChange={(e) => {
                      const newGroupValues = { ...editForm.group_values };
                      if (e.target.value) {
                        newGroupValues[dimension.id] = e.target.value;
                      } else {
                        delete newGroupValues[dimension.id];
                      }
                      onChange({ ...editForm, group_values: newGroupValues });
                    }}
                    className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
                  >
                    <option value="">-- 未选择 --</option>
                    {dimension.options
                      .sort((a, b) => a.sort_order - b.sort_order)
                      .map((option) => (
                        <option key={option.id} value={option.id}>
                          {option.name}
                        </option>
                      ))}
                  </select>
                </div>
              ))}
          </div>
        </div>
      )}

      {/* Sale/Rent Settings */}
      <div className="pt-3 border-t border-white/5 mb-3">
        <div className="text-xs text-gray-500 uppercase tracking-wider mb-3">合租/出售</div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">状态</label>
            <select
              value={editForm.sale_status}
              onChange={(e) => onChange({ ...editForm, sale_status: e.target.value as '' | 'rent' | 'sell' })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            >
              <option value="">不出售/不合租</option>
              <option value="rent">招租中</option>
              <option value="sell">出售中</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">联系链接</label>
            <input
              type="text"
              value={editForm.sale_contact_url}
              onChange={(e) => onChange({ ...editForm, sale_contact_url: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
              placeholder="https://t.me/xxx 或 mailto:xxx@example.com"
            />
          </div>
        </div>
        {editForm.sale_status && !editForm.sale_contact_url && (
          <p className="mt-2 text-xs text-amber-400">⚠️ 请填写联系链接，否则用户无法联系您</p>
        )}
      </div>

      {/* Traffic Settings */}
      <div className="pt-3 border-t border-white/5 mb-3">
        <div className="text-xs text-gray-500 uppercase tracking-wider mb-3">流量设置</div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">月流量限制 (GB)</label>
            <input
              type="number"
              min="0"
              step="0.01"
              value={editForm.traffic_limit_gb}
              onChange={(e) => onChange({ ...editForm, traffic_limit_gb: e.target.value })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
              placeholder="0 = 不限制"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">计算方式</label>
            <select
              value={editForm.traffic_threshold_type}
              onChange={(e) => onChange({ ...editForm, traffic_threshold_type: e.target.value as 'sum' | 'max' | 'up' | 'down' })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            >
              <option value="sum">双向流量之和</option>
              <option value="max">双向流量较大值</option>
              <option value="up">仅上传流量</option>
              <option value="down">仅下载流量</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">重置日 (每月)</label>
            <select
              value={editForm.traffic_reset_day}
              onChange={(e) => onChange({ ...editForm, traffic_reset_day: parseInt(e.target.value) })}
              className="w-full px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none focus:border-blue-500/50"
            >
              {Array.from({ length: 28 }, (_, i) => i + 1).map((day) => (
                <option key={day} value={day}>
                  {day} 日
                </option>
              ))}
            </select>
          </div>
        </div>
        <p className="mt-2 text-xs text-gray-500">
          💡 设置月流量限制后，可在 Dashboard 查看流量使用情况，并配置超额告警
        </p>
      </div>

      <div className="flex justify-end gap-2">
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-lg bg-white/5 hover:bg-white/10 text-gray-400 text-sm transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onSave}
          disabled={editLoading}
          className="px-4 py-2 rounded-lg bg-blue-500 hover:bg-blue-600 text-white text-sm font-medium transition-colors disabled:opacity-50"
        >
          {editLoading ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  );
}

