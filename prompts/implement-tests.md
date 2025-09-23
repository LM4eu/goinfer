# Implement tests

## Goal

You are an **experienced** Go developer highly skilled about the latest Go idioms introduced in the latest Go versions. You must create, update and fix the tests for the Go repository, ensure the codebase is lint‑free, and verify that all tests pass.

## Procedure

1. **Create unit tests**  
   Add Go test files covering the exported functions and all other functions doing some processing. Update the existing tests to reflect the current behavior of the code. Identify and remove all test function without any read added value.

2. **Run lint/format**  
   From the root directory of the repository, run exactly the following commands without any changes:

   ```bash
   go mod tidy
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
   ```

   The first command `go mod tidy` fixes issues about dependencies. The second command runs the linters, formats the code, applies some automatic fixes, and lists any remaining warnings. Therefore, do not use `gofmt`. Do not insert the `//nolint` in the code source, instead fix the lint issues seriously, or ask the user for clarification.

3. **Fix lint warnings**  
   Manually edit the code to eliminate all warnings reported in step 2.

4. **Verify tests**  
   Run the test suite (e.g., `go test ./...`) and ensure every test passes.

5. **Iterate**  
   Repeat steps 2–4 until the lint command reports no warnings and all tests succeed.

## Modern Go idioms

The project targets the latest Go release, version 1.25. As the source code uses a very recent Go version, use the modern way to modify/introduce any Go code. For example, use:  

- use `var = max(var, 0)` instead of `if var < 0 { var = 0 }`  
- use `any` instead of `interface{}`  
- use `for v := range strings.SplitSeq(str, ",")` instead of `for _, v := range strings.Split(str, ",")`
- use plain assignment `err = ... ; if err != nil` (reuse `err` variable if any), instead of inline error handling `if err = ...; err != nil`
- start each test function by calling `t.Parallel()`
- Start each subtest function by calling `t.Parallel()` such as the following example:  

   ```go
   func TestFunctionRangeMissingCallToParallel(t *testing.T) {
      t.Parallel() // <--- Start the test function by calling t.Parallel()
      testCases := []struct { name string }{{name: "foo"}}
      for _, tc := range testCases {
         t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // <--- Start the subtest function by calling t.Parallel()
            fmt.Println(tc.name)
         })
   }  }
   ```

## Go coding rules

Apply the KISS principles and the following guidelines:

- retain existing short identifiers and rename only when an identifier is ambiguous or misleading, ensuring the new name does not exceed the original length.
- document with concise comments that explain the purpose of code without duplicating existing documentation
- favor simple, direct implementations
- remove unnecessary abstractions or wrapper functions
- enforce consistent naming conventions: camelCase for variables, PascalCase for exported identifiers, and lower‑case, short package names
- detect and eliminate dead code, redundant interfaces, and superfluous wrappers
- verify that error handling follows idiomatic Go patterns, including proper error wrapping and returning errors as the final return value
- ensure that concurrency constructs such as goroutines, channels, and sync primitives are used safely and free of race conditions.

 **Important: use the specified XML or MCP tools to access the files.**

## Success criteria

- All unit tests run without failures.  
- The lint command reports zero warnings after fixes.  
- The source code is properly formatted.

-------------------------------

## Implementation Request

Please implement the described plan.

## Linting Procedure

After every source code modification:

1. Format and lint the code **using exactly** the command below.
2. Resolve any linting issues reported by the tool.

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
```

## Modern Go idioms

The project targets the latest Go release, version 1.25. As the source code uses a very recent Go version, use the modern way to modify/introduce any Go code. For example, use:  

- use `var = max(var, 0)` instead of `if var < 0 { var = 0 }`  
- use `any` instead of `interface{}`  
- use `for v := range strings.SplitSeq(str, ",")` instead of `for _, v := range strings.Split(str, ",")`
- use plain assignment `err = ... ; if err != nil` (reuse `err` variable if any), instead of inline error handling `if err = ...; err != nil`
- start each test function by calling `t.Parallel()`
- Start each subtest function by calling `t.Parallel()` such as the following example:  

   ```go
   func TestFunctionRangeMissingCallToParallel(t *testing.T) {
      t.Parallel() // <--- Start the test function by calling t.Parallel()
      testCases := []struct { name string }{{name: "foo"}}
      for _, tc := range testCases {
         t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // <--- Start the subtest function by calling t.Parallel()
            fmt.Println(tc.name)
         })
   }  }
   ```

-------------------------------

# Implement and fix tests

## Goal

You are an **experienced** Go developer highly skilled about the latest Go idioms introduced in the latest Go versions. You must create, update and fix the tests for the Go repository, ensure the codebase is lint‑free, and verify that all tests pass.

## Procedure

1. **Create unit tests**  
   Add Go test files covering the exported functions and all other functions doing some processing. Update the existing tests to reflect the current behavior of the code. Identify and remove all test function without any read added value.

2. **Run lint/format**  
   From the root directory of the repository, run exactly the following commands without any changes:

   ```bash
   go mod tidy
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
   ```

   The first command `go mod tidy` fixes issues about dependencies. The second command runs the linters, formats the code, applies some automatic fixes, and lists any remaining warnings. Therefore, do not use `gofmt`. Do not insert the `//nolint` in the code source, instead fix the lint issues seriously, or ask the user for clarification.

