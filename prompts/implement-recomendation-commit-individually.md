# Go Codebase Audit, Refactor, and Commit (Enhanced)

## Role  

Act as a senior Go engineer with deep expertise in static analysis, Go idioms, SOLID principles, KISS, and Emoji Commits. Your responsibility is to transform the repository into a clean, maintainable, and production‑ready state.

## Objective  

Conduct a comprehensive audit of every Go source file, produce actionable improvement recommendations, apply those recommendations iteratively, and generate a series of well‑structured commits that leave the repository lint‑free, test‑passing, formatted according to `golangci-lint-v2`, and ready for release.

## Scope of Analysis  

1. Load each `.go` file, parse its AST, and read the source multiple times to achieve full comprehension of logic and intent.  
2. Preserve all exported symbols and public APIs unless a modification is required for correctness, security, or performance.  
3. Keep existing short identifiers; rename only when ambiguity forces a change, and ensure the new name does not exceed the original length.  
4. Add succinct comments that explain why code exists, avoiding duplication of existing documentation.  
5. Apply the KISS principle: favour simple, direct implementations and eliminate unnecessary abstractions/functions.  
6. Enforce consistent naming conventions, including camelCase for variables, PascalCase for exported identifiers, and appropriate package‑level naming.  
7. Detect and remove dead code, redundant interfaces, and superfluous wrapper functions.  
8. Verify that error handling follows idiomatic Go patterns, and that panics are used only for unrecoverable states.  
9. Ensure that concurrency constructs (goroutines, channels, sync primitives) are used safely and documented.  

## Output Specification  

### Findings  

Present the findings (rationale and location).

### Recommendations  

For each finding, deduce the corresponding recommendation. Deduplicate similar recommendations. Present the recommendations (rationale, location, change to apply). Consolidate recommendations that address a similar pattern across multiple files into a single entry.

## Procedure  

1. Execute the audit and output the Findings and Recommendations exactly as described.  
2. For each recommendation, perform the following cycle:  
   a. Modify the source to satisfy the recommendation.  
   b. Run exactly the following pipeline in the repository root:  

   ```bash
   go mod tidy
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
   go test ./...
   ```  

   c. If any step fails, revise the change until the pipeline completes without errors.  
   d. When the pipeline succeeds, generate a commit message that follows the Emoji‑Conventional style with a title that summarizes the purpose and a body that describes the rationale, the affected component, and any impact on the public API. Then stage the changes with `git add -u` and commit using separate `-m` flags for the title and body lines.  
   e. Continue with the next recommendation.  
