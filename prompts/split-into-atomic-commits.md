# Split a large Git commit into atomic commits

## Goal  

Re‑organize the diff in the **Goinfer** repository into a series of logically independent commits. Each commit may span multiple files but must be safe to reorder or cherry‑pick without further manual adjustments. The entire process must run autonomously, ending with a clean working tree and a series of well‑documented commits.

## Overview  

The workflow is a loop that repeatedly examines the current diff, isolate the smallest conceptual change, stage only the hunks belonging to that change, create a commit with a descriptive Gitmoji‑style message, and then repeat until `git status --porcelain` reports no remaining modifications. No interactive prompts are allowed; all staging and committing must be performed non‑interactively.

## Detailed Workflow  

### Step 1 – Analyze the complete diff and determine conceptual groups  

Read the full output of `git diff`. For every changed line, determine its conceptual purpose. This conceptual purpose may span several files, such as, for example, a symbol renamed across the codebase, code block moved from one place to other ones, updated code in one file and its updated documentation in another one, the same fix applied in different files. Group together all lines that share a similar purpose, even when they appear in different files. Treat each group as a candidate independent commit.

### Step 2 – Identify the minimal conceptual change  

Count the number of modified lines in each conceptual group. Select the group with the fewest modified lines; this becomes the **Current Change** for this current iteration. If two groups have equal number of modified lines, prioritize the the most significant one (e.g., code change is more significant than version bump).

### Step 3 – Map hunks to the Current Change and stage them  

Process every file listed by the diff.

*For deleted files*:  
Restore the file temporarily with `git restore -- "$file"`. Edit the restored file to remove only the lines that belong to the Current Change, stage the edited file using `git add -- "$file"`, then delete the file with `rm -- "$file"`.

*For existing files*:  
Generate a zero‑context diff with `git diff -U0 -- "$file"`. Examine each hunk and decide whether it belongs to the Current Change. Build a newline‑separated string where each line is `y` for a hunk to stage and `n` for a hunk to ignore. Feed this string to `git add -p` non‑interactively with `printf "%s\n" "$answers" | git add -p -- "$file"`. Only hunks that correspond to the Current Change will be staged; all other modifications remain unstaged.

### Step 4 – Compose the commit message  

Carefully examine the staged diff with `git diff --staged`. Write a commit message that follows the **Gitmoji** convention:

* Title: start with an appropriate emoji and succinctly state the intent.  
* Body: list atomic aspects in order of importance, and cover intent, purpose, motivation, rationale, benefits, trade‑offs, testing considerations, and the affected symbols (packages, files, classes, functions, variables).

Ensure the message is clear enough for future maintainers to understand the “why” without consulting the diff.

### Step 5 – Commit with fixed author and committer metadata  

Create the commit using the predetermined identity values:  

```bash
GIT_AUTHOR_NAME="Oliver" GIT_AUTHOR_EMAIL="oliver@LM4eu.eu" \
GIT_COMMITTER_NAME="GPT OSS 120B RooCode" GIT_COMMITTER_EMAIL="g120r@LM4eu.eu" \
git commit -m "$commit_message"
```

### Step 6 – Loop control  

Run `git status --porcelain` to check for any remaining modifications. If the command produces output, return to **Step 1** to process the next conceptual change. If the output is empty, the working tree is clean and the loop terminates.

## Final Summary  

When the loop terminates, output a concise list of all newly created commits. For each commit, display the commit message. This summary provides a quick verification that the diff has been successfully split into independent, well‑documented commits.
