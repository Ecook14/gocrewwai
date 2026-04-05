import React from 'react';
import { Rocket, Shield, Activity } from 'lucide-react';

interface HeaderProps {
  onKickoff: () => void;
}

const Header: React.FC<HeaderProps> = ({ onKickoff }) => {
  return (
    <header className="fixed top-0 left-0 right-0 h-16 glass z-50 flex items-center justify-between px-6">
      <div className="flex items-center gap-3">
        <div className="p-2 bg-blue-600 rounded-lg shadow-lg shadow-blue-500/20">
          <Rocket className="w-6 h-6 text-white" />
        </div>
        <div>
          <h1 className="text-xl font-bold tracking-tight text-white">Crew<span className="text-blue-500">-GO</span></h1>
          <p className="text-[10px] text-slate-400 font-medium uppercase tracking-widest">Visual Orchestrator v1</p>
        </div>
      </div>
      
      <div className="flex items-center gap-6">
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Activity className="w-4 h-4 text-emerald-500" />
          <span>Engine: <span className="text-emerald-500 font-mono">CONNECTED</span></span>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Shield className="w-4 h-4 text-blue-500" />
          <span>Security: <span className="text-blue-500 font-mono">SANDBOXED</span></span>
        </div>
        <button 
          onClick={onKickoff}
          className="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 text-white text-sm font-medium rounded-md transition-all shadow-lg shadow-blue-600/20"
        >
          KICKOFF CREW
        </button>
      </div>
    </header>
  );
};

export default Header;
