# ADR-011: AI Workflow Automation — Skills, Hooks, and Agents

**Status:** Draft (under review)

## Context

Development sessions with Claude Code are inconsistent. The project has thorough
documentation (CLAUDE.md, AGENTS.md, diagrams, ADRs) that describes *what the
system is*, but nothing that encodes *how we want to work on it*. The process —
plan before implementing, review at the PR stage, capture learnings — lives in
the developer's head and depends on memory to activate.

This leads to:
- Skipping planning on "simple" changes that turn out to be complex.
- No adversarial review unless explicitly remembered and requested.
- Failure mode analysis never happens unless prompted.
- Inconsistent quality gates across sessions.
- Learnings from one session don't transfer to the next.
- Starting a new repository means rebuilding workflow habits from scratch.

### The Cost Model of Human Attention

Implementation is getting cheaper. With AI agents, the cost of writing code
approaches zero — but the cost of *wrong code landing in main* stays high.
The developer's judgment is the scarce resource. It should be spent where it
has the highest leverage:

1. **Design** (high leverage) — Choosing the right approach, questioning
   tradeoffs, approving the plan. Wrong design wastes entire implementation
   cycles.
2. **Implementation** (low leverage) — Following the approved plan, writing
   code, running tests, fixing lint. The agent can do this autonomously.
3. **PR review** (high leverage) — Verifying the outcome against the intent,
   catching failure modes, ensuring architectural consistency. This is where
   bugs are cheapest to catch before they reach main.

The developer should be a **gatekeeper at the boundaries**, not a supervisor
during execution. The workflow should reflect this:

```
session start ──► /plan ──► approval ──► autonomous impl ──► /pr ──► merge
     │                                   (no human gates)      │
     │                        hooks: lint, vet, tests          │
  (hook asks                  rules: arch constraints          │
   "what are                  impl rule: branch, commit,       │
   you building?")            test, stop if plan is wrong      │
     │                                                         │
     └──────── /retro ◄── (hook: "run /retro?") ◄─────────────┘
```

### Claude Code Automation Primitives

1. **Skills** (`.claude/skills/`) — reusable prompt-driven workflows, invoked
   via `/skill-name` or triggered automatically by context.
2. **Hooks** (`settings.json`) — deterministic shell commands that run at
   lifecycle events (PreToolUse, PostToolUse, etc.). Can block actions.
3. **Agents** (`.claude/agents/`) — persistent subagents with custom system
   prompts, tool restrictions, and cross-session memory.
4. **Rules** (`.claude/rules/`) — path-scoped instructions that only load when
   Claude works on matching files.

## Decision

We will implement a three-layer automation system split between global (portable
across repos) and project-specific (per-repo) configuration.

The core principle: **human judgment at the boundaries, autonomous execution in
the middle. Automation ensures the boundaries are never skipped.**

### Layer 1: Global Skills (`~/.claude/skills/`) — Portable Workflow

These encode the development methodology and travel with the developer.

#### Design Boundary — `/plan`

| Skill | Invocation | Purpose |
|-------|------------|---------|
| `/plan` | Prompted by `SessionStart` hook, or manually | Hard gate: explore → ask → propose alternatives with tradeoffs → wait for explicit approval |

The developer is fully involved here. No code is written until the plan is
approved. This is the highest-leverage moment — catching a wrong approach here
saves an entire implementation cycle.

The `/plan` output must include a **deviation clause**: "If during implementation
you discover the plan is wrong — an assumption doesn't hold, a dependency
doesn't work as expected, or the approach hits an unforeseen blocker — stop
implementation, present what changed and why, and wait for the developer to
approve a revised plan before continuing."

This prevents the two failure modes of autonomous implementation:
- **Silent deviation**: the agent adapts the plan without telling you, and the
  PR contains surprises.
- **Indefinite blocking**: the agent gets stuck and waits forever instead of
  surfacing the problem.

#### PR Boundary — `/pr`

| Skill | Invocation | Purpose |
|-------|------------|---------|
| `/pr` | Developer invokes when implementation is complete | Self-contained: runs adversarial review, runs chaos analysis (if applicable), generates PR with findings |

