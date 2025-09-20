import json
import sseclient
import requests

# in this example we use the model:
# https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
MODEL = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
KEY = "c0ffee15c00150c0ffee15600dbadc0de15d3ad101cafe61f760cafe7357c0d3"
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
