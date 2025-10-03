#!/usr/bin/env node
import { createParser } from 'eventsource-parser'

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de";
const template = "<s>[INST] {prompt} [/INST]";
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

async function runInference() {
  const paramDefaults = {
    prompt: prompt,
    template: template,
    llama: {
      stream: true,
    }
  };

  const completionParams = { ...paramDefaults, prompt };

  const response = await fetch("http://localhost:4444/completion", {
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
  return await runInference();
}

(async () => {
  try {
    await main();
  } catch (e) {
    throw e
  }
})();