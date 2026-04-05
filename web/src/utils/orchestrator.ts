import { Node, Edge } from '@xyflow/react';

export interface CrewPayload {
  session_id: string;
  agents: Array<{
    role: string;
    goal: string;
    backstory: string;
  }>;
  tasks: Array<{
    description: string;
    agent_role: string;
  }>;
}

export function mapGraphToCrewConfig(nodes: Node[], edges: Edge[]): CrewPayload {
  const agents: CrewPayload['agents'] = [];
  const tasks: CrewPayload['tasks'] = [];

  // 1. Extract Agents
  nodes.filter(n => n.type === 'agentNode').forEach(n => {
    agents.push({
      role: n.data.role as string,
      goal: n.data.goal as string,
      backstory: (n.data.backstory as string) || "Elite Crew-GO Agent",
    });
  });

  // 2. Extract Tasks and map to connected Agents
  nodes.filter(n => n.type === 'taskNode').forEach(n => {
    // Find the agent connected to this task (incoming edge to task)
    const edge = edges.find(e => e.target === n.id);
    const sourceAgent = nodes.find(sn => sn.id === edge?.source);

    tasks.push({
      description: n.data.description as string,
      agent_role: sourceAgent?.data.role as string || "Unassigned",
    });
  });

  return {
    session_id: `sess_${Date.now()}`,
    agents,
    tasks,
  };
}
