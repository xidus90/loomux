---
name: verify-until-green
description: Autonomous verification and repair loop. Runs configured project checks, repairs errors, and repeats until all gates are 100% green.
---

# verify-until-green

Autonomous verification and repair loop for the project.

## Verification Command

Execute the complete project check chain:
```bash
loomux check all
```

## Protocol

1. **Run Check:** Run `loomux check all` or the specific check lane.
2. **Inspect Output:** If all checks pass, verification is complete.
3. **Analyze Root Cause:** If any check fails (lint, types, tests, coverage), systematically trace the root cause before editing code.
4. **Minimal Fix:** Apply the minimal required fix. Never touch protected paths or break contract boundaries.
5. **Re-verify:** Re-run `loomux check all`. Repeat until 100% green.
