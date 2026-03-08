package crew

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Flow Visualization — DOT Graph Export
// ---------------------------------------------------------------------------
//
// Generates Graphviz DOT format for visualizing crew execution flows,
// agent relationships, task dependencies, and delegation chains.
//
// Usage:
//
//	dot := crew.GenerateDOT(myCrew)
//	os.WriteFile("flow.dot", []byte(dot), 0644)
//	// Render: dot -Tpng flow.dot -o flow.png

// GenerateDOT creates a Graphviz DOT representation of a crew's execution flow.
func GenerateDOT(c *Crew) string {
	var sb strings.Builder

	sb.WriteString("digraph CrewFlow {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box, style=\"rounded,filled\", fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [fontname=\"Helvetica\", fontsize=10];\n")
	sb.WriteString("\n")

	// Crew node
	crewLabel := "Crew"
	if c.Process != "" {
		crewLabel = fmt.Sprintf("Crew\\n(%s)", c.Process)
	}
	sb.WriteString(fmt.Sprintf("  crew [label=\"%s\", shape=ellipse, fillcolor=\"#4A90D9\", fontcolor=white];\n", crewLabel))
	sb.WriteString("\n")

	// Agent nodes
	sb.WriteString("  // Agents\n")
	sb.WriteString("  subgraph cluster_agents {\n")
	sb.WriteString("    label=\"Agents\";\n")
	sb.WriteString("    style=dashed;\n")
	sb.WriteString("    color=\"#888888\";\n")
	for i, agent := range c.Agents {
		agentID := fmt.Sprintf("agent_%d", i)
		toolCount := len(agent.Tools)
		label := fmt.Sprintf("%s\\n(%d tools)", agent.Role, toolCount)
		sb.WriteString(fmt.Sprintf("    %s [label=\"%s\", fillcolor=\"#50C878\", fontcolor=white];\n", agentID, label))
	}
	sb.WriteString("  }\n\n")

	// Task nodes and edges
	sb.WriteString("  // Tasks\n")
	sb.WriteString("  subgraph cluster_tasks {\n")
	sb.WriteString("    label=\"Tasks\";\n")
	sb.WriteString("    style=dashed;\n")
	sb.WriteString("    color=\"#888888\";\n")
	for i, task := range c.Tasks {
		taskID := fmt.Sprintf("task_%d", i)
		desc := task.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fillColor := "#FFB347"
		if task.AsyncExecution {
			fillColor = "#FF6B6B"
			desc += "\\n(async)"
		}
		sb.WriteString(fmt.Sprintf("    %s [label=\"Task %d\\n%s\", fillcolor=\"%s\"];\n", taskID, i+1, desc, fillColor))
	}
	sb.WriteString("  }\n\n")

	// Crew → Task edges (sequential flow)
	sb.WriteString("  // Flow edges\n")
	if len(c.Tasks) > 0 {
		sb.WriteString("  crew -> task_0 [label=\"start\"];\n")
	}
	for i := 0; i < len(c.Tasks)-1; i++ {
		sb.WriteString(fmt.Sprintf("  task_%d -> task_%d;\n", i, i+1))
	}

	// Task → Agent assignments
	sb.WriteString("\n  // Task-Agent assignments\n")
	for i, task := range c.Tasks {
		if task.Agent != nil {
			for j, agent := range c.Agents {
				if agent.Role == task.Agent.Role {
					sb.WriteString(fmt.Sprintf("  task_%d -> agent_%d [style=dashed, label=\"assigned\", color=\"#888888\"];\n", i, j))
					break
				}
			}
		}
	}

	// Task context dependencies
	sb.WriteString("\n  // Context dependencies\n")
	for i, task := range c.Tasks {
		if len(task.Context) > 0 {
			for _, dep := range task.Context {
				for j, t := range c.Tasks {
					if t == dep && j != i {
						sb.WriteString(fmt.Sprintf("  task_%d -> task_%d [style=dotted, label=\"context\", color=\"#4A90D9\"];\n", j, i))
					}
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// GenerateTaskDependencyDOT creates a focused dependency graph for tasks only.
func GenerateTaskDependencyDOT(c *Crew) string {
	var sb strings.Builder

	sb.WriteString("digraph TaskDeps {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=record, style=filled, fontname=\"Helvetica\"];\n\n")

	for i, task := range c.Tasks {
		desc := task.Description
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}
		agentRole := "unassigned"
		if task.Agent != nil {
			agentRole = task.Agent.Role
		}
		label := fmt.Sprintf("{Task %d|%s|Agent: %s}", i+1, desc, agentRole)
		sb.WriteString(fmt.Sprintf("  task_%d [label=\"%s\", fillcolor=\"#FFB347\"];\n", i, label))
	}

	sb.WriteString("\n  // Sequential flow\n")
	for i := 0; i < len(c.Tasks)-1; i++ {
		sb.WriteString(fmt.Sprintf("  task_%d -> task_%d;\n", i, i+1))
	}

	sb.WriteString("\n  // Context deps\n")
	for i, task := range c.Tasks {
		for _, dep := range task.Context {
			for j, t := range c.Tasks {
				if t == dep {
					sb.WriteString(fmt.Sprintf("  task_%d -> task_%d [style=dashed, color=blue];\n", j, i))
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// GenerateMermaidFlow creates a Mermaid flowchart for web-based rendering.
func GenerateMermaidFlow(c *Crew) string {
	var sb strings.Builder

	sb.WriteString("graph TD\n")
	sb.WriteString("  Start([Start]) --> ")

	if len(c.Tasks) > 0 {
		sb.WriteString("T0\n")
	} else {
		sb.WriteString("End([End])\n")
		return sb.String()
	}

	for i, task := range c.Tasks {
		desc := task.Description
		if len(desc) > 35 {
			desc = desc[:32] + "..."
		}

		taskStyle := "T%d[\"%s\"]"
		if task.AsyncExecution {
			taskStyle = "T%d{{\"%s (async)\"}}"
		}
		sb.WriteString(fmt.Sprintf("  "+taskStyle+"\n", i, desc))

		if task.Agent != nil {
			sb.WriteString(fmt.Sprintf("  T%d -.-> A%d(\"%s\")\n", i, i, task.Agent.Role))
		}

		if i < len(c.Tasks)-1 {
			sb.WriteString(fmt.Sprintf("  T%d --> T%d\n", i, i+1))
		} else {
			sb.WriteString(fmt.Sprintf("  T%d --> End([End])\n", i))
		}
	}

	// Context dependencies
	for i, task := range c.Tasks {
		for _, dep := range task.Context {
			for j, t := range c.Tasks {
				if t == dep {
					sb.WriteString(fmt.Sprintf("  T%d -.context.-> T%d\n", j, i))
				}
			}
		}
	}

	return sb.String()
}
