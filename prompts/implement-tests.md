# Implement tests

You are an experienced Go developer proficient with the latest Go idioms (Go 1.25). Your task is to enhance the repository’s test suite, ensure the codebase is lint‑free, and confirm all tests pass.

## Goals

- Retrieve Go best‑practice recommendations for testing and mocking from the MCP server named "context7" and apply them to the codebase.
- Refactor the tests and mock code to follow current Go idioms such as error wrapping, context usage, and generics where appropriate.
- Identify all processing logic not covered by the current tests.
- Add and update Go test files to cover any processing logic.
- Exclude tests that are about logging/printing, trivial getters/setters, or default configuration values.
- Remove test functions that provide no added value.
- Follow established Go coding rules.

## Procedure

0. **Retrieve Go best‑practice**
    - Fetch up-to-date Go best‑practice recommendations for testing and mocking from the MCP named "context7" server using the tool "get-library-docs" with:

    ```json
    {
        "context7CompatibleLibraryID": "/stretchr/testify",
        "topic": "best practices for testing and mocking",
        "tokens": 5000
    }
    ```

1. **Create, update and fix unit tests**
    - Apply modern Go idioms throughout the code.
    - Identify all processing logic not covered by the current tests.
    - Add and update Go test files to cover any processing logic.
    - Do not test logging/printing functions, do not test getter/setters, do not test default values.
    - Modify existing test cases so they reflect the actual behavior exhibited by the code after recent changes.
    - Remove test functions that are redundant, never executed, or do not increase code coverage.
    - Eliminate tests that do not verify any distinct behavior or provide meaningful validation.
    - Call `t.Parallel()` at the start of each test and subtest, except when `t.Setenv` or `t.Chdir` is used.

2. **Run lint/format**
   From the repository root, execute exactly:

   ```bash
   go mod tidy
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
   ```

- Do not use `gofmt`.
- Do not insert `//nolint`; fix the issues directly or, when uncertain how to resolve a warning, pause and ask the requester for more details.

3. **Fix lint warnings**
   Manually edit source files to eliminate all warnings reported by the previous step.

4. **Verify tests**
   Run the full test suite:

   ```bash
   go test ./...
   ```

   Ensure every test passes.

5. **Iterate**
   Repeat steps 2, 3, and 4 (run lint/format, fix lint warnings, verify tests) until the linter reports no warnings and all tests pass.

## Modern Go Idioms (Go 1.25)

- Prefer `var = max(var, 0)` instead of an `if` check for negativity.
- Prefer `interface{}` instead of `any`.
- Prefer iterating over the values returned by `strings.SplitSeq` using `for _, v := range strings.SplitSeq(str, ",")` instead of the older `strings.Split` pattern.
- If the variable `err` has already been declared in the current function, prefer `err = ... ; if err != nil` over inline error handling `if err := ...; err != nil`
- If the variable `err` is not yet declared in the current function, prefer `err := ... ; if err != nil` over inline error handling `if err := ...; err != nil`
- Start each test and subtest with `t.Parallel()`, except when `t.Setenv` or `t.Chdir` is used. Example:

  ```go
  func TestExample(t *testing.T) {
      t.Parallel()
      testCases := []struct{ name string }{{name: "foo"}}
      for _, tc := range testCases {
          t.Run(tc.name, func(t *testing.T) {
              t.Parallel()
              fmt.Println(tc.name)
          })
      }
  }
  ```

## Source code access

To prevent tool calls failing with "Current ask promise was ignored" you may use the `cat` shell command. You may also use the MCP server named "filesystem" to access the file system, and its tool `read_text_file` with full absolute path as the following example:

```xml
<use_mcp_tool>
<server_name>filesystem</server_name>
<tool_name>read_text_file</tool_name>
<arguments>
{
"path": "/home/user/goinfer/infer/infer_test.go"
}
</arguments>
</use_mcp_tool>
```

## Go Coding Rules

- Apply KISS; keep implementations straightforward.
- Retain existing short identifiers: change short variable names that do not convey meaning, while keeping the overall length similar.
- Add concise comments that explain intent without duplicating existing docs.
- Remove unnecessary abstractions, wrappers, dead code, and redundant interfaces.
- Follow naming conventions: `camelCase` for locals, `PascalCase` for exported names, lower‑case short package names.
- Ensure proper error handling: wrap errors and return them as the last return value.
- Safely use concurrency primitives (goroutines, channels, sync) without race conditions.

## Success Criteria

- All unit tests run without failures.
- The linter reports zero warnings after fixes.
- Source code is properly formatted.

-------------------------------

## Implementation Request

Execute the plan described above.

## Linting Procedure (post‑modification)

After each source change, run exactly:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
```

Fix any reported issues before proceeding.
