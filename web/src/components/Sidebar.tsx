import React from 'react';
import { UserPlus, FilePlus, Settings, Database, Cpu } from 'lucide-react';

interface SidebarProps {
  onAddNode: (type: 'agentNode' | 'taskNode') => void;
}

const Sidebar: React.FC<SidebarProps> = ({ onAddNode }) => {
  return (
    <aside className="w-72 h-full bg-slate-900 border-r border-slate-800 flex flex-col p-6 gap-8">
      <div>
        <h3 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-4">Node Inventory</h3>
        <div className="grid grid-cols-1 gap-3">
          <button 
            onClick={() => onAddNode('agentNode')}
            className="flex items-center gap-3 p-3 bg-slate-800/50 hover:bg-slate-800 border border-slate-700/50 hover:border-blue-500/50 rounded-xl transition-all group"
          >
            <div className="p-2 bg-blue-600/10 group-hover:bg-blue-600/20 rounded-lg">
              <UserPlus className="w-5 h-5 text-blue-500" />
            </div>
            <div className="text-left">
              <p className="text-sm font-semibold text-slate-200">New Agent</p>
              <p className="text-[10px] text-slate-500 tracking-tight">Autonomous persona</p>
            </div>
          </button>
          
          <button 
             onClick={() => onAddNode('taskNode')}
            className="flex items-center gap-3 p-3 bg-slate-800/50 hover:bg-slate-800 border border-slate-700/50 hover:border-emerald-500/50 rounded-xl transition-all group"
          >
            <div className="p-2 bg-emerald-600/10 group-hover:bg-emerald-600/20 rounded-lg">
              <FilePlus className="w-5 h-5 text-emerald-500" />
            </div>
            <div className="text-left">
              <p className="text-sm font-semibold text-slate-200">New Task</p>
              <p className="text-[10px] text-slate-500 tracking-tight">Logical unit of work</p>
            </div>
          </button>
        </div>
      </div>

      <div>
        <h3 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-4">Configurations</h3>
        <div className="space-y-2">
           <div className="flex items-center justify-between text-xs p-2 hover:bg-slate-800/50 rounded-lg cursor-pointer transition-all">
             <div className="flex items-center gap-2 text-slate-300">
               <Database className="w-4 h-4 text-slate-500" />
               <span>Memory Store</span>
             </div>
             <span className="text-[10px] bg-blue-900/30 text-blue-400 px-1.5 py-0.5 rounded">REDIS</span>
           </div>
           <div className="flex items-center justify-between text-xs p-2 hover:bg-slate-800/50 rounded-lg cursor-pointer transition-all">
             <div className="flex items-center gap-2 text-slate-300">
               <Cpu className="w-4 h-4 text-slate-500" />
               <span>Inter-Agent Mesh</span>
             </div>
             <span className="text-[10px] bg-emerald-900/30 text-emerald-400 px-1.5 py-0.5 rounded">gRPC</span>
           </div>
           <div className="flex items-center gap-2 text-xs p-2 hover:bg-slate-800/50 rounded-lg text-slate-300 cursor-pointer transition-all">
             <Settings className="w-4 h-4 text-slate-500" />
             <span>Global Engine Rules</span>
           </div>
        </div>
      </div>
      
      <div className="mt-auto pt-6 border-t border-slate-800">
        <div className="bg-slate-800/30 p-4 rounded-xl">
           <div className="flex items-center justify-between mb-2">
             <h4 className="text-[10px] font-bold text-slate-400 uppercase tracking-widest">Real-time Usage</h4>
             <span className="text-[10px] text-emerald-500">$0.12 total</span>
           </div>
           <div className="w-full bg-slate-700/50 h-1 rounded-full overflow-hidden">
             <div className="bg-blue-600 h-full w-1/4 rounded-full"></div>
           </div>
        </div>
      </div>
    </aside>
  );
};

export default Sidebar;
