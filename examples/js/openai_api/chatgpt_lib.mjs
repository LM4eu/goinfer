#!/usr/bin/env node
import { ChatGPTAPI } from 'chatgpt'

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de";
//const template = "### Instruction: {prompt}\n\n### Response:";
const template = "<s>[INST] {prompt} [/INST]";
const prompt = "List the planets in the solar system";

const api = new ChatGPTAPI({
  apiKey: apiKey,
  apiBaseUrl: "http://localhost:5143/v1",
  completionParams: {
    model: model,
    stream: true,
  },
  debug: true,
});

async function main() {
  const finalPrompt = template.replace("{prompt}", prompt);
  const res = await api.sendMessage(finalPrompt, {
    onProgress: (partialResponse) => {
      //console.log("Progress:", typeof partialResponse, partialResponse);
      process.stdout.write(partialResponse.delta)
    }
  })
  console.log("Response:", res)
  return res
}

(async () => {
  try {
    const data = await main();
    console.log("Final response:");
    console.log(data);
  } catch (e) {
    throw e
  }
})();