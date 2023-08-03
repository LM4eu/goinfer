interface ModelStateContract {
  models: Array<string>;
  isModelLoaded: boolean;
  loadedModel: string;
  ctx: number;
}

interface InferResultContract {
  text: string;
  thinkingTime: number;
  thinkingTimeFormat: string;
  inferenceTime: number;
  emitTime: number;
  emitTimeFormat: string;
  totalTime: number;
  totalTimeFormat: string;
  tokensPerSecond: number;
  totalTokens: number;
}

interface TemporaryInferResult {
  thinkingTimeFormat?: string;
  emitTimeFormat?: string;
  totalTimeFormat?: string;
  tokensPerSecond: number;
  totalTokens: number;
}

interface InferParams {
  threads: number;
  tokens: number;
  topK: number;
  topP: number;
  temp: number;
  freqPenalty: number;
  presPenalty: number;
  tfs: number;
  stop: string;
}

interface Task {
  name: string;
  model: string;
  template: string;
  modelConf?: { ctx: number }
  inferParams?: {
    threads?: number;
    tokens?: number;
    topK?: number;
    topP?: number;
    temp?: number;
    freqPenalty?: number;
    presPenalty?: number;
    tfs?: number;
    stop?: string;
  }
}

interface LmTemplate {
  name: string;
  content: string;
  vars?: Array<string>;
}

interface LmPrompt {
  name: string;
  content: string;
}

type FormatMode = "Html" | "Text" | "Markdown" | "Json";

export {
  ModelStateContract,
  InferResultContract,
  InferParams,
  TemporaryInferResult,
  LmTemplate,
  LmPrompt,
  Task,
  FormatMode,
}