#!/usr/bin/env node
import { PromptTemplate } from "modprompt";

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "c0ffee15c00150c0ffee15600dbadc0de15d3ad101cafe61f760cafe7357c0d3";
const prompt = "What is the capital of Kenya?";

async function readState() {
  const response = await fetch(`http://localhost:5143/model/state`, {
    method: 'GET',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
  });
  if (response.status != 200) {
    throw new Error("Can not load models state", response)
  }
  const data = await response.json();
  const models = data.models;
  console.log(models);
  return models
}

async function infer(models) {
  const template = models[model];
  const tpl = new PromptTemplate(template.name);
  const finalPrompt = tpl.prompt(prompt);
  console.log(finalPrompt);
  // run the inference query
  const response2 = await fetch(`http://localhost:5143/completion`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: {
        name: model,
        ctx: tpl.ctx,
      },
      prompt: finalPrompt,
      temperature: 1.0,
      top_p: 0.2,
      stop: [tpl.stop],
    })
  });
  if (response2.ok) {
    const data = await response2.json();
    return data
  } else {
    throw new Error(`Error ${response2.status} ${response2}`)
  }
}

async function main() {
  const models = await readState();
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