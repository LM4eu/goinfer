#!/usr/bin/env node
import { PromptTemplate } from "modprompt";

// in this example we use the model:
// https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v0.3-GGUF/resolve/main/tinyllama-1.1b-chat-v0.3.Q8_0.gguf
// and the predefined task uses this one:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "tinyllama-1.1b-chat-v0.3.Q8_0.gguf"
const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const template = "<|im_start|>system\nYou are a javascript coding assistant<|im_end|>\n<|im_start|>user\n{prompt}<|im_end|>\n<|im_start|>assistant ```json";

async function baseQuery(prompt) {
  // load the model
  /*const response = await fetch(`http://localhost:5143/model/start`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      name: model,
      ctx: 4096,
    })
  });
  if (response.status != 204) {
    throw new Error("Can not load model", response)
  }*/
  // run the inference query
  const response = await fetch(`http://localhost:5143/completion`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      prompt: prompt,
      template: template,
      temperature: 0.8,
      tfs: 2,
      model: { name: model }
    })
  });
  if (response.ok) {
    const data = await response.json();
    return data
  } else {
    throw new Error(`Error ${response.status} ${response}`)
  }
}

async function fixJson(prompt) {
  const task = "code/json/fix";
  const response = await fetch(`http://localhost:5143/task/execute`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      task: task,
      prompt: prompt,
    })
  });
  if (response.ok) {
    console.log("Resp", response)
    const data = await response.json();
    return data.text;
  } else {
    throw new Error(`Error ${response.status} ${response}`)
  }
}


async function main() {
  const prompt = "list the planets in the solar system and their distance from the sun";
  console.log("Prompt: ", prompt);
  const lmResponse = await baseQuery(prompt);
  console.log("Response:");
  console.log(lmResponse.text);
  let data = lmResponse.text;
  if (data.includes(["```"])) {
    data = data.split("```")[0]
  }
  console.log("Validating json");
  try {
    JSON.parse(data);
    console.log("The json is valid");
    return data
  } catch (e) {
    console.log("Found invalid json with this error:");
    console.log("------------------------")
    console.log(e)
    console.log("------------------------")
    console.warn("=> Running a fix json task ...");
    const fdata = await fixJson(data);
    return fdata.replace("```json", "").replace("```", "")
  }
}

(async () => {
  try {
    const data = await main();
    console.log("Final response:");
    console.log(data);
    console.log("Json:")
    console.log(JSON.parse(data))
  } catch (e) {
    throw e
  }
})();