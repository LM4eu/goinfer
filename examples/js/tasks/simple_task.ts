#!/usr/bin/env node

// run it with: ts-node-esm simple_task.ts

// fix json task model:
// wget https://huggingface.co/TheBloke/Nous-Hermes-Llama-2-7B-GGML/resolve/main/nous-hermes-llama-2-7b.ggmlv3.q4_K_M.bin
const task = "code/json/fix";
const _prompt = `{a: 1, b: [42,43,],}`;
const apiKey = "c0ffee15c00150c0ffee15600dbadc0de15d3ad101cafe61f760cafe7357c0d3";

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