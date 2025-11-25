# Test Coverage TODO

## Current state

- `cmd/agent-align`: 23.2% (missing tests for most helpers and the CLI entry logic)
- `internal/config`: 82.6%
- `internal/syncer`: 89.1%

## Coverage goals

1. **Exercise CLI flag helpers** – write table-driven tests for `parseAgents`.
   Cover `parseSelectionIndices`, `resolveExecutionMode`, and `validateCommand`.
   These cases prove the basic flow control logic stays correct.
2. **Cover the interactive helpers** – supply synthetic readers for `promptSourceAgent`.
   Exercise `promptTargetAgents` with those inputs to assert invalid-input loops stay
   aligned.
3. **Test file operations** – verify `writeConfigFile` (and `ensureConfigFile`) creates
   directories/files correctly and surfaces helpful errors when writes fail.
4. **Signal incremental progress** – rerun `go test ./...` after each of the earlier steps.
   This keeps the suite green before moving to the next item.

Completing these steps should push `cmd/agent-align` coverage well above the
current 23% while leaving the already strong `internal/*` packages untouched.
