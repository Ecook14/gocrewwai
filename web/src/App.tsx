import React, { useCallback, useState } from 'react';
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
  Node,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import AgentNode from './components/AgentNode';
import TaskNode from './components/TaskNode';
import Header from './components/Header';
import Sidebar from './components/Sidebar';

import { useStream } from './hooks/useSSE';
import {
  kickoffCrew,
  getToken,
  setToken as persistToken,
  hasToken,
} from './api/client';
import {
  mapGraphToKickoff,
  describeKickoffPair,
  DEFAULT_MODEL,
  NoAgentTaskPairError,
} from './utils/orchestrator';

type AgentNodeData = {
  role: string;
  goal: string;
  model?: string;
  backstory?: string;
  status: string;
};

type TaskNodeData = {
  description: string;
  expectedOutput?: string;
  status: string;
};

// CrewNode pins the node data shape so useNodesState does not infer a narrow
// literal union from the initial array (which breaks every setNodes call).
type CrewNode = Node<AgentNodeData | TaskNodeData>;

const initialNodes: CrewNode[] = [
  {
    id: 'agent-1',
    type: 'agentNode',
    position: { x: 250, y: 100 },
    data: {
      role: 'Lead Researcher',
      goal: 'Gather market intelligence',
      model: DEFAULT_MODEL,
      status: 'idle',
    },
  },
  {
    id: 'task-1',
    type: 'taskNode',
    position: { x: 250, y: 300 },
    data: { description: 'Analyze AI chip market trends', status: 'pending' },
  },
];

const initialEdges: Edge[] = [
  { id: 'e1-2', source: 'agent-1', target: 'task-1', animated: true },
];

const nodeTypes = {
  agentNode: AgentNode,
  taskNode: TaskNode,
};

// agentStatus maps backend agent event types onto node status chips.
function agentStatus(eventType: string): string | null {
  if (eventType === 'agent.thinking' || eventType.startsWith('agent.reasoning')) {
    return 'thinking';
  }
  if (
    eventType === 'agent.execution.started' ||
    eventType === 'agent.feedback.received'
  ) {
    return 'running';
  }
  if (
    eventType === 'agent.execution.completed' ||
    eventType === 'agent.reasoning.completed'
  ) {
    return 'idle';
  }
  if (eventType === 'agent.execution.error') {
    return 'error';
  }
  return null;
}

// taskStatus maps backend task event types onto task node chips. With one
// task per run the event applies to every task node on the canvas.
function taskStatus(eventType: string): string | null {
  if (eventType === 'task.started') return 'running';
  if (eventType === 'task.completed' || eventType === 'task.evaluated') return 'done';
  if (eventType === 'task.failed') return 'failed';
  return null;
}

function Flow() {
  const [nodes, setNodes, onNodesChange] = useNodesState<CrewNode>(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(initialEdges);
  const [activeSession, setActiveSession] = useState<string | null>(null);
  const [token, setTokenState] = useState<string>(() => getToken() ?? '');
  const [kickoffError, setKickoffError] = useState<string | null>(null);
  const [kickoffBusy, setKickoffBusy] = useState(false);

  const { lastEvent, status: streamStatus } = useStream(activeSession);

  // Update nodes in real-time based on backend events.
  React.useEffect(() => {
    if (!lastEvent) return;
    const agentNext = agentStatus(lastEvent.type);
    const taskNext = taskStatus(lastEvent.type);
    if (!agentNext && !taskNext) return;

    setNodes((nds) =>
      nds.map((node) => {
        if (node.type === 'agentNode' && agentNext && lastEvent.source) {
          const data = node.data as AgentNodeData;
          if (data.role === lastEvent.source) {
            return { ...node, data: { ...data, status: agentNext } };
          }
        }
        if (node.type === 'taskNode' && taskNext) {
          const data = node.data as TaskNodeData;
          return { ...node, data: { ...data, status: taskNext } };
        }
        return node;
      }),
    );
  }, [lastEvent, setNodes]);

  const onConnect = useCallback(
    (params: Connection | Edge) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  const onTokenChange = useCallback((value: string) => {
    setTokenState(value);
    persistToken(value.trim() ? value.trim() : null);
  }, []);

  const onKickoff = async () => {
    setKickoffError(null);
    if (!hasToken()) {
      setKickoffError('Set the API token in the header first — the backend rejects unauthenticated kickoffs.');
      return;
    }
    let payload;
    try {
      payload = mapGraphToKickoff(nodes, edges);
    } catch (err) {
      setKickoffError(err instanceof NoAgentTaskPairError ? err.message : 'Invalid graph.');
      return;
    }
    setKickoffBusy(true);
    try {
      const res = await kickoffCrew(payload);
      setActiveSession(res.session_id);
    } catch (err) {
      setKickoffError(describeKickoffFailure(err));
    } finally {
      setKickoffBusy(false);
    }
  };

  const pairDescription = describeKickoffPair(nodes, edges);

  return (
    <div className="w-full h-screen bg-slate-950">
      <Header
        onKickoff={onKickoff}
        kickoffBusy={kickoffBusy}
        kickoffHint={pairDescription}
        token={token}
        onTokenChange={onTokenChange}
        streamStatus={streamStatus}
        sessionId={activeSession}
      />
      <div className="flex h-full pt-16">
        <Sidebar
          onAddNode={(type: 'agentNode' | 'taskNode') => {
            const id = `${type}-${nodes.length + 1}`;
            setNodes((nds) => [
              ...nds,
              {
                id,
                type,
                position: { x: Math.random() * 400, y: Math.random() * 400 },
                data:
                  type === 'agentNode'
                    ? { role: 'New Agent', goal: 'Define goal', model: DEFAULT_MODEL, status: 'idle' }
                    : { description: 'New task', status: 'pending' },
              },
            ]);
          }}
          sessionId={activeSession}
        />
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
          {kickoffError && (
            <div className="absolute top-4 left-1/2 -translate-x-1/2 max-w-xl bg-red-950/90 border border-red-500/40 text-red-200 text-sm px-4 py-2 rounded-lg shadow-xl">
              {kickoffError}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// describeKickoffFailure turns backend/transport failures into actionable UI
// text instead of leaking raw payloads.
function describeKickoffFailure(err: unknown): string {
  if (err && typeof err === 'object' && 'response' in err) {
    const resp = (err as { response?: { status?: number; data?: { error?: string; message?: string } } }).response;
    const status = resp?.status;
    const detail = resp?.data?.error ?? resp?.data?.message ?? '';
    if (status === 401) return 'Backend rejected the token (401). Check the API token in the header.';
    if (status === 409) return 'A run with this idempotency key is already in flight (409). Wait for it or retry.';
    if (status === 503) return 'The backend has no provider key configured (503): set OPENAI_API_KEY on the server.';
    if (status === 400) return `Backend rejected the request (400): ${detail || 'invalid payload'}`;
    if (typeof status === 'number') return `Kickoff failed (${status}): ${detail || 'see server logs'}`;
  }
  if (err instanceof Error) return `Kickoff failed: ${err.message}`;
  return 'Kickoff failed: unreachable backend. Is the server running?';
}

export default function App() {
  return (
    <ReactFlowProvider>
      <Flow />
    </ReactFlowProvider>
  );
}
