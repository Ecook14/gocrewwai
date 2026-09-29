import React from 'react';
import { Rocket, Shield, Activity, KeyRound } from 'lucide-react';
import type { StreamStatus } from '../hooks/useSSE';

interface HeaderProps {
  onKickoff: () => void;
  kickoffBusy: boolean;
  kickoffHint: string | null;
  token: string;
  onTokenChange: (value: string) => void;
  streamStatus: StreamStatus;
  sessionId: string | null;
}

const STATUS_LABEL: Record<StreamStatus, { text: string; className: string }> = {
  connected: { text: 'STREAMING', className: 'text-emerald-500' },
  connecting: { text: 'CONNECTING', className: 'text-amber-500' },
  disconnected: { text: 'IDLE', className: 'text-slate-500' },
  unauthorized: { text: 'BAD TOKEN', className: 'text-red-500' },
};

const Header: React.FC<HeaderProps> = ({
  onKickoff,
  kickoffBusy,
  kickoffHint,
  token,
  onTokenChange,
  streamStatus,
  sessionId,
}) => {
  const status = STATUS_LABEL[streamStatus];
  return (
    <header className="fixed top-0 left-0 right-0 h-16 glass z-50 flex items-center justify-between px-6">
      <div className="flex items-center gap-3">
        <div className="p-2 bg-blue-600 rounded-lg shadow-lg shadow-blue-500/20">
          <Rocket className="w-6 h-6 text-white" />
        </div>
        <div>
          <h1 className="text-xl font-bold tracking-tight text-white">
            Crew<span className="text-blue-500">-GO</span>
          </h1>
          <p className="text-[10px] text-slate-400 font-medium uppercase tracking-widest">
            Visual Orchestrator
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <label className="flex items-center gap-2 text-xs text-slate-400">
          <KeyRound className="w-4 h-4 text-slate-500" />
          <input
            type="password"
            value={token}
            onChange={(e) => onTokenChange(e.target.value)}
            placeholder="API token"
            autoComplete="off"
            spellCheck={false}
            className="w-44 bg-slate-800/70 border border-slate-700/60 rounded-md px-2 py-1 text-xs font-mono text-slate-200 placeholder:text-slate-600 focus:outline-none focus:border-blue-500/60"
          />
        </label>
        <div className="flex items-center gap-2 text-xs text-slate-400" title={sessionId ? `Session ${sessionId}` : 'No active session'}>
          <Activity className="w-4 h-4 text-emerald-500" />
          <span>
            Engine:{' '}
            <span className={`${status.className} font-mono`}>{status.text}</span>
          </span>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-400" title="Requests carry your bearer token; sessions are owner-scoped server-side">
          <Shield className="w-4 h-4 text-blue-500" />
          <span>
            Auth: <span className="text-blue-500 font-mono">{token ? 'TOKEN SET' : 'NO TOKEN'}</span>
          </span>
        </div>
        <button
          onClick={onKickoff}
          disabled={kickoffBusy}
          title={kickoffHint ?? 'Connect a task node to an agent node first'}
          className="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 disabled:bg-slate-700 disabled:text-slate-400 text-white text-sm font-medium rounded-md transition-all shadow-lg shadow-blue-600/20"
        >
          {kickoffBusy ? 'STARTING…' : 'KICKOFF CREW'}
        </button>
      </div>
    </header>
  );
};

export default Header;
