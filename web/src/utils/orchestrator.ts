import { Node, Edge } from '@xyflow/react';
import type { KickoffRequest } from '../api/client';

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

// NoAgentTaskPairError means the canvas has no task wired to an agent, so
// there is nothing the backend can execute. The backend runs exactly one
// agent and one task per kickoff; when several pairs exist the first
// task (in canvas order) with an incoming edge from an agent wins, and the
// UI surfaces which pair was sent.
export class NoAgentTaskPairError extends Error {
  constructor() {
    super('Connect a task node to an agent node before kicking off.');
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

// mapGraphToKickoff converts the canvas into the backend's flat kickoff
// schema. The backend accepts one agent and one task per request
// (agent_role + task_description are required), so this picks the first
// task with an incoming edge from an agent.
export function mapGraphToKickoff(nodes: Node[], edges: Edge[]): KickoffRequest {
  for (const taskNode of nodes) {
    const task = taskData(taskNode);
    if (!task) continue;
    const edge = edges.find((e) => e.target === taskNode.id);
    if (!edge) continue;
    const source = nodes.find((sn) => sn.id === edge.source);
    if (!source) continue;
    const agent = agentData(source);
    if (!agent) continue;
    return {
      session_id: `sess_${Date.now()}`,
      agent_role: agent.role,
      agent_goal: agent.goal,
      agent_backstory: agent.backstory,
      agent_model: agent.model,
      task_description: task.description,
      task_expected_output: task.expectedOutput || undefined,
      crew_process: 'sequential',
    };
  }
  throw new NoAgentTaskPairError();
}

// describeKickoffPair returns a human-readable "role → task" summary of what
// mapGraphToKickoff would send, for confirmation UI. Null when invalid.
export function describeKickoffPair(nodes: Node[], edges: Edge[]): string | null {
  try {
    const req = mapGraphToKickoff(nodes, edges);
    const short =
      req.task_description.length > 60
        ? req.task_description.slice(0, 60) + '…'
        : req.task_description;
    return `${req.agent_role} → ${short}`;
  } catch {
    return null;
  }
}
