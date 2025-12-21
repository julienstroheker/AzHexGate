You are implementing a single GitHub issue for the AzHexGate project.

Before writing any code:
- Read and follow docs/architecture.md strictly.
- Treat the architecture document as the source of truth.
- Do NOT change the architecture unless explicitly instructed in this issue.

Scope rules:
- This issue must result in ONE small, focused pull request.
- Do NOT implement features outside the scope of this issue.
- Do NOT refactor unrelated code.
- Do NOT introduce new dependencies unless explicitly required.
- Do NOT break existing functionality or CI.

Quality rules:
- All code must compile.
- All tests must pass.
- Follow existing project structure and naming conventions.
- Prefer simple, explicit code over clever abstractions.
- Add tests when appropriate for the scope of this issue.

CI expectations:
- The PR must pass all GitHub Actions checks.
- If CI fails, fix the issue in the same PR.
- Do not leave TODOs that would cause CI or runtime failures.

Implementation guidance:
- Follow the architecture’s separation of concerns:
  - CLI logic stays in cmd/azhexgate and client/
  - Gateway logic stays in cmd/gateway and gateway/
  - Shared logic goes in internal/
- Do not mix responsibilities across layers.

Testing guidance:
- Unit tests are preferred over integration tests unless explicitly requested.
- Do not introduce Azure dependencies unless the issue explicitly requires them.
- Mock external services where possible.

Commit & PR rules:
- Use clear, descriptive commit messages.
- The PR description must explain:
  - What was implemented
  - Why it was implemented
  - How it aligns with the architecture
- Keep the PR easy to review.

If something is unclear:
- Make the smallest reasonable assumption.
- Document the assumption in the PR description.
- Do NOT expand the scope to “fix” unrelated issues.

Your goal:
Implement exactly what this issue asks for, no more and no less, while keeping the repository in a clean, working state.
