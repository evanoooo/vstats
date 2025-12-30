/**
 * Batch Update Header Component
 * Shows agent count and batch update controls
 */

import type { RemoteServer } from './types';

export interface BatchUpdateHeaderProps {
  servers: RemoteServer[];
  agentStatus: Record<string, boolean>;
  batchUpdating: boolean;
  batchUpdateProgress: { current: number; total: number; currentServer: string };
  onUpdateAll: (force: boolean) => void;
}

export function BatchUpdateHeader({ 
  servers, 
  agentStatus, 
  batchUpdating, 
  batchUpdateProgress, 
  onUpdateAll 
}: BatchUpdateHeaderProps) {
  const onlineCount = Object.values(agentStatus).filter(Boolean).length;

  return (
    <>
      <div className="flex items-center justify-between mb-4 p-3 rounded-lg bg-white/[0.02] border border-white/5">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-400">
            {servers.length} 个 Agent · {onlineCount} 在线
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onUpdateAll(false)}
            disabled={batchUpdating || onlineCount === 0}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2 ${
              batchUpdating
                ? 'bg-cyan-500/20 text-cyan-400 cursor-wait'
                : onlineCount === 0
                ? 'bg-gray-500/10 text-gray-600 cursor-not-allowed'
                : 'bg-cyan-500/10 hover:bg-cyan-500/20 text-cyan-400 border border-cyan-500/30 hover:border-cyan-500/50'
            }`}
          >
            {batchUpdating ? (
              <>
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                <span>升级中 ({batchUpdateProgress.current}/{batchUpdateProgress.total})</span>
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <span>升级所有</span>
              </>
            )}
          </button>
          <button
            onClick={() => onUpdateAll(true)}
            disabled={batchUpdating || onlineCount === 0}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-all flex items-center gap-2 ${
              batchUpdating
                ? 'bg-amber-500/20 text-amber-400 cursor-wait'
                : onlineCount === 0
                ? 'bg-gray-500/10 text-gray-600 cursor-not-allowed'
                : 'bg-amber-500/10 hover:bg-amber-500/20 text-amber-400 border border-amber-500/30 hover:border-amber-500/50'
            }`}
            title="强制升级会忽略版本检查，重新下载并安装最新版本"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>强制升级</span>
          </button>
        </div>
      </div>

      {/* Batch Update Progress */}
      {batchUpdating && (
        <div className="mb-4 p-3 rounded-lg bg-cyan-500/10 border border-cyan-500/20">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-cyan-400">
              正在升级: <span className="font-medium">{batchUpdateProgress.currentServer}</span>
            </span>
            <span className="text-xs text-cyan-400/70">
              {batchUpdateProgress.current} / {batchUpdateProgress.total}
            </span>
          </div>
          <div className="w-full h-1.5 bg-cyan-500/20 rounded-full overflow-hidden">
            <div
              className="h-full bg-cyan-500 rounded-full transition-all duration-500"
              style={{ width: `${(batchUpdateProgress.current / batchUpdateProgress.total) * 100}%` }}
            />
          </div>
        </div>
      )}
    </>
  );
}