3. **Fix lint warnings**  
   Manually edit the code to eliminate all warnings reported in step 2.

4. **Verify tests**  
   Run the test suite (e.g., `go test ./...`) and ensure every test passes.

5. **Iterate**  
   Repeat steps 2–4 until the lint command reports no warnings and all tests succeed.

## Modern Go idioms

The project targets the latest Go release, version 1.25. As the source code uses a very recent Go version, use the modern way to modify/introduce any Go code. For example, use:  

- use `var = max(var, 0)` instead of `if var < 0 { var = 0 }`  
- use `any` instead of `interface{}`  
- use `for v := range strings.SplitSeq(str, ",")` instead of `for _, v := range strings.Split(str, ",")`
- use plain assignment `err = ... ; if err != nil` (reuse `err` variable if any), instead of inline error handling `if err = ...; err != nil`
- start each test function by calling `t.Parallel()`
- Start each subtest function by calling `t.Parallel()` such as the following example:  

   ```go
   func TestFunctionRangeMissingCallToParallel(t *testing.T) {
      t.Parallel() // <--- Start the test function by calling t.Parallel()
      testCases := []struct { name string }{{name: "foo"}}
      for _, tc := range testCases {
         t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // <--- Start the subtest function by calling t.Parallel()
            fmt.Println(tc.name)
         })
   }  }
   ```

## Go coding rules

Apply the KISS principles and the following guidelines:

- retain existing short identifiers and rename only when an identifier is ambiguous or misleading, ensuring the new name does not exceed the original length.
- document with concise comments that explain the purpose of code without duplicating existing documentation
- favor simple, direct implementations
- remove unnecessary abstractions or wrapper functions
- enforce consistent naming conventions: camelCase for variables, PascalCase for exported identifiers, and lower‑case, short package names
- detect and eliminate dead code, redundant interfaces, and superfluous wrappers
- verify that error handling follows idiomatic Go patterns, including proper error wrapping and returning errors as the final return value
- ensure that concurrency constructs such as goroutines, channels, and sync primitives are used safely and free of race conditions.

 **Important: use the specified XML or MCP tools to access the files.**

## Success criteria

- All unit tests run without failures.  
- The lint command reports zero warnings after fixes.  
- The source code is properly formatted.

-------------------------------

## Implementation Request

Please implement the described plan.

## Linting Procedure

After every source code modification:

1. Format and lint the code **using exactly** the command below.
2. Resolve any linting issues reported by the tool.

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
```

-------------------------------

## Role

You are an experienced Go developer proficient with the latest Go idioms (Go 1.25). Your task is to enhance the repository’s test suite, ensure the codebase is lint‑free, and confirm all tests pass.

## Goals

- Add and update Go test files to cover exported functions and any processing logic.
- Remove test functions that provide no added value.
- Apply modern Go idioms throughout the code.
- Follow established Go coding rules.
- Use the designated XML or MCP tools to access repository files.

## Procedure

1. **Create and update unit tests**
   - Add test files for exported functions and processing functions.
   - Update existing tests to match current behavior.
   - Delete any test functions that add no value.
   - Call `t.Parallel()` at the start of each test and subtest.

2. **Run lint/format**
   From the repository root, execute exactly:

   ```bash
   go mod tidy
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
   ```

   - Do not use `gofmt`.
   - Do not insert `//nolint`; fix the issues directly or ask for clarification.

3. **Fix lint warnings**
   Manually edit source files to eliminate all warnings reported by the previous step.

4. **Verify tests**
   Run the full test suite:

   ```bash
   go test ./...
   ```

   Ensure every test passes.

5. **Iterate**
   Repeat steps 2–4 until the linter reports zero warnings and all tests succeed.

## Modern Go Idioms (Go 1.25)

- Prefer `var = max(var, 0)` instead of an `if` check for negativity.
- Prefer `interface{}` instead of `any`.
- Prefer `for v := range strings.SplitSeq(str, ",")` over `for _, v := range strings.Split(str, ",")`.
- Prefer plain assignment `err = … ; if err != nil` and reuse the `err` variable.
- Start each test and subtest with `t.Parallel()`. Example:

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

## Go Coding Rules

- Apply KISS; keep implementations straightforward.
- Retain existing short identifiers; rename only ambiguous ones, preserving length.
- Add concise comments that explain intent without duplicating existing docs.
- Remove unnecessary abstractions, wrappers, dead code, and redundant interfaces.
- Follow naming conventions: `camelCase` for locals, `PascalCase` for exported names, lower‑case short package names.
- Ensure proper error handling: wrap errors and return them as the last return value.
- Safely use concurrency primitives (goroutines, channels, sync) without race conditions.

## Source code access

To prevent tool calls failing with "Current ask promise was ignored" you may use the `cat` shell command. You may also use the MCP server named "filesystem" to access the file system, and its tool `read_text_file` with full absolute path.

## Success Criteria

- All unit tests run without failures.
- The linter reports zero warnings after fixes.
- Source code is properly formatted.

## Implementation Request

Execute the plan described above.

## Linting Procedure (post‑modification)

After each source change, run:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --fix
```

Fix any reported issues before proceeding.
