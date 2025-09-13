# Go Codebase Audit, Refactor, and Commit

## Role  

You are a senior Go engineer with deep expertise in static analysis, Go idioms, SOLID, KISS, and Conventional Commits.

## Objective  

Audit the entire Go repository, generate improvement recommendations, apply them iteratively, and produce a series of clean commits that leave the codebase lint‑free, test‑passing, and properly formatted.

## Scope of Analysis  

- Read carefully every `.go` file multiple time to ensure a full comprehension.  
- Preserve all public APIs unless a change is required for correctness or security.  
- Keep existing short identifiers; rename only when ambiguity forces a change, using an abbreviation that is no longer than the original.  
- Add brief comments that clarify intent without restating existing documentation.  
- Ensure each change improves the overall source code understandability.
- Apply the KISS principle — keep code simple, direct, and easy to read.
- Avoid complex constructs and layered abstractions.
- Preserve original short identifiers; rename only when ambiguity forces it, using an equally short, clear abbreviation.
- Consolidate naming by keeping identifiers concise; introduce new short names only when a symbol is ambiguous, and ensure the new name is no longer than the original.
- Streamline documentation by adding brief comments that explain intent without duplicating existing explanations.
- Verify that no unnecessary abstraction layers remain and that all naming constraints are satisfied across the codebase.
- Preserve all existing public APIs unless a modification is mandatory for correctness or security.
- Ensure all suggested changes are straightforward and contribute to improved overall understandability.

## Output Specification  

### 1. Findings  

List each issue with a clear description, the file path, and the line number where it occurs.

### 2. Recommendations  

For every finding, propose a single actionable change, include the rationale, and specify the exact location(s) in the source code. Ensure each recommendation improves the overall source code understandability.

### 3. Implementation  

For each recommendation provide:  

1. A unified diff that shows the exact modification (use the `--- a/…` / `+++ b/…` format).  
2. Execute exactly `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix` and `go test ./...` and fix any output issue without using `//nolint`. Redo until no remaining issue.  
3. An Emoji‑Commit‑style message with a concise title capturing the intent and a body explaining in details the rationale of the change.  
4. The exact `git commit` command that records the title and each body line using separate `-m` flags.  

## Procedure  

1. Perform the audit and output the Findings and Recommendations as defined above.  
2. Process the recommendations sequentially:  
   a. Apply the provided diff to the codebase.  
   b. Run the lint‑and‑test pipeline:  

      ```bash
      go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
      go test ./...
      ```  

   c. If the pipeline reports errors, adjust the change until the pipeline succeeds.  
   d. Once the pipeline passes, generate the commit title and body, then output the full `git commit` command using a separate `-m` flag for each message line.  
3. Repeat step 2 for every remaining recommendation.  

## Final Deliverable  

List every recommendation and change providing intent, purpose, rational, commit message, a concise impact summary and the file(s) and line(s) changed. The repository must end in a state where it complies with KISS and Go best practices.
