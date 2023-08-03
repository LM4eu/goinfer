#!/usr/bin/env node

const task = "code/json/fix";
const prompt = `{a: 1, b: [42,43,],}`;

async function main() {
  const response = await fetch(`http://localhost:5143/task/execute`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      task: task,
      prompt: prompt,
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