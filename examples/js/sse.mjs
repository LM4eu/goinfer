#!/usr/bin/env node
import { EventSourcePolyfill } from 'event-source-polyfill';

const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";

async function baseQuery(prompt) {
  const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
  const template = "### Instruction: {prompt}\n\n### Response:";
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
      temp: 0.3,
    })
  });
  if (response2.ok) {
    const data = await response2.json();
    return data
  } else {
    throw new Error(`Error ${response2.status} ${response2}`)
  }
}

async function main() {
  const url = "http://localhost:5143/infer/sse";
  const eventSource = new EventSourcePolyfill(url, {
    headers: {
      'Authorization': `Bearer ${apiKey}`
    },
    withCredentials: true,
  });

  eventSource.onmessage = function (event) {
    console.log('Received data:', event.data);
  };

  eventSource.onerror = function (error) {
    console.error('EventSource failed:', error);
    eventSource.close();
  };

  const prompt = "list the planets names in the solar system and their distance from the sun in kilometers";
  console.log("Prompt: ", prompt);
  const lmResponse = await baseQuery(prompt);
  console.log("Response:");
  console.log(lmResponse.text);
}

(async () => {
  try {
    const data = await main();
    console.log("Final response:");
    console.log(data);
  } catch (e) {
    throw e
  }
})();