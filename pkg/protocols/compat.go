// Package protocols implements interoperability protocols for Gocrew agents.
//
// This package provides adapters that make gocrewwai compatible with
// external agent frameworks:
//
//   - ADK (Google Agent Development Kit): google.golang.org/adk/v2
//   - A2A (Agent-to-Agent): a2aproject/a2a-go
//   - MCP (Model Context Protocol): modelcontextprotocol/go-sdk
//
// Architecture:
//
//	Gocrewwai Core <-> Protocol Adapters <-> External Frameworks
//
// The adapters translate between gocrewwai's native types and the
// external framework's types, enabling seamless interoperation.
package protocols
