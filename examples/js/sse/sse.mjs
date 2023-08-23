#!/usr/bin/env node
import { createParser } from 'eventsource-parser'

// in this example we use the model:
// https://huggingface.co/s3nh/mamba-gpt-3b-v3-GGML/resolve/main/mamba-gpt-3b-v3.ggmlv3.q8_0.bin
// converted to gguf with Llama.cpp
const model = "mamba-gpt-3b-v3.gguf.q8_0"
const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const template = "### Instruction: {prompt}\n\n### Response:";
const prompt = "List the planets in the solar system";

function onParse(event) {
  if (event.data == "[DONE]") {
    return
  }
  const msg = JSON.parse(event.data);
  switch (msg.msg_type) {
    case "system":
      if (msg.content == "start_emitting") {
        console.log("Thinking time:", msg.data.thinking_time_format)
      } else if (msg.content == "result") {
        console.log(msg.data)
      }
      break;
    case "error":
      throw new Error("Error:", msg.content)
    default:
      process.stdout.write(msg.content);
  }
}

const parser = createParser(onParse)

async function loadModel() {
  // load the model
  const response = await fetch(`http://localhost:5143/model/load`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: model
    })
  });
  if (response.status != 204) {
    throw new Error("Can not load model", response)
  }
}

async function runInference() {
  const paramDefaults = {
    prompt: prompt,
    template: template,
    stream: true,
  };
  const completionParams = { ...paramDefaults, prompt };
  const response = await fetch("http://localhost:5143/completion", {
    method: 'POST',
    body: JSON.stringify(completionParams),
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream',
      'Authorization': `Bearer ${apiKey}`,
    },
  });
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  while (true) {
    const result = await reader.read();
    if (result.done) {
      break;
    }
    const chunk = decoder.decode(result.value);
    parser.feed(chunk)
  }
}

async function main() {
  await loadModel();
  return await runInference();
}

(async () => {
  try {
    await main();
  } catch (e) {
    throw e
  }
})();