`/pr` is a single invocation that orchestrates the full PR boundary:

1. **Diff analysis** — `git diff main...HEAD` to understand full scope of
   changes.
2. **Adversarial review** — For each changed file: correctness (edge cases,
   off-by-one, null handling), error handling (swallowed errors, missing
   cleanup), security (injection, auth bypass, leaked secrets), design
   (CLAUDE.md violations, unnecessary complexity).
3. **Chaos analysis** (conditional) — If changes touch events, projections,
   sync, workers, or CRDT logic: failure mode analysis covering crash failures,
   network failures, concurrent access, ordering issues, and idempotency.
4. **Findings report** — Present 🔴 CRITICAL / 🟡 WARNING / 💡 SUGGESTION to
   the developer. Fix any CRITICAL issues before proceeding.
5. **PR creation** — Generate PR with summary, test plan, and any unresolved
   warnings.

The developer reviews the PR output and findings, not the intermediate
implementation steps. This is the second high-leverage gate.

#### Session Boundary — `/retro`

| Skill | Invocation | Purpose |
|-------|------------|---------|
| `/retro` | Prompted by `Stop` hook, or manually | Capture learnings, update memories, identify new rules/skills |

Closes the feedback loop. Patterns discovered in retros become new rules,
memories, or skills — the system improves itself.

### Layer 2: Project Rules (`.claude/rules/`) — Scoped Constraints

#### Architectural Rules

Path-scoped rules that extract critical constraints from CLAUDE.md into files
that only load when relevant, reducing context waste:

| Rule file | Scope | Contents |
|-----------|-------|----------|
| `go-backend.md` | `api/**` | Event sourcing invariants, table-driven tests, HLC usage, user scoping |
| `react-frontend.md` | `web/**` | Dexie.js reads, offline-first writes, Safari-only APIs, accessibility |

These duplicate a subset of CLAUDE.md intentionally — rules load based on file
path, CLAUDE.md loads always. The overlap ensures constraints are visible when
working in a specific area without relying on the full CLAUDE.md being in context
after compaction.

#### Implementation Behavior Rule

A global rule (no path scope) that defines how the agent should work during
the autonomous implementation phase:

| Rule file | Scope | Contents |
|-----------|-------|----------|
| `implementation.md` | all files | Branch strategy, commit discipline, test expectations, deviation protocol |

This rule covers what the ADR's first draft left undefined — how the agent
behaves when unsupervised:

- **Branching**: Create a feature branch from main before starting implementation.
- **Commits**: Commit in logical chunks that match plan steps, not one giant
  commit at the end. Each commit message references what plan step it implements.
- **Testing**: Run relevant tests after each significant change. Don't wait
  until the end to discover the first change broke everything.
- **Scope discipline**: Don't refactor code outside the plan scope. Don't add
  features not in the plan. Don't "improve" adjacent code.
- **Deviation protocol**: If the plan hits reality and an assumption is wrong,
  stop, present the deviation and proposed adjustment, and wait for approval
  before continuing. Do not silently adapt.
- **Stuck protocol**: If blocked after 2-3 genuine attempts at a step, surface
  the blocker to the developer rather than going in circles.

### Layer 3: Enforcement Hooks — Automated Quality Gates

Hooks enforce the workflow boundaries automatically. The developer doesn't need
to remember — the system reminds or enforces.

#### Session Lifecycle Hooks

| Hook | Event | Action |
|------|-------|--------|
| Session opener | `SessionStart` | Surface a prompt: "What are you building? Run /plan to get started." |
| Session closer | `Stop` | Surface a reminder: "Session ending — consider running /retro to capture learnings." |

These are lightweight nudges, not hard gates. They solve the "I forgot to plan"
and "I never run retro" problems without being obnoxious. The developer can
ignore them on trivial sessions.

#### Implementation Quality Hooks

| Hook | Event | Action |
|------|-------|--------|
| Go vet on save | `PostToolUse` (Edit/Write on `api/**/*.go`) | Run `go vet` and surface errors |
| ESLint on save | `PostToolUse` (Edit/Write on `web/src/**`) | Run ESLint on changed file |

