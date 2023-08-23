#!/usr/bin/env node

// in this example we use the model:
// https://huggingface.co/s3nh/mamba-gpt-3b-v3-GGML/resolve/main/mamba-gpt-3b-v3.ggmlv3.q8_0.bin
// converted to gguf with Llama.cpp
const model = "mamba-gpt-3b-v3.gguf.q8_0"
const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const template = "### Instruction: {prompt}\n\n### Response: (answer in json)\n\n```json";

async function baseQuery(prompt) {
  // load the model
  const response = await fetch(`http://localhost:5143/model/load`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model: model,
      ctx: 4096,
    })
  });
  if (response.status != 204) {
    throw new Error("Can not load model", response)
  }
  // run the inference query
  const response2 = await fetch(`http://localhost:5143/completion`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      prompt: prompt,
      template: template,
      temperature: 0.5,
      tfs_z: 1.4,
      stop: ["```"]
    })
  });
  if (response2.ok) {
    const data = await response2.json();
    return data
  } else {
    throw new Error(`Error ${response2.status} ${response2}`)
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
      instruction: "convert the distance into numbers"
    })
  });
  if (response.ok) {
    console.log("Resp", response)
    const data = await response.json();
    return data.text
  } else {
    throw new Error(`Error ${response.status} ${response}`)
  }
}


async function main() {
  const prompt = "list the planets names in the solar system and their distance from the sun in millions of kilometers";
  console.log("Prompt: ", prompt);
  const lmResponse = await baseQuery(prompt);
  console.log("Response:");
  console.log(lmResponse.text);
  const data = lmResponse.text;
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