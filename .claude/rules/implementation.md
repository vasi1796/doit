---
paths: []
---

# Autonomous Implementation Behavior

These rules govern how the agent works during the implementation phase — after
the plan is approved and before the PR is created. The developer is not actively
supervising during this phase.

## Branching
- Create a feature branch from main before starting implementation
- Branch naming: `feat/short-description`, `fix/short-description`, or
  `refactor/short-description`

## Commit Discipline
- Commit in logical chunks that match plan steps — not one giant commit at
  the end, not one commit per line changed
- Use conventional commit messages: `type(scope): description`
- Each commit should be a coherent, reviewable unit of work

## Testing
- Run relevant tests after each significant change — don't wait until the end
  to discover the first change broke everything
- For Go changes: `cd api && go test ./... -count=1 -short`
- For frontend changes: `cd web && npm run lint && npm test`
- Fix test failures immediately before moving to the next plan step

## Scope Discipline
- Implement exactly what the plan specifies — nothing more
- Do not refactor code outside the plan scope
- Do not add features not in the plan
- Do not "improve" adjacent code, add docstrings to unchanged functions, or
  clean up unrelated files
- If you notice something worth improving, note it for the PR description —
  don't fix it now

## Deviation Protocol
- If an assumption from the plan doesn't hold, STOP
- Present what changed: "Step 3 assumed X, but I found Y"
- Propose an adjustment: "I recommend changing the approach to Z because..."
- Wait for developer approval before continuing
- Do NOT silently adapt the plan and keep going

## Stuck Protocol
- If blocked after 2-3 genuine attempts at a step, surface the blocker
- Show what you tried and why it failed
- Suggest what information or decision you need from the developer
- Do not go in circles retrying the same approach

## Quality During Implementation
- Hooks will run go vet and ESLint automatically on file saves — fix any
  issues they surface before moving on
- Do not suppress or ignore hook output