These run during autonomous implementation. The agent sees the lint/vet output
and fixes issues inline without human involvement.

**What we explicitly chose NOT to hook (and why):**

- **Pre-commit test gate**: Rejected. The agent should run tests as part of
  implementation (guided by the implementation rule), not be blocked by a hook.
  Full test suites are too slow for a pre-commit gate.
- **Auto-format on save**: Rejected. Formatting changes mid-session create
  noisy diffs that confuse the agent. Formatting is enforced by CI.
- **Mandatory review before PR**: Rejected. `/pr` already includes review
  as an integrated step. A separate hook would be redundant and would block
  draft PRs or trivial fixes.
- **Hard-block SessionStart**: Rejected. A hard gate that forces `/plan` on
  every session would be obnoxious for quick fixes, config changes, or
  exploratory sessions. A nudge is sufficient.

### What We're NOT Building (Yet)

**Persistent agents** (`.claude/agents/`): Deferred. Agents with cross-session
memory are powerful but add complexity. We'll evaluate after using skills for
several sessions to identify what patterns a persistent agent would learn that
skills can't capture. The risk of premature agent creation is an agent that
accumulates stale or wrong memories and degrades output quality.

**Separate `/review` and `/chaos` skills**: These could exist as standalone
skills for ad-hoc use, but the primary invocation path is through `/pr`. If
retros reveal that developers want to run adversarial review or chaos analysis
at other moments (e.g., during design, or mid-implementation on complex
changes), we'll extract them into standalone skills at that point.

## Alternatives Considered

### 1. Adopt superpowers framework wholesale

