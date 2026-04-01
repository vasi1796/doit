---
name: pr
description: >
  Use when implementation is complete and ready for PR. Triggers on: "open a PR",
  "create PR", "ready for review", "let's PR this", or when the implementation
  plan is fully executed.
disable-model-invocation: false
context: fork
allowed-tools: Read, Grep, Glob, Bash
---

## PR Gate — Review, Analyze, Ship

You are preparing a pull request. This is the second high-leverage gate where
human judgment catches what autonomous implementation missed. Run all steps
before presenting findings to the developer.

### Step 1: Diff Analysis

```bash
git diff main...HEAD --stat
git diff main...HEAD
git log main..HEAD --oneline
```

Understand the full scope of changes — every file modified, every commit made.
This is what the reviewer will see.

### Step 2: Adversarial Review

For EACH changed file, systematically check:

**Correctness:**
- What breaks if input is empty, null, max-size, or wrong type?
- Are there off-by-one errors in loops, slices, or array indexing?
- What happens if this function is called twice? Is it idempotent?
- Are error return values checked? Any silently swallowed errors?

**Security:**
- Is user input validated and sanitized before use?
- Are SQL queries parameterized (no string concatenation)?
- Is authentication/authorization checked on every endpoint?
- Are there any leaked secrets, tokens, or credentials?

**Design:**
- Does this change violate any rule in CLAUDE.md?
- Is there a simpler way to achieve the same result?
- Is scope creep present — changes beyond what the plan specified?
- Will this be obvious to someone reading it in 6 months?

### Step 3: Chaos Analysis (Conditional)

Run this step ONLY if the changes touch any of: events, projections, sync
engine, workers, CRDT logic, message broker, outbox, or database transactions.

For each distributed/async component affected:

**Crash failures:**
- What happens if this process dies mid-operation?
- Is there partial state left behind? Is it recoverable?

**Network failures:**
- What if the connection drops between step N and N+1?
- What if a request succeeds but the response is lost?

**Concurrency:**
- What if two users/devices do this simultaneously?
- What if events arrive out of order?
- What if the same event is processed twice — is it idempotent?

**Resource:**
- Does anything grow unbounded over time?
- What if the queue backs up significantly?

### Step 4: Present Findings

Organize all findings by severity:

- **CRITICAL** — Must fix before merge. These block the PR.
- **WARNING** — Should fix. Present to developer for decision.
- **SUGGESTION** — Consider improving. Non-blocking.

For each finding:
- File and line number
- What's wrong
- How to fix it

If no issues found, say so explicitly — don't invent problems to look thorough.

### Step 5: Fix Critical Issues

Fix any CRITICAL findings before proceeding. Present the fixes to the developer.

### Step 6: Create PR

After the developer has reviewed findings and approved:

Generate the PR using `gh pr create` with:
- **Title**: Short, conventional commit format (under 70 chars)
- **Summary**: 1-3 bullet points explaining what and why
- **Test plan**: How to verify the changes work
- **Review findings**: Any unresolved WARNINGs or SUGGESTIONs

Format:
```
gh pr create --title "type(scope): description" --body "$(cat <<'EOF'
## Summary
- bullet points

## Test plan
- [ ] verification steps

## Review findings
- Any warnings or suggestions from automated review

Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### What NOT to do

- Do not skip the adversarial review because "the changes are simple"
- Do not create the PR before presenting findings to the developer
- Do not suppress findings to make the PR look clean
- Do not fix WARNING-level issues without asking — the developer may disagree
