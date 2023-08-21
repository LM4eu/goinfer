import trafilatura
import requests

KEY = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
URL = "https://news.ycombinator.com/"
MODEL = "mamba-gpt-3b-v3.ggmlv3.q8_0"
TEMPLATE = "### Instruction: {prompt}\n\n### Response:"
PROMPT = "show the news about llm, ggml, vector databases " \
  "and llama 2 from this text:"

downloaded = trafilatura.fetch_url(URL)

"""text = trafilatura.extract(
  downloaded, 
  # include_links=True, 
  url=URL, 
  favor_precision=True,
  include_formatting=True,
)"""

text = trafilatura.html2txt(downloaded)

print("Extracted text from url:")
print("------------------------")
print(text)
print("------------------------")
print("Summarizing text ...")

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
   "prompt": f"{PROMPT}\n\n{text}",
   "template": TEMPLATE,
}
url = 'http://localhost:5143/completion'
response = requests.post(url, headers=headers, json=payload)
print(response.text)