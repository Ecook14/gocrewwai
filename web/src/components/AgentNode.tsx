import React from 'react';
import { Handle, Position } from '@xyflow/react';
import { User, Cpu, Zap, MoreVertical } from 'lucide-react';

const AgentNode = ({ data }: { data: any }) => {
  return (
    <div className="w-64 glass border border-blue-500/30 rounded-xl overflow-hidden shadow-2xl shadow-blue-500/10 group hover:border-blue-500/60 transition-all">
      <div className="bg-blue-600/10 p-3 flex items-center justify-between border-b border-blue-500/20">
        <div className="flex items-center gap-2">
           <div className="p-1.5 bg-blue-600 rounded-md">
             <User className="w-3.5 h-3.5 text-white" />
           </div>
           <span className="text-xs font-bold text-white uppercase tracking-wider">Agent Node</span>
        </div>
        <MoreVertical className="w-4 h-4 text-slate-500 cursor-pointer hover:text-white transition-colors" />
      </div>
      
      <div className="p-4 space-y-3">
        <div>
          <label className="text-[9px] font-bold text-blue-500 uppercase tracking-widest mb-1 block">Role</label>
          <div className="text-sm font-semibold text-slate-100">{data.role}</div>
        </div>
        
        <div className="pt-2 flex items-center gap-4">
          <div className="flex items-center gap-1.5">
            <Cpu className="w-3.5 h-3.5 text-slate-500" />
            <span className="text-[10px] text-slate-400 font-medium">GPT-4o</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Zap className="w-3.5 h-3.5 text-amber-500" />
            <span className="text-[10px] text-slate-400 font-medium">Verbose</span>
          </div>
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="w-3 h-3 bg-blue-500 border-2 border-slate-900"
      />
    </div>
  );
};

export default AgentNode;