The [superpowers](https://github.com/obra/superpowers) framework provides a
complete methodology with brainstorming, planning, git worktrees, and TDD
enforcement.

**Rejected because:**
- It's opinionated about TDD (red-green-refactor enforced everywhere). DoIt
  has integration tests and visual regression tests that don't fit strict TDD.
- It creates spec and plan files in `docs/superpowers/` — adding framework
  artifacts to the repo.
- The brainstorming skill asks questions one-at-a-time, which can be slow for
  experienced developers who already know what they want.
- It treats every phase as requiring human involvement. Our model explicitly
  makes implementation autonomous.
- Better to adopt the *principles* (hard gates at boundaries, mandatory skill
  invocation, evidence-based verification) and implement skills tailored to
  our workflow.

### 2. All project-specific, nothing global

Put everything in `.claude/` per-repo.

**Rejected because:** The core workflow (plan → autonomous build → PR review →
retro) is developer methodology, not project architecture. It should transfer
when starting a new repository. Project-specific rules belong in the repo;
process skills belong with the developer.

### 3. Encode everything in CLAUDE.md

Add workflow instructions directly to CLAUDE.md (e.g., "always plan before
implementing").

**Rejected because:**
- CLAUDE.md instructions are best-effort — the agent can rationalize skipping
  them. Skills are explicit invocations with dedicated prompts.
- CLAUDE.md grows without bound. Adding workflow on top of architecture rules
  makes it harder to maintain and more likely to be compacted away in long
  sessions.
- Skills load only when relevant (~100 tokens for metadata scan, <5k tokens
  when activated). CLAUDE.md loads always.

### 4. Human gates during implementation

Require developer approval at each step: plan → approve → implement step 1 →
approve → implement step 2 → approve → ...

**Rejected because:**
- Implementation is the cheap part. The developer's attention is the expensive
  part. Spending judgment on reviewing intermediate implementation steps is
  misallocated.
- This model was appropriate when AI output was unreliable and needed constant
  supervision. With good plans, rules, and hooks, the agent can execute
  autonomously and the developer reviews the outcome.
- The developer still has the option to watch and intervene — they're just not
  *required* to.

### 5. Heavy hook enforcement (block commits, auto-run tests, mandatory review)

Gate every commit behind full test suite, require `/review` before any PR.

**Rejected because:**
- Hooks are deterministic — they can't make judgment calls. A hook doesn't
  know if you're committing a typo fix or a sync engine rewrite.
- Over-enforcement slows down trivial changes and trains the developer to
  resent the tooling.
- The right model: hooks for fast, objective checks (lint, vet) during
  autonomous implementation. Skills for judgment-heavy steps (review, chaos)
  at boundaries.

### 6. Skills at boundaries without automation (original ADR draft)

The first draft of this ADR proposed `/plan`, `/review`, `/chaos`, and `/retro`
as skills the developer invokes manually.

**Rejected because:**
- It moved the consistency problem instead of solving it. "Remember to invoke
  the skill" is the same failure mode as "remember to do the review."
- No guidance for autonomous implementation — the agent had rules about *what*
  not to do but nothing about *how* to work.
- `/review` and `/chaos` as separate invocations before `/pr` added friction
  without adding value — they should be integrated into the PR workflow.
- No hooks to nudge at session start/end meant planning and retros would be
  skipped on most sessions.

## Consequences

**Benefits:**
- **Developer time is spent where it matters**: Design decisions and PR review
  — the two moments where human judgment has the highest leverage.
- **Implementation accelerates**: No human gates during the build phase. The
  agent works autonomously, constrained by rules and hooks, and the developer
  re-engages at PR time.
- **Consistency without discipline**: Hooks nudge at session boundaries. `/pr`
  includes review automatically. The developer doesn't need to remember — the
  system prompts them.
- **Graceful degradation**: Everything is a nudge, not a hard block. Quick
  fixes can skip `/plan`. Trivial PRs can skip `/pr`. The system supports
  both rigorous and lightweight sessions without fighting the developer.
- **Portability**: Global skills transfer to any new repository. Only project
  rules need to be recreated per-repo.
- **Self-improving**: `/retro` captures learnings that become new memories,
  rules, or skills — the system gets better over time.
- **Plan deviation is handled**: The deviation protocol prevents both silent
  adaptation and indefinite blocking — the two failure modes of autonomous
  implementation.

**Costs:**
- **Bad plans are expensive**: If the design is wrong, autonomous implementation
  produces a lot of wrong code quickly. Mitigation: `/plan` forces explicit
  tradeoff discussion and the deviation protocol catches wrong assumptions
  during implementation.
- **PR reviews are heavier**: Since the developer skips intermediate review,
  the PR review must be thorough. `/pr` automates the adversarial review and
  chaos analysis, but the developer must actually engage with the findings.
- **Nudges can be ignored**: Session hooks are reminders, not gates. A
  developer in a hurry will dismiss them. This is intentional — the
  alternative (hard gates) creates resentment — but it means consistency
  depends on the developer choosing to engage with the nudges most of the time.
- **Implementation rule needs tuning**: The autonomous implementation behavior
  (commit frequency, test cadence, deviation threshold) will need adjustment
  based on real usage. Start with the documented defaults and refine via
  `/retro` feedback.
- **Maintenance**: Skills and rules need updating as workflow evolves. Stale
  skills are worse than no skills (they waste context and produce irrelevant
  output). `/retro` should catch staleness.

## Implementation Plan

1. Create global skills in `~/.claude/skills/`:
   - `plan/SKILL.md` — design gate with deviation clause
   - `pr/SKILL.md` — self-contained: review + chaos + PR creation
   - `retro/SKILL.md` — session retrospective
2. Create project rules in `.claude/rules/`:
   - `go-backend.md` — event sourcing, testing, HLC constraints
   - `react-frontend.md` — Dexie, offline-first, Safari, accessibility
   - `implementation.md` — branch, commit, test, deviation, stuck protocols
3. Add hooks to project `.claude/settings.json`:
   - `SessionStart` — nudge to plan
   - `Stop` — nudge to retro
   - `PostToolUse` — go vet on Go files, ESLint on frontend files
4. Use the setup for 3-5 sessions, run `/retro` at each session end
5. Evaluate: what's working, what's ignored, what's missing?
6. Decide on persistent agents based on patterns observed in retros
7. Consider extracting `/review` and `/chaos` as standalone skills if retros
   show demand for ad-hoc invocation outside the PR flow
