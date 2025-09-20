import trafilatura
import requests


# in this example we use the model:
# https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
MODEL = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
KEY = "c0ffee15c00150c0ffee15600dbadc0de15d3ad101cafe61f760cafe7357c0d3"
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
    "model": {
        "name": MODEL,
        "ctx": 8192,
    },
    "prompt": f"{PROMPT}\n\n{text}",
    "template": TEMPLATE,
}
url = "http://localhost:5143/completion"
headers = {"Authorization": f"Bearer {KEY}"}
response = requests.post(url, headers=headers, json=payload)
data = response.json()
print("Model response:")
print(data["text"])
print("Raw response:")
print(data)
