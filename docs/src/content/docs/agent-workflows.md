---
title: Agent workflows
description: How agents should call grepai for find, explore, and before-edit
---

Three compact workflows for AI agents. Prefer **compact** MCP responses and exact `grep` for identifiers.

## find-symbol

1. `grepai grep` / `grepai_grep` for the exact identifier
2. If no hit, `grepai search` with a short natural-language query
3. Use `trace_callers` before editing a shared symbol

## explore-concept

1. `grepai search` for the concept (one sentence)
2. Open top paths with the host file reader
3. RPG tools only if search is empty or too broad

## before-edit

1. Exact grep for the symbol you will change
2. `trace_callers` (and callees if the API surface moves)
3. After the edit batch: `grepai reindex --paths …` or rely on adaptive watch debounce
4. Optional: `grepai watch --freeze` during a large multi-file rewrite, then `--unfreeze`

## Host profile tips

```yaml
watch:
  debounce_ms: 500
  debounce_max_ms: 5000
  debounce_adaptive: true
  discover_worktrees: false
chunking:
  strategy: ast   # treesitter build
```
