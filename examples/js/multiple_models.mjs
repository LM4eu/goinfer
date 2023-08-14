#!/usr/bin/env node


async function baseQuery(prompt) {
  const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
  const template = "### Instruction: {prompt}\n\n### Response: (answer in json)";
  // load the model
  const response = await fetch(`http://localhost:5143/model/load`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: model
    })
  });
  if (response.status != 204) {
    throw new Error("Can not load model", response)
  }
  // run the inference query
  const response2 = await fetch(`http://localhost:5143/infer`, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      prompt: prompt,
      template: template,
      temperature: 0.2,
      tfs_z: 1.8,
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
    },
    body: JSON.stringify({
      task: task,
      prompt: prompt,
      instruction: "convert the distance into numbers"
    })
  });
  if (response.ok) {
    const data = await response.json();
    return data.text
  } else {
    throw new Error(`Error ${response.status} ${response}`)
  }
}


async function main() {
  const prompt = "list the planets names in the solar system and their distance from the sun in kilometers";
  console.log("Prompt: ", prompt);
  const lmResponse = await baseQuery(prompt);
  console.log("Response:");
  console.log(lmResponse.text);
  const data = lmResponse.text.replace("```json", "").replace("```", "");
  console.log("Validating json");
  try {
    const res = JSON.parse(data);
    console.log("The json is valid");
    return res
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