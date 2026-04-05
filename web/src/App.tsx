import React, { useCallback, useMemo } from 'react';
import {
  ReactFlow,
  MiniMap,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  Edge,
  ReactFlowProvider,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import AgentNode from './components/AgentNode';
import TaskNode from './components/TaskNode';
import Header from './components/Header';
import Sidebar from './components/Sidebar';

import { useSSE } from './hooks/useSSE';
import { kickoffCrew } from './api/client';
import { mapGraphToCrewConfig } from './utils/orchestrator';

const initialNodes = [
  {
    id: 'agent-1',
    type: 'agentNode',
    position: { x: 250, y: 100 },
    data: { role: 'Lead Researcher', goal: 'Gather market intelligence', status: 'idle' },
  },
  {
    id: 'task-1',
    type: 'taskNode',
    position: { x: 250, y: 300 },
    data: { description: 'Analyze AI chip market trends 2024', status: 'pending' },
  },
];

const initialEdges = [{ id: 'e1-2', source: 'agent-1', target: 'task-1', animated: true }];

const nodeTypes = {
  agentNode: AgentNode,
  taskNode: TaskNode,
};

function Flow() {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [activeSession, setActiveSession] = React.useState<string | null>(null);

  const { lastEvent } = useSSE(activeSession);

  // Update nodes in real-time based on backend events
  React.useEffect(() => {
    if (!lastEvent) return;

    setNodes((nds: Node[]) => nds.map((node: Node) => {
      if (node.type === 'agentNode' && node.data.role === lastEvent.agent_role) {
        return {
          ...node,
          data: { ...node.data, status: lastEvent.type.includes('thinking') ? 'thinking' : 'idle' }
        };
      }
      return node;
    }));
  }, [lastEvent, setNodes]);

  const onConnect = useCallback(
    (params: Connection | Edge) => setEdges((eds: Edge[]) => addEdge(params, eds)),
    [setEdges],
  );

  const onKickoff = async () => {
    const payload = mapGraphToCrewConfig(nodes, edges);
    try {
      console.log('🚀 Starting Crew:', payload);
      await kickoffCrew(payload);
      setActiveSession(payload.session_id);
    } catch (err) {
      console.error('❌ Kickoff failed:', err);
    }
  };

  return (
    <div className="w-full h-screen bg-slate-950">
      <Header onKickoff={onKickoff} />
      <div className="flex h-full pt-16">
        <Sidebar onAddNode={(type: 'agentNode' | 'taskNode') => {
          const id = `${type}-${nodes.length + 1}`;
          setNodes((nds: Node[]) => [...nds, {
            id,
            type,
            position: { x: Math.random() * 400, y: Math.random() * 400 },
            data: type === 'agentNode' 
              ? { role: 'New Agent', goal: 'Define Goal', status: 'idle' } 
              : { description: 'New Task', status: 'pending' }
          }]);
        }} />
        <div className="flex-grow relative h-full">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect}
            nodeTypes={nodeTypes}
            fitView
            colorMode="dark"
          >
            <Controls />
            <MiniMap className="bg-slate-900 border-slate-800" nodeColor="#3b82f6" />
            <Background color="#334155" gap={20} />
          </ReactFlow>
        </div>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <ReactFlowProvider>
      <Flow />
    </ReactFlowProvider>
  );
}
