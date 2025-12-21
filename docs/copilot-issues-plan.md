# Copilot Issues Plan — AzHexGate

This document defines the ordered list of GitHub issues required to implement AzHexGate incrementally.
Each issue is intentionally small, scoped to a single concern, and designed to keep the repository in a working state at all times.
Issues should be created and assigned to Copilot **one at a time**, following this order.

---

## Issue 1 — Repository scaffolding and CI pipeline

**Labels:** `type:scaffold`, `area:ci`, `copilot:ready`  
**Milestone:** M0 – Scaffolding & CI

Create the initial repository structure, Go module, and empty entrypoints for the CLI and Cloud Gateway.
Add a GitHub Actions workflow that runs `go mod tidy`, `go build ./...`, and `go test ./...`.
This pipeline must fail on compilation or test errors and serve as the primary feedback loop for Copilot.
No Azure dependencies or infrastructure code should be introduced.

---

## Issue 2 — CLI entrypoint skeleton

**Labels:** `type:feature`, `area:cli`, `copilot:ready`  
**Milestone:** M1 – Local Client MVP

Implement a minimal `azhexgate` CLI with a `start` command that parses flags and exits cleanly.
No network calls or tunnel logic should be added yet.
This issue establishes the CLI contract and ensures the binary is runnable and testable.

---

## Issue 3 — Cloud Gateway entrypoint skeleton

**Labels:** `type:feature`, `area:gateway`, `copilot:ready`  
**Milestone:** M2 – Cloud Gateway MVP

Add a minimal HTTP server for the Cloud Gateway with a health endpoint only.
No routing, Relay integration, or management logic should be included.
The goal is to ensure the gateway binary builds, runs, and integrates cleanly with CI.

---

## Issue 4 — Shared configuration and logging package

**Labels:** `type:feature`, `area:internal`, `copilot:ready`  
**Milestone:** M0 – Scaffolding & CI

Introduce shared configuration loading and structured logging utilities under `internal/`.
These utilities should be usable by both the CLI and the Cloud Gateway.
No behavior changes are expected; this issue is purely foundational plumbing.

---

## Issue 5 — Management API skeleton

**Labels:** `type:feature`, `area:management-api`, `copilot:ready`  
**Milestone:** M3 – Management API MVP

Add the Management API HTTP handlers, including a `/api/tunnels` endpoint that returns a static mock response.
No Azure Relay, Key Vault, or authentication logic should be implemented yet.
This locks the API shape early and allows downstream components to integrate safely.

---

## Issue 6 — CLI to Management API integration (mocked)

**Labels:** `type:feature`, `area:cli`, `copilot:ready`  
**Milestone:** M3 – Management API MVP

Wire the CLI `start` command to call the Management API and print the returned public URL.
Use a mocked HTTP server in tests to avoid external dependencies.
This validates the CLI ↔ Management API contract without introducing Azure complexity.

---

## Issue 7 — Azure Relay abstraction layer

**Labels:** `type:feature`, `area:relay`, `copilot:ready`  
**Milestone:** M4 – Azure Relay Integration

Introduce interfaces and adapters for Azure Relay interactions without making real Azure calls.
This abstraction layer will isolate Azure SDK usage and enable mocking in tests.
No real Relay connections should be created in this issue.

---

## Issue 8 — Local Client listener and forwarding loop (in-memory)

**Labels:** `type:feature`, `area:cli`, `copilot:ready`  
**Milestone:** M1 – Local Client MVP

Implement the Local Client listener loop using in-memory streams instead of Azure Relay.
Forward incoming requests to a local HTTP server and return responses.
This validates request/response forwarding logic independently of Azure.

---

## Issue 9 — Cloud Gateway sender logic (in-memory)

**Labels:** `type:feature`, `area:gateway`, `copilot:ready`  
**Milestone:** M2 – Cloud Gateway MVP

Implement the Cloud Gateway request forwarding logic using in-memory streams.
Mirror the Local Client behavior from Issue 8 to validate symmetry.
No Azure Relay or authentication logic should be introduced yet.

---

## Issue 10 — Azure Relay real integration

**Labels:** `type:feature`, `area:relay`, `copilot:ready`  
**Milestone:** M4 – Azure Relay Integration

Replace the in-memory Relay implementation with real Azure Relay Hybrid Connections.
Use Managed Identity for Gateway authentication and short-lived SAS tokens for the Local Client.
Ensure all existing tests continue to pass and add minimal integration coverage where appropriate.

---

## Issue 11 — Infrastructure Bicep MVP

**Labels:** `type:infra`, `area:infra`, `copilot:ready`  
**Milestone:** M6 – Infrastructure & E2E

Add minimal Bicep templates to deploy the Azure Relay namespace and App Service.
No DNS, certificates, or Front Door integration should be included yet.
Infrastructure must be validated in CI using Bicep build or what-if checks.

---

## Issue 12 — End-to-end tunnel integration test

**Labels:** `type:test`, `area:ci`, `copilot:ready`  
**Milestone:** M6 – Infrastructure & E2E

Add an end-to-end integration test that validates real tunneling through Azure Relay.
The test should spin up a local HTTP server, start the CLI, deploy the Gateway, and verify traffic flows end-to-end.
This test should run in CI and serve as the final validation of the system.

---

## Notes

- Only one issue should be assigned to Copilot at a time.
- Each issue must result in a single, small pull request.
- The repository must remain green after every merge.
- Architecture changes require explicit human approval.
