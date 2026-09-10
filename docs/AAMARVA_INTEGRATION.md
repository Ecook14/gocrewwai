# AAMARVA Integration Plan

## Overview

AAMARVA is an autonomous agent-to-agent communication network. As of analysis date, the network has:
- **61 registered agents**, **159 posts**, **739 replies**, **161 connections**
- Public Floor for discovery + private connections for collaboration
- API key + token-based authentication (24h access tokens, 7d refresh tokens)
- Rate limits: 60 writes/min, 300 reads/min per agent

## Phase 1: Client Core ✅ COMPLETE

**Module:** `pkg/aamarva/`

### Implemented
- `Client` struct with full HTTP lifecycle
- `Register()` — Register a new agent identity
- `Login()` — Authenticate and get access token
- `SearchAgents()` — Query agent directory
- `SearchPosts()` — Query Floor posts
- `CreatePost()` — Publish to Floor (Emit/Intake)
- `ReplyToPost()` — Reply to posts
- `EstablishConnection()` — Create private channel
- `GetStats()` — Network statistics
- `GetADK()` — Retrieve platform specification
- `GetADK()` — Retrieve platform specification
- Full test coverage (6 tests, all passing)

### Files
- `pkg/aamarva/client.go` — Core client implementation
- `pkg/aamarva/client_test.go` — Integration tests
- `docs/aamarva_adk_spec.json` — Full AAMARVA ADK specification (56KB)

## Phase 2: Scout Agent ⚠️ CAVEATS

### Dependencies
**Internal:**
- `pkg/aamarva/client.go` (Phase 1, complete)
- `pkg/agents/agent.go` (Agent framework, complete)
- `pkg/flow/flow.go` (Flow orchestration)
- Background process for continuous polling
- Persistent storage for API keys (env vars or vault)

**External:**
- AAMARVA API key + Agent ID (must be obtained via Phase 1 register)
- Stable internet connection to aamarva.com
- API key must be stored securely (never in code)

### Caveats
1. **Rate Limits:** 60 writes/minute per agent. A Scout polling every 5s = 12 writes/min. Safe but leaves no room for other actions.
2. **Token Expiry:** Access tokens expire every 24h. Need refresh logic (`POST /api/auth/refresh`).
3. **API Key Exposure:** API keys are shown **exactly once** during registration. Must be persisted securely in env vars or vault.
4. **No Webhook Infrastructure:** AAMARVA has `GET /api/webhooks/events` but we need a running HTTP server to receive them.
5. **Small Network:** 61 agents, mostly hobbyist/sci-fi roleplay. Few enterprise agents. Discovery quality may be low.
6. **All agents "not verified":** No trust mechanism works until verification exists.

## Phase 3: Community Plugin ⚠️ CAVEATS

### Additional Dependencies
- Complete Phase 2
- Plugin/skill marketplace infrastructure
- gocrewwai plugin architecture (does not exist yet)
- Documentation and distribution channel

### Caveats
1. **No Plugin Architecture:** gocrewwai has no plugin/skill marketplace yet. Would need to build the entire plugin system first.
2. **Revenue Model:** No clear monetization. AAMARVA API is free. Who pays?
3. **Network Maturity:** 61 agents is too small for a marketplace. Need 1000+ for viable plugin ecosystem.
4. **Maintenance Overhead:** Plugin maintenance requires ongoing AAMARVA API compatibility.
5. **Competition:** OpenClaw Agent (AMR-TEUT-P3RE) already on AAMARVA using similar framework.
6. **Reputation Risk:** AAMARVA community is heavily sci-fi/roleplay. Enterprise positioning may not fit.

## Profitability Analysis

### Short-term (Phase 1-2): NOT PROFITABLE
- AAMARVA is free to use
- No revenue model visible
- Small network (61 agents) = low task volume
- Development cost exceeds any potential return

### Long-term (Phase 3+): CONDITIONALLY PROFITABLE
- IF AAMARVA grows to 1000+ agents
- IF gocrewwai becomes the preferred framework
- IF premium features are introduced (advanced discovery, automated delegation)
- IF plugin marketplace charges a commission

### Cost-Benefit Summary
| Factor | Assessment |
|--------|-----------|
| Development Cost (Phase 1) | Low (~1 day) |
| Development Cost (Phase 2) | Medium (~1-2 weeks) |
| Development Cost (Phase 3) | High (~1-2 months) |
| Revenue Potential | Near-zero currently |
| Strategic Value | High (first-mover advantage) |
| Maintenance Cost | Low (Phase 1-2), Medium (Phase 3) |

## Recommendation

Phase 1 is done. Phase 2 is technically feasible but **not yet economically justified**. Keep the client module as-is and revisit when:
1. AAMARVA network grows to 500+ agents
2. A clear monetization path emerges
3. Enterprise-grade agents appear on the network
4. AAMARVA introduces paid features or premium tiers
