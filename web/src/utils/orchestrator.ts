import { Node, Edge } from '@xyflow/react';
import type { KickoffRequest, KickoffAgent, KickoffTask } from '../api/client';

export const DEFAULT_MODEL = 'gpt-4o';

export interface AgentSpec {
  role: string;
  goal: string;
  model: string;
  backstory: string;
}

export interface TaskSpec {
  description: string;
  expectedOutput: string;
}

// NoAgentTaskPairError means the canvas has no executable content, so there
// is nothing the backend can run: at least one agent and one task are
// required. The UI surfaces this instead of sending a doomed request.
export class NoAgentTaskPairError extends Error {
  constructor() {
    super('Add at least one agent node and one task node before kicking off.');
    this.name = 'NoAgentTaskPairError';
  }
}

function agentData(n: Node): AgentSpec | null {
  if (n.type !== 'agentNode' || typeof n.data !== 'object' || n.data === null) return null;
  const d = n.data as Record<string, unknown>;
  const role = typeof d.role === 'string' ? d.role.trim() : '';
  if (!role) return null;
  return {
    role,
    goal: typeof d.goal === 'string' ? d.goal : '',
    model: typeof d.model === 'string' && d.model ? d.model : DEFAULT_MODEL,
    backstory: typeof d.backstory === 'string' ? d.backstory : '',
  };
}

function taskData(n: Node): TaskSpec | null {
  if (n.type !== 'taskNode' || typeof n.data !== 'object' || n.data === null) return null;
  const d = n.data as Record<string, unknown>;
  const description = typeof d.description === 'string' ? d.description.trim() : '';
  if (!description) return null;
  return {
    description,
    expectedOutput: typeof d.expectedOutput === 'string' ? d.expectedOutput : '',
  };
}

// mapGraphToKickoff converts the whole canvas into the backend's multi-agent
// kickoff schema: every valid agent node becomes an agents[] entry and every
// valid task node a tasks[] entry, wired by incoming edges. Tasks without an
// incoming edge leave agent_role empty, which the backend resolves only when
// exactly one agent exists (otherwise it rejects with a clear 400 the UI
// surfaces). Throws NoAgentTaskPairError when there is nothing executable.
export function mapGraphToKickoff(nodes: Node[], edges: Edge[]): KickoffRequest {
  const agents: KickoffAgent[] = [];
  for (const n of nodes) {
    const agent = agentData(n);
    if (agent) agents.push(agent);
  }
  const tasks: KickoffTask[] = [];
  for (const n of nodes) {
    const task = taskData(n);
    if (!task) continue;
    const edge = edges.find((e) => e.target === n.id);
    const source = edge ? nodes.find((sn) => sn.id === edge.source) : undefined;
    const sourceAgent = source ? agentData(source) : null;
    tasks.push({
      description: task.description,
      expected_output: task.expectedOutput || undefined,
      agent_role: sourceAgent ? sourceAgent.role : undefined,
    });
  }
  if (agents.length === 0 || tasks.length === 0) {
    throw new NoAgentTaskPairError();
  }
  return {
    session_id: `sess_${Date.now()}`,
    agents,
    tasks,
    crew_process: 'sequential',
  };
}

// describeKickoffPair returns a human-readable summary of what
// mapGraphToKickoff would send, for confirmation UI. Null when invalid.
export function describeKickoffPair(nodes: Node[], edges: Edge[]): string | null {
  try {
    const req = mapGraphToKickoff(nodes, edges);
    const nAgents = req.agents?.length ?? 0;
    const nTasks = req.tasks?.length ?? 0;
    return `${nAgents} agent${nAgents === 1 ? '' : 's'}, ${nTasks} task${nTasks === 1 ? '' : 's'}`;
  } catch {
    return null;
  }
}
