#!/usr/bin/env node
import { PromptTemplate } from "modprompt";

// in this example we use the model:
// https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v0.3-GGUF/resolve/main/tinyllama-1.1b-chat-v0.3.Q8_0.gguf
const model = "tinyllama-1.1b-chat-v0.3.Q8_0.gguf"
const apiKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de";
const template = "<|im_start|>system\nYou are a javascript coding assistant<|im_end|>\n<|im_start|>user\n{prompt}<|im_end|>\n<|im_start|>assistant ```json";

async function baseQuery(prompt) {
  // run the inference query
  const response = await fetch(`http://localhost:4444/completion`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      prompt: prompt,
      template: template,
      model: model,
      llama:{
        temperature: 0.8
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
  const prompt = "list the planets in the solar system and their distance from the sun";
  console.log("Prompt: ", prompt);
  const lmResponse = await baseQuery(prompt);
  console.log("Response:");
  console.log(lmResponse.text);
  let data = lmResponse.text;
  return data
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