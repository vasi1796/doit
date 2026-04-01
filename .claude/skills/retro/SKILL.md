---
name: retro
description: >
  Session retrospective. Use at the end of a work session to capture learnings.
  Triggers on: "retro", "what did we learn", "session recap", "wrapping up",
  or when Stop hook reminds to run retro.
disable-model-invocation: true
---

## Session Retrospective

Review the conversation history and capture what matters for future sessions.

### What worked

- Which approaches succeeded on first try? Why?
- What decisions saved time or avoided problems?
- Were there moments where the plan, rules, or hooks caught something useful?
- Should any successful approach be repeated in future sessions?

### What didn't work

- Where did we go in circles or waste time?
- What assumptions were wrong?
- Did the agent deviate from the plan? Was it caught or did it slip through?
- Were there issues that should have been caught earlier (at plan time vs PR time)?
- Did any hooks or rules fail to catch something they should have?

### Learnings to persist

For each learning, decide where it belongs:

| Type | When to use | Action |
|------|-------------|--------|
| **Feedback memory** | The user corrected an approach or confirmed a non-obvious one | Save to `~/.claude/projects/.../memory/` |
| **Project memory** | Learned something about the project's state, goals, or constraints | Save to `~/.claude/projects/.../memory/` |
| **CLAUDE.md rule** | Discovered an architectural constraint that should be enforced | Propose addition to CLAUDE.md |
| **New rule file** | Found a path-scoped convention worth encoding | Propose `.claude/rules/` file |
| **Skill update** | A skill was missing a step, or a new skill is needed | Propose the change |
| **Nothing** | The learning is already captured or too ephemeral | Skip |

### Action items

Present a summary to the developer:

1. **Memories to save** (with draft content)
2. **Rules to add or update** (with draft content)
3. **Skills to modify** (with what to change)
4. **CLAUDE.md changes** (if any)

Wait for the developer to approve each item before saving.

### What NOT to do

- Do not save memories about ephemeral task details (file paths changed, bugs fixed)
- Do not save what's already in git history or code comments
- Do not create duplicate memories — check existing ones first
- Do not save without developer approval
- Do not skip the retro because "nothing interesting happened" — even confirming
  that the workflow worked smoothly is worth noting
