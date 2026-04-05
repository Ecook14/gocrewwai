import React from 'react';
import { Handle, Position } from '@xyflow/react';
import { FileText, Target, Clock, MoreVertical } from 'lucide-react';

const TaskNode = ({ data }: { data: any }) => {
  return (
    <div className="w-64 glass border border-emerald-500/30 rounded-xl overflow-hidden shadow-2xl shadow-emerald-500/10 group hover:border-emerald-500/60 transition-all">
      <div className="bg-emerald-600/10 p-3 flex items-center justify-between border-b border-emerald-500/20">
        <div className="flex items-center gap-2">
           <div className="p-1.5 bg-emerald-600 rounded-md">
             <FileText className="w-3.5 h-3.5 text-white" />
           </div>
           <span className="text-xs font-bold text-white uppercase tracking-wider">Task Node</span>
        </div>
        <MoreVertical className="w-4 h-4 text-slate-500 cursor-pointer hover:text-white transition-colors" />
      </div>
      
      <div className="p-4 space-y-3">
        <div>
          <label className="text-[9px] font-bold text-emerald-500 uppercase tracking-widest mb-1 block">Description</label>
          <div className="text-xs font-medium text-slate-200 line-clamp-2">{data.description}</div>
        </div>
        
        <div className="pt-2 flex items-center gap-4">
          <div className="flex items-center gap-1.5">
            <Target className="w-3.5 h-3.5 text-slate-500" />
            <span className="text-[10px] text-slate-400 font-medium">Sequential</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-slate-500" />
            <span className="text-[10px] text-slate-400 font-medium">300s Limit</span>
          </div>
        </div>
      </div>

      <Handle
        type="target"
        position={Position.Top}
        className="w-3 h-3 bg-emerald-500 border-2 border-slate-900"
      />
      <Handle
        type="source"
        position={Position.Bottom}
        className="w-3 h-3 bg-emerald-500 border-2 border-slate-900"
      />
    </div>
  );
};

export default TaskNode;
