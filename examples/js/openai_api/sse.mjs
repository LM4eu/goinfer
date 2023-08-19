#!/usr/bin/env node
import { createParser } from 'eventsource-parser'

const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
const template = "### Instruction: {prompt}\n\n### Response:";
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
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": template.replace("{prompt}", prompt)
      }
    ]
  };
  const completionParams = { ...params, prompt };
  const response = await fetch("http://localhost:5143/v1/chat/completions", {
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