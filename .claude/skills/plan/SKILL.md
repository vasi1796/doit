---
name: plan
description: >
  Use when starting any feature, bugfix, or refactor. Triggers on: building
  something new, implementing a change, starting a task, or when SessionStart
  hook prompts "what are you building?"
disable-model-invocation: false
---

## Design Gate

You are in the planning phase. **No code is written until the plan is approved.**
This is the highest-leverage moment — catching a wrong approach here saves an
entire implementation cycle.

### Step 1: Explore

Read the relevant code, documentation, and tests. Understand the current state
before proposing changes. Do not assume you know the structure — verify it.

Check:
- CLAUDE.md and AGENTS.md for architectural constraints
- Existing code in the area you'll be modifying
- Related tests to understand expected behavior
- Recent git history for context on why things are the way they are

### Step 2: Ask

Ask the user 2-3 clarifying questions about intent, scope, and constraints.

Rules:
- Ask questions **one at a time**, not as a wall of text
- Focus on questions that would change your approach if answered differently
- Skip this step only if the user has already provided a detailed spec

### Step 3: Propose

Present 2-3 alternative approaches with explicit tradeoffs. Always include the
simplest option even if it feels too simple.

For each approach:
- **What**: One-sentence description
- **How**: Key files to create/modify
- **Tradeoff**: What you gain and what you give up
- **Risk**: What could go wrong

Recommend one approach and explain why.

### Step 4: Wait

Wait for explicit user approval before proceeding. Do not start implementing
on a "sounds good" — confirm which approach they want.

### Step 5: Plan Output

After approval, break the work into steps that take 2-5 minutes each:
- Exact file paths to create or modify
- What changes in each file and why
- Verification step for each change (test command, expected output)

### Deviation Clause

Include this in every approved plan:

> **If during implementation you discover the plan is wrong** — an assumption
> doesn't hold, a dependency doesn't work as expected, or the approach hits an
> unforeseen blocker — **stop implementation, present what changed and why, and
> wait for the developer to approve a revised plan before continuing.** Do not
> silently adapt the plan. Do not block indefinitely without surfacing the issue.

### What NOT to do

- Do not write code, scaffolding, or stubs during planning
- Do not open files in an editor "to prepare"
- Do not create branches or make git changes
- Do not skip planning because the task "looks simple"
- Do not present a single approach as if it's the only option
