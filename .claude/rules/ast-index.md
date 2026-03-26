---
description: Use ast-index for all code search tasks in this project
paths: 
  - "**/*.go"
---

# ast-index Rules

## Mandatory Search Rules

1. **ALWAYS use ast-index FIRST** for any code search task
2. **NEVER duplicate results** — if ast-index found usages/implementations, that IS the complete answer
3. **DO NOT run grep "for completeness"** after ast-index returns results
4. **Use grep/Search ONLY when:**
   - ast-index returns empty results
   - Searching for regex patterns (ast-index uses literal match)
   - Searching for string literals inside code (`"some text"`)
   - Searching in comments content

## Why ast-index

ast-index is 17-69x faster than grep (1-10ms vs 200ms-3s) and returns structured, accurate results.

## Command Reference

| Task | Command |
|------|---------|
| Find structs/interfaces | `ast-index class "TypeName"` |
| Find functions/methods | `ast-index symbol "FuncName"` |
| Find usages | `ast-index usages "SymbolName"` |
| Find callers | `ast-index callers "funcName"` |
| Find interface implementations | `ast-index implementations "InterfaceName"` |
| Show file structure | `ast-index outline "file.go"` |
| Show imports | `ast-index imports "file.go"` |
| Find goroutine usages | `ast-index search "go FuncName"` |
| Find error handling | `ast-index search "errors.Is"` |