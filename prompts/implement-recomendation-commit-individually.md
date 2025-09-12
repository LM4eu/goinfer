## Task Flow

Skip the recommendations #4 and #5. Process only the recommendations #1, #2, #3, #6, and #7 in order.

After each recommendation is implemented, run the following commands:

```bash
golangci-lint-v2 run --fix
go test ./...
```

If any lint errors or test failures appear, fix them and rerun the commands until there is no output.

When the current recommendation has been fully applied and passes linting and testing with no errors/failures, examine the changes. Determine the intent, purpose, and rationale of the modifications. Using this analysis, write a commit message that follows conventional best practices: a concise title that captures the intent, purpose, and recommendation, followed by a blank line and a detailed body describing the rationale and the specific changes made.

Commit the changes with the generated commit message, then move on to the next recommendation. Continue this cycle until all specified recommendations have been processed.