#!/usr/bin/env node
import { useGoinfer, ModelState } from "@goinfer/api";
import { PromptTemplate } from "modprompt";

// use ts-node-esm goinfer_lib.ts to run this

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "c0ffee15c00150c0ffee15600dbadc0de15d3ad101cafe61f760cafe7357c0d3";
const prompt = "List the planets in the solar system";

const api = useGoinfer({
  serverUrl: "http://localhost:5143",
  apiKey: apiKey,
  onToken: (token) => {
    process.stdout.write(token);
  },
});

async function main() {
  const modelsState: ModelState = await api.modelsState();
  const template = new PromptTemplate(modelsState.models[model].name);
  const result = await api.infer(prompt, template.render(), {
    temperature: 0.6
  });
  console.log(result);
}

(async () => {
  await main();
})();

