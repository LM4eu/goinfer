#!/usr/bin/env node
const { useApi } = require("restmix");

// doc: https://synw.github.io/restmix/ts/postsse

// wget https://huggingface.co/s3nh/mamba-gpt-3b-v3-GGML/resolve/main/mamba-gpt-3b-v3.ggmlv3.q8_0.bin
const model = "mamba-gpt-3b-v3.ggmlv3.q8_0";
const apiKey = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465";
const template = "### Instruction: {prompt}\n\n### Response:";
const prompt = "List the planets in the solar system";

const api = useApi({ "serverUrl": "http://localhost:5143" });
api.addHeader('Authorization', `Bearer ${apiKey}`);

async function loadModel() {
  const res = await api.post("/model/load", {
    model: model
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