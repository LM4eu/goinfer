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
  const data = JSON.parse(event.data);
  process.stdout.write(data.choices[0].delta.content);
}

const parser = createParser(onParse)

async function runInference() {
  const params = {
    prompt: prompt,
    template: template,
    model: model,
    stream: true,
    temperature: 0.5,
    messages: [
      {
        "role": "system", // can also be "developer"
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": template.replace("{prompt}", prompt)
      }
    ]
  };
  const completionParams = { ...params, prompt };
  const response = await fetch("http://localhost:8080/v1/chat/completions", {
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

(async () => {
  await runInference();
})();