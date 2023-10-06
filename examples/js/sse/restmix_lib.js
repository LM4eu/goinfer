#!/usr/bin/env node
import { useApi } from "restmix";

// doc: https://synw.github.io/restmix/ts/postsse

// in this example we use the model:
// https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
const model = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const template = "<s>[INST] {prompt} [/INST]";
const prompt = "List the planets in the solar system";

const api = useApi({ "serverUrl": "http://localhost:5143" });
api.addHeader('Authorization', `Bearer ${apiKey}`);

async function loadModel() {
  const res = await api.post("/model/load", {
    name: model
  });
  if (!res.ok) {
    throw new Error("Can not load model", res)
  }
}

async function runInference() {
  process.stdout.setEncoding('utf8');
  const onChunk = (payload) => {
    switch (payload.msg_type) {
      case "token":
        process.stdout.write(payload.content)
        break;
      case "system":
        console.log("\nSystem msg:", payload);
      default:
        break;
    }
  };
  const abortController = new AbortController();
  const _payload = {
    prompt: prompt,
    template: template,
    stream: true,
    temperature: 0.6,
  };
  await api.postSse(
    "/completion",
    _payload,
    onChunk,
    abortController,
    false,
    true,
  );
}

async function main() {
  await loadModel();
  await runInference();
}

(async () => {
  await main();
})();