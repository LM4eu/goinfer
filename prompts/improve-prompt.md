# Improve prompt

**Task**: As a prompt engineer, process the *Provided Prompt* (the text that appears after the line containing three hyphens). Follow the workflow below and output the final revised prompt.

### Definitions

- **Provided Prompt**: Text following the line `---` in the user message.  
- **Corrected Draft**: The Provided Prompt with all spelling, punctuation, and grammar errors fixed.  
- **Final Revised Prompt**: The refactored version of the Corrected Draft, ready for AI execution.

### Step 1 – Proofread

1. Read the Provided Prompt thoroughly to grasp its intent.  
2. Fix any spelling, punctuation, or grammatical errors.  
3. Save the result as *Corrected Draft*.

### Step 2 – Ambiguity Check

- If any part of the Corrected Draft is unclear:  
  1. Output the Corrected Draft in a code block, enclosing each ambiguous segment in `<<…>>`.  
  2. List each ambiguity, explain why it is unclear, and give **at least two** alternative phrasings.  
  3. **Stop** after this output (do not proceed to Step 3).

### Step 3 – Refactor (only if no ambiguities)

Using the Corrected Draft, rewrite it so an AI can execute it, applying the **Refactoring Guidelines**:

#### Refactoring Guidelines

- **Structure**: Rearrange, split, combine, or reorder bullet-point blocks for clarity.  
- **Headings**: Add headings whenever a new idea, step, or argument begins; each heading must have a clear purpose.  
- **Paragraph Flow**: Order paragraphs logically; rephrase unclear sentences.  
- **Sentence Conciseness**: Merge short, related sentences into concise statements.  
- **Navigation**: Insert missing headings or improve existing ones.  
- **Logical Flow**: Ensure each section naturally leads to the next.  
- **Clarity**: Rephrase as needed for maximum clarity.  
- **Thematic Division**: Group content into thematic sections.  
- **Section Order**: Follow a natural progression (e.g., background → method → conclusion).  
- **Redundancy**: Remove duplicate content by merging overlapping sections.

### Cyclical Review

1. Review line-by-line, identify any remaining ambiguities, and rewrite to resolve them.  
2. Repeat until the prompt is unambiguous, preserves the original intent, and uses the fewest tokens possible.

### Final Output

1. Echo the original *Provided Prompt* exactly as received.  
2. Print a blank line.  
3. Print a line containing three hyphens (`---`).  
4. Print a blank line.  
5. Print the *Final Revised Prompt* you have produced.
