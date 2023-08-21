import json
import sseclient
import requests

# wget https://huggingface.co/s3nh/mamba-gpt-3b-v3-GGML/resolve/main/mamba-gpt-3b-v3.ggmlv3.q8_0.bin
MODEL = "mamba-gpt-3b-v3.ggmlv3.q8_0"
KEY = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
TEMPLATE = "### Instruction: {prompt}\n\n### Response:"
PROMPT = "list the planets in the solar system"

# load a language model
payload = {
   "model": MODEL,
   "ctx": 4096,
}
headers = {'Authorization': f'Bearer {KEY}'}
url = 'http://localhost:5143/model/load'
response = requests.post(url, headers=headers, json=payload)
if (response.status_code != 204):
  raise Exception(f"Error loading model ${response.content}")

# run the inference query
payload = {
   "prompt": PROMPT,
   "template": TEMPLATE,
   "stream": True,
   "temperature": 0.6,
}
headers['Accept'] = 'text/event-stream'
url = 'http://localhost:5143/completion'
response = requests.post(url, stream=True, headers=headers, json=payload)
client = sseclient.SSEClient(response)
for event in client.events():
    data = json.loads(event.data)
    if data["msg_type"] == "token":
      print(data["content"], end='', flush=True)
    elif data["msg_type"] == "system":
      if data["content"] == "result":
        print("\n\nRESULT:")
        print(data)
      else:
        print("SYSTEM:", data, "\n")
    
