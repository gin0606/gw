# CLAUDE.md

## Design Principles

- Features achievable through hooks are NOT built into the tool itself. Always prefer hooks over new built-in functionality.
- Do not reimplement git behavior. Delegate to git commands via the `internal/git` package.

## Specification

`SPECIFICATION.md` defines the expected behavior, including conventions not evident from the code (stdout/stderr usage, rejection of undefined arguments).

## Verification

CI runs these:

```sh
goimports -l .        # should produce no output
go vet ./...
staticcheck ./...
go test ./...
```
