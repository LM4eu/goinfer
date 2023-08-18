#!/usr/bin/env node
import { ChatGPTAPI } from 'chatgpt'

const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
//const template = "### Instruction: {prompt}\n\n### Response:";
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
  //const finalPrompt = template.replace("{prompt}", prompt);
  const res = await api.sendMessage(prompt, {
    onProgress: (partialResponse) => {
      //console.log("Progress:", typeof partialResponse, partialResponse);
      process.stdout.write(partialResponse.delta)
    }
  })
  console.log("RESP", res)
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