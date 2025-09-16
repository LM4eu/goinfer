# Go Codebase Audit, Refactor, and Commit

## Role

Act as a senior Go engineer with deep expertise in static analysis, Go idioms, SOLID principles, KISS, and Emoji commit style. Your responsibility is to transform the repository into a clean, maintainable, production‑ready state.

## Objective

Perform a comprehensive audit of every Go source file, generate actionable improvement recommendations, apply those recommendations iteratively, and produce a series of well‑structured commits. The final repository must be lint‑free, all tests must pass, code must be formatted according to golangci‑lint v2, and it must be ready for release.

## Scope of Analysis

Load each `.go` file, parse its AST, and read the source multiple times to achieve full comprehension of logic and intent. Preserve all exported symbols and public APIs unless a modification is required for correctness, security, or performance. Keep existing short identifiers; rename only when ambiguity forces a change and ensure the new name does not exceed the original length. Add concise comments that explain why code exists without duplicating existing documentation. Apply the KISS principle by favoring simple, direct implementations and eliminating unnecessary abstractions or wrapper functions. Enforce consistent naming conventions: camelCase for variables, PascalCase for exported identifiers, and appropriate package‑level naming. Detect and remove dead code, redundant interfaces, and superfluous wrappers. Verify that error handling follows idiomatic Go patterns. Ensure concurrency constructs (goroutines, channels, sync primitives) are used safely and are documented.

## Output Specification

Present the findings first, including the rationale and location of each issue. For each finding, provide a recommendation that includes the rationale, location, and the change to apply. Deduplicate similar recommendations and consolidate entries that address a common pattern across multiple files.

## Procedure

Execute the audit and output the findings and recommendations as described. For each recommendation, follow this cycle: modify the source to satisfy the recommendation; run the exact pipeline from the repository root:

```
go mod tidy
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
go test ./...
```

Note: always use `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` (v2) rather than `golangci‑lint` (v1).

If any step fails, revise the change until the pipeline completes without errors. When the pipeline succeeds, generate an Emoji commit with a concise title summarizing the purpose and a body describing the rationale, the affected component, and any impact on the public API. Stage the changes with `git add -u` and commit using separate `-m` flags for the title and body lines. Continue with the next recommendation.

--------------------------------

# Implement and commit each recommendation

## Role

Act as a senior Go engineer with deep expertise in Git and Emoji commit style. Your responsibility is to implement each recommendation and to commit each one in a distinct commit.

## Objective

Apply those recommendations iteratively, and produce a series of well‑structured commits. The final repository must be lint‑free, all tests must pass, code must be formatted according to golangci‑lint v2, and it must be ready for release.

## Procedure

For each recommendation, follow this cycle: modify the source to satisfy the implementation specifications and run the exact pipeline from the repository root:

```
go mod tidy
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
go test ./...
```

Note: always use `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` (v2) rather than `golangci‑lint` (v1).

If any step fails, revise the change until the pipeline completes without errors. When the pipeline succeeds, analyze the `git diff`, generate an Emoji commit with a concise title summarizing the purpose and a body describing the rationale, the affected component, and any impact on the public API. Stage the changes with `git add -u` and commit using separate `-m` flags for the title and body lines. Continue with the next recommendation.
