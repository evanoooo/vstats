/**
 * Server Card Component
 * Displays server information with edit, update, and delete actions
 */

import type { GroupDimension } from '../../types';
import type { RemoteServer } from './types';
import type { EditForm } from './serverManagementTypes';
import { ServerEditForm } from './ServerEditForm';

export interface ServerCardProps {
  server: RemoteServer;
  isOnline: boolean;
  isUpdating: boolean;
  editingServer: string | null;
  editForm: EditForm;
  editLoading: boolean;
  editSuccess: boolean;
  editError: string | null;
  copiedToken: string | null;
  dimensions: GroupDimension[];
  onEdit: (server: RemoteServer) => void;
  onSaveEdit: () => void;
  onCancelEdit: () => void;
  onEditFormChange: (form: EditForm) => void;
  onUpdate: (serverId: string, force: boolean) => void;
  onDelete: (serverId: string) => void;
  onCopyToken: (token: string) => void;
}

export function ServerCard({
  server,
  isOnline,
  isUpdating,
  editingServer,
  editForm,
  editLoading,
  editSuccess,
  editError,
  copiedToken,
  dimensions,
  onEdit,
  onSaveEdit,
  onCancelEdit,
  onEditFormChange,
  onUpdate,
  onDelete,
  onCopyToken,
}: ServerCardProps) {
  return (
    <div className="p-4 rounded-xl bg-white/[0.02] border border-white/5 hover:border-white/10 transition-colors">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-sm font-bold text-blue-400">
            {server.location || '??'}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-medium text-white">{server.name}</span>
              <span className="px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 text-[10px] font-bold uppercase">
                Agent
              </span>
              <span
                className={`w-2 h-2 rounded-full ${
                  isOnline ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]' : 'bg-gray-500'
                }`}
              />
              <span className={`text-xs ${isOnline ? 'text-emerald-400' : 'text-gray-500'}`}>
                {isOnline ? 'Online' : 'Offline'}
              </span>
            </div>
            <div className="flex items-center gap-2 mt-1 flex-wrap">
              <div className="text-xs text-gray-700 dark:text-gray-500 font-mono">ID: {server.id.slice(0, 8)}...</div>
              {server.ip && <span className="text-xs text-cyan-400 font-mono">{server.ip}</span>}
              {server.version && <span className="text-xs text-gray-600 font-mono">v{server.version}</span>}
              {server.tag && (
                <span className="px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-400 text-xs">{server.tag}</span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {server.provider && (
            <span className="px-2 py-1 rounded bg-amber-500/10 text-amber-400 text-xs">{server.provider}</span>
          )}
          <button
            onClick={() => onEdit(server)}
            className="p-2 rounded-lg hover:bg-blue-500/10 text-gray-500 hover:text-blue-400 transition-colors"
            title="Edit Server"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            onClick={() => onUpdate(server.id, false)}
            disabled={!isOnline || isUpdating}
            className={`px-2 py-1 rounded-lg text-xs font-medium transition-colors ${
              isOnline && !isUpdating
                ? 'hover:bg-cyan-500/10 text-gray-500 hover:text-cyan-400 border border-transparent hover:border-cyan-500/30'
                : 'text-gray-600 cursor-not-allowed'
            }`}
            title={isOnline ? '升级 Agent（如果已是最新版本会跳过）' : 'Agent 离线'}
          >
            {isUpdating ? '升级中...' : '升级'}
          </button>
          <button
            onClick={() => onUpdate(server.id, true)}
            disabled={!isOnline || isUpdating}
            className={`px-2 py-1 rounded-lg text-xs font-medium transition-colors ${
              isOnline && !isUpdating
                ? 'hover:bg-amber-500/10 text-gray-500 hover:text-amber-400 border border-transparent hover:border-amber-500/30'
                : 'text-gray-600 cursor-not-allowed'
            }`}
            title={isOnline ? '强制升级（忽略版本检查，重新下载安装）' : 'Agent 离线'}
          >
            强制
          </button>
          <button
            onClick={() => onDelete(server.id)}
            className="p-2 rounded-lg hover:bg-red-500/10 text-gray-500 hover:text-red-400 transition-colors"
            title="Delete Server"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      {/* Edit Form */}
      {editingServer === server.id && (
        <ServerEditForm
          editForm={editForm}
          editLoading={editLoading}
          editSuccess={editSuccess}
          editError={editError}
          dimensions={dimensions}
          onChange={onEditFormChange}
          onSave={onSaveEdit}
          onCancel={onCancelEdit}
        />
      )}

      {server.token && (
        <div className="mt-3 pt-3 border-t border-white/5">
          <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1">Agent Token</div>
          <div className="flex items-center gap-2">
            <code className="flex-1 px-2 py-1 rounded bg-black/20 text-xs text-emerald-400 font-mono truncate">
              {server.token}
            </code>
            <button
              onClick={() => onCopyToken(server.token || '')}
              className={`px-2 py-1 rounded text-xs transition-colors ${
                copiedToken === server.token
                  ? 'bg-emerald-500/20 text-emerald-400'
                  : 'bg-white/5 hover:bg-white/10 text-gray-400 hover:text-white'
              }`}
            >
              {copiedToken === server.token ? 'Copied!' : 'Copy'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

