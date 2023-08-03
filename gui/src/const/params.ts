import { InferParams } from "../interfaces";

const defaultInferenceParams: InferParams = {
  threads: 4,
  tokens: 512,
  topK: 40,
  topP: 0.95,
  temp: 0.2,
  freqPenalty: 0.0,
  presPenalty: 0.0,
  tfs: 1.0,
  stop: "",
};

export { defaultInferenceParams }