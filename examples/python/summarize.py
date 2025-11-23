import trafilatura
import requests

# in this example we use the model:
# https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
MODEL = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
KEY = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de"
URL = "https://152334h.github.io/blog/non-determinism-in-gpt-4/"
TEMPLATE = "<s>[INST] {prompt} [/INST]"
PROMPT = "summarize this text to the main bullet points:"
# PROMPT = "extract the links from this text:"

downloaded = trafilatura.fetch_url(URL)

text = trafilatura.extract(downloaded, include_links=True, url=URL)

print("Extracted text from url:")
print("------------------------")
print(text)
print("------------------------")
print("Summarizing text ...")

# run the inference query
payload = {
    "model": MODEL,
    "ctx": 8192,
    "template": TEMPLATE,
    "prompt": f"{PROMPT}\n\n{text}",
}
url = "http://localhost:8080/completion"
headers = {"Authorization": f"Bearer {KEY}"}
response = requests.post(url, headers=headers, json=payload)
data = response.json()
print("Model response:")
print(data["text"])
print("Raw response:")
print(data)
