#!/usr/bin/env node

const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
const template = "### Instruction: {prompt}\n\n### Response:";
const prompt = "List the planets in the solar system";

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
  };
  const completionParams = { ...paramDefaults, prompt };
  const response = await fetch("http://localhost:5143/infer", {
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
  let content = "";
  while (true) {
    const result = await reader.read();
    if (result.done) {
      break;
    }
    //console.log(result);
    const text = decoder.decode(result.value);
    const payload = JSON.parse(text);
    if (payload.msg_type == "token") {
      process.stdout.write(payload.content);
    } else {
      content = payload
    }
  }
  return content
}

async function main() {
  await loadModel();
  return await runInference();
}

(async () => {
  try {
    const data = await main();
    console.log("\n----------------------------");
    console.log(data);
    console.log("----------------------------");
  } catch (e) {
    throw e
  }
})();