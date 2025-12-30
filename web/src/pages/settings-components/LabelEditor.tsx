/**
 * Label Editor Component
 * Manages server labels with color support
 */

import { useState } from 'react';
import type { ServerLabel } from './types';
import { LABEL_COLOR_OPTIONS, getLabelColorClasses } from './serverManagementConstants';

export interface LabelEditorProps {
  labels: ServerLabel[];
  onChange: (labels: ServerLabel[]) => void;
  borderColor?: 'emerald' | 'blue';
}

export function LabelEditor({ labels, onChange, borderColor = 'emerald' }: LabelEditorProps) {
  const [newLabelName, setNewLabelName] = useState('');
  const [newLabelColor, setNewLabelColor] = useState('blue');
  
  const borderColorClass = borderColor === 'emerald' ? 'border-emerald-500/10' : 'border-white/5';
  const focusBorderClass = borderColor === 'emerald' ? 'focus:border-emerald-500/50' : 'focus:border-blue-500/50';

  const addLabel = () => {
    if (!newLabelName.trim()) return;
    if (labels.some(l => l.name === newLabelName.trim())) return; // Prevent duplicates
    
    onChange([...labels, { name: newLabelName.trim(), color: newLabelColor }]);
    setNewLabelName('');
  };

  const removeLabel = (index: number) => {
    onChange(labels.filter((_, i) => i !== index));
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addLabel();
    }
  };

  return (
    <div className={`pt-4 border-t ${borderColorClass} mb-4`}>
      <div className="text-xs text-gray-500 uppercase tracking-wider mb-3">标签</div>
      
      {/* Existing Labels */}
      {labels.length > 0 && (
        <div className="flex flex-wrap gap-2 mb-3">
          {labels.map((label, index) => {
            const colorClasses = getLabelColorClasses(label.color);
            return (
              <span
                key={index}
                className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${colorClasses.bg} ${colorClasses.text} border ${colorClasses.border}`}
              >
                {label.name}
                <button
                  onClick={() => removeLabel(index)}
                  className="hover:opacity-70 transition-opacity"
                  type="button"
                >
                  <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </span>
            );
          })}
        </div>
      )}

      {/* Add New Label */}
      <div className="flex gap-2">
        <input
          type="text"
          value={newLabelName}
          onChange={(e) => setNewLabelName(e.target.value)}
          onKeyPress={handleKeyPress}
          className={`flex-1 px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none ${focusBorderClass}`}
          placeholder="输入标签名称..."
        />
        <select
          value={newLabelColor}
          onChange={(e) => setNewLabelColor(e.target.value)}
          className={`px-3 py-2 rounded-lg bg-white/5 border border-white/10 text-white text-sm focus:outline-none ${focusBorderClass}`}
        >
          {LABEL_COLOR_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        <button
          type="button"
          onClick={addLabel}
          disabled={!newLabelName.trim()}
          className={`px-4 py-2 rounded-lg ${borderColor === 'emerald' ? 'bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400' : 'bg-blue-500/10 hover:bg-blue-500/20 text-blue-400'} text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed`}
        >
          添加
        </button>
      </div>
    </div>
  );
}

