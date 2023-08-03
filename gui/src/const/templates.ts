import { LmTemplate } from "@/interfaces";

const alpaca = `### Instruction: {prompt}

### Response:`;

const vicuna1 = `### User: {prompt}

### Assistant:`;

const vicuna2 = `USER: {prompt}

ASSISTANT:`;

const orca2 = `### System:
You are an AI assistant that follows instruction extremely well. Help as much as you can.

### User:
{prompt}

### Response:`;

const orca2context = `### System:
You are an AI assistant that follows instruction extremely well. Help as much as you can.

### User:
{prompt}

### Input: {context}

### Response:`;

const templates: Record<string, LmTemplate> = {
  alpaca: {
    name: "Alpaca",
    content: alpaca,
  },
  vicuna1: {
    name: "Vicuna1",
    content: vicuna1,
  },
  vicuna2: {
    name: "Vicuna2",
    content: vicuna2,
  },
  orca2: {
    name: "Orca2",
    content: orca2,
  },
  orca2context: {
    name: "Orca2 context",
    content: orca2context,
    vars: ["context"]
  },
}

export {
  templates,
}