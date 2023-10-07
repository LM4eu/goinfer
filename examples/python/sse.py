import json
import sseclient
import requests

# in this example we use the model:
# https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
MODEL = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
KEY = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
TEMPLATE = "<s>[INST] {prompt} [/INST]"
PROMPT = "list the planets in the solar system"

# run the inference query
payload = {
    "model": {
        "name": MODEL,
        "ctx": 4096,
    },
    "prompt": PROMPT,
    "template": TEMPLATE,
    "stream": True,
    "temperature": 0.6,
}
headers = {"Authorization": f"Bearer {KEY}", "Accept": "text/event-stream"}
url = "http://localhost:5143/completion"
response = requests.post(url, stream=True, headers=headers, json=payload)
client = sseclient.SSEClient(response)
for event in client.events():
    data = json.loads(event.data)
    if data["msg_type"] == "token":
        print(data["content"], end="", flush=True)
    elif data["msg_type"] == "system":
        if data["content"] == "result":
            print("\n\nRESULT:")
            print(data)
        else:
            print("SYSTEM:", data, "\n")
