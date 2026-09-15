# Security Policy

## Supported Versions

| Version | Status | Notes |
|---------|--------|-------|
| 1.0.x   | Current | Active development, security fixes |
| 0.9.x   | Maintained | Security fixes until EOL date |
| <0.9    | EOL    | No additional security patches |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in gocrewwai, please report it responsibly:

**Do NOT:**
- Open a public GitHub issue
- Discuss the vulnerability in public forums or chats
- Exploit the vulnerability beyond proof-of-concept

**DO:**
- Email: security@ecook14.dev
- Include: description, impact, reproduction steps, affected version
- Allow up to 48 hours for initial response
- Expect a fix timeline based on severity:
  - Critical: 24-48 hours
  - High: 1 week
  - Medium: 2 weeks
  - Low: next release cycle

## Security Considerations for Users

- Always validate and sanitize inputs to tools and agents
- Use the built-in guardrails for production deployments
- Enable human-in-the-loop for actions that modify external systems
- Rotate API keys regularly and use environment variables, never hardcode
- Use the WASM sandbox for untrusted code execution
- Review tool permissions: each tool has specific access scope
- Monitor telemetry for unusual patterns (unexpected tool calls, high token usage)

## Disclosure Policy

When a vulnerability is fixed, we will:
1. Publish a security advisory on GitHub
2. Update the CHANGELOG with the fix
3. Release a patched version within the timeline above
4. Credit the reporter (unless they request anonymity)
