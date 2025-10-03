#!/usr/bin/env node
import { PromptTemplate } from "modprompt";

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de";
const prompt = "What is the capital of Kenya?";

async function listModels() {
  const response = await fetch(`http://localhost:4444/models`, {
    method: 'GET',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
  });
  if (response.status != 200) {
    throw new Error("Can not list the available models", response)
  }
  const data = await response.json();
  const models = data.models;
  console.log(models);
  return models
}

async function infer(models) {
  const template = models[model].template;
  const tpl = new PromptTemplate(template.name);
  const finalPrompt = tpl.prompt(prompt);
  console.log(finalPrompt);
  // run the inference query
  const response = await fetch(`http://localhost:4444/completion`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: model,
      ctx: tpl.ctx,
      llama:{
        prompt: finalPrompt,
        temperature: 1.0,
        top_p: 0.2,
        stop: [tpl.stop],
      }
    })
  });
  if (response.ok) {
    const data = await response.json();
    return data
  } else {
    throw new Error(`Error ${response.status} ${response}`)
  }
}

async function main() {
  const models = await listModels();
  const response = await infer(models);
  console.log(response);
}

(async () => {
  try {
    await main();
  } catch (e) {
    throw e
  }
})();