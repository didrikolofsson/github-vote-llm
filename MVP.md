# vote-llm – MVP Specification

## 1. Product Definition (MVP)

### Core Value Proposition

When a feature request receives enough traction in GitHub, a draft pull request is automatically generated for review.

This MVP is:

- GitHub-native
- Vote-triggered
- Human-governed
- AI-executed
- Safe-by-default

---

## 2. MVP Objective

Within 8 weeks, prove that:

- High-vote feature requests can reliably become reviewable PRs.
- Engineers feel it reduces implementation time.
- Teams keep it enabled after initial usage.

This is a validation experiment, not a full SaaS platform.

---

## 3. Target User (Alpha)

Small SaaS teams:

- 2–12 engineers
- GitHub-centric workflow
- Structured feature requests via Issues
- AI-curious
- Care about shipping speed

---

## 4. System Architecture (MVP Scope)

### 4.1 GitHub App

Responsibilities:

- Receive webhook events
- Read issues, labels, reactions
- Write labels, comments, and PRs
- Operate with scoped repo permissions

No user accounts.
No external dashboard.
GitHub is the UI.

---

### 4.2 Execution Engine (Backend Service)

Language: Go

Responsibilities:

- Webhook verification
- Vote counting
- State machine transitions
- Trigger AI execution
- Run sandbox
- Enforce cost and time caps
- Create PR
- Post metadata comments

Persistence:

- Lightweight storage (in-memory or SQLite) for:
  - Processed issue IDs
  - Execution state
  - Idempotency control

---

### 4.3 Sandbox Runtime

Mandatory for credibility and safety.

Requirements:

- Docker-based isolation
- No external network access (unless explicitly allowed)
- CPU and memory limits
- Hard timeout
- Ephemeral workspace
- Automatic cleanup after execution

Execution Flow:

1. Clone repository
2. Checkout new branch
3. Inject structured prompt
4. Run Claude
5. Run tests
6. If tests pass:
   - Create PR
7. Else:
   - Mark as failed
8. Cleanup workspace

---

## 5. Issue State Machine

### Labels

| State                  | Label              |
| ---------------------- | ------------------ |
| Feature request        | `feature-request`  |
| Vote threshold met     | `candidate`        |
| Approved for execution | `approved-for-dev` |
| Running                | `llm-in-progress`  |
| PR created             | `llm-pr-created`   |
| Failed                 | `llm-failed`       |

### Rules

- Only trigger if:
  - Label `feature-request` is present
  - Vote count >= threshold
- Only execute if:
  - `approved-for-dev` is present (configurable)
- Execution must be idempotent.
- An issue cannot be processed twice unless manually reset.

---

## 6. Configuration (Repo-Level)

Example: `vote-llm.yaml`

```yaml
vote_threshold: 3
require_manual_approval: true
run_tests: true
max_budget_usd: 5
timeout_minutes: 20
sandbox:
  memory_mb: 2048
  cpu_limit: 1
```

Defaults allowed if file not present.

---

## 7. Prompt Strategy (MVP)

Claude receives:

- Issue title and body
- Full directory tree
- Relevant files (via grep keyword extraction)
- Existing test structure
- Contribution guidelines (if present)

Constraints enforced in prompt:

- Modify minimal number of files
- Add or update tests
- No unnecessary refactors
- Do not modify unrelated files
- Follow existing patterns and style

Goal: deterministic, minimal, safe diffs.

---

## 8. PR Output Requirements

Each PR must include:

- Summary of implementation
- Files changed
- Lines added/removed
- Execution time
- Budget used
- Test result status

This builds trust and transparency.

---

## 9. 8-Week Development Roadmap

### Phase 1 (Week 1–2): GitHub App + Webhook Hardening

Deliverables:

- Proper GitHub App
- Verified webhook handling
- Permission scoping
- Basic state machine
- Vote counting
- Reliable label transitions

**Milestone:** Issue → candidate transition works reliably.

### Phase 2 (Week 3–4): Sandbox Execution Engine

Deliverables:

- Docker-based execution
- Budget enforcement
- Timeout enforcement
- Workspace cleanup
- Fail-safe error handling
- Test runner integration

**Milestone:** Approved issue → safe execution → PR creation.

### Phase 3 (Week 5–6): PR Quality Iteration

Deliverables:

- Improved context injection
- Better file selection heuristics
- Structured prompting refinements
- Diff minimization logic
- Test compatibility improvements

**Milestone:** Majority of simple feature PRs require only minor edits.

### Phase 4 (Week 7): Integration Test Harness

Create automated test repos:

- Simple Node project
- Simple Go project
- Repo with DB migration
- Repo with enforced test suite

Simulate full workflow in CI:

- Issue created
- Votes added
- Threshold reached
- Execution triggered
- PR created

**Milestone:** Deterministic E2E automation coverage.

### Phase 5 (Week 8): Alpha Testing

- Install on 2–3 real SaaS repos
- Execute at least 10 features
- Collect structured feedback
- Measure:
  - % accepted PRs
  - Review time
  - Rewrite effort
  - Developer sentiment

---

## 10. MVP Success Criteria

**Success if:**

- ≥ 60% PRs accepted with minor edits
- Developers report measurable time savings
- Teams execute multiple features
- No major security incidents

**Failure if:**

- PRs require heavy rewrite
- Teams disable tool quickly
- Tests frequently break
- Sandbox instability observed

---

## 11. Explicitly Out of Scope (MVP)

Do NOT build:

- Dashboard UI
- Billing system
- Multi-source aggregation
- AI ranking layer
- Advanced analytics
- Enterprise authentication
- Marketing site

Focus exclusively on execution quality and safety.

---

## 12. Strategic Positioning (MVP Phase)

This is not:

- A product management platform
- A governance tool
- A full SaaS system

This is:

- A focused experiment in demand-triggered execution acceleration inside GitHub.

The only question the MVP must answer:

> Can approved backlog items reliably become high-quality draft pull requests?
