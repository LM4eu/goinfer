#!/usr/bin/env node

// run it with: ts-node-esm simple_task.ts

// fix json task model:
// wget https://huggingface.co/TheBloke/Nous-Hermes-Llama-2-7B-GGML/resolve/main/nous-hermes-llama-2-7b.ggmlv3.q4_K_M.bin
const task = "code/json/fix";
const _prompt = `{a: 1, b: [42,43,],}`;
const apiKey = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de";

async function main() {
  const response = await fetch(`http://localhost:5143/task/execute`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      task: task,
      prompt: _prompt,
    })
  });
  if (response.ok) {
    const data = await response.json();
    console.log(data)
  } else {
    console.log("Error", response.status)
  }
}

(async () => {
  try {
    await main();
  } catch (e) {
    throw e
  }
})();