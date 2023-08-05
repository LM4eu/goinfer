import trafilatura
import requests

URL = "https://www.emencia.com/news/2023/04/26/logique-contractuelle-des-structures-de-donnees-dans-django-ninja/"
MODEL = "nous-hermes-llama-2-7b.ggmlv3.q4_K_M"
TEMPLATE = "### Instruction: {prompt}\n\n### Response:"
PROMPT = "résume ce texte:"

downloaded = trafilatura.fetch_url(URL)

text = trafilatura.extract(downloaded, include_links=True, url=URL)

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
url = 'http://localhost:5143/model/load'
response = requests.post(url, json=payload)
if (response.status_code != 204):
  raise Exception(f"Error loading model ${response.content}")

# run the inference query
payload = {
   "prompt": f"{PROMPT}\n\n{text}",
   "template": TEMPLATE,
}
url = 'http://localhost:5143/infer'
response = requests.post(url, json=payload)
print(response.text)