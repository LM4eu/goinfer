import requests
import feedparser


# in this example we use the model:
# https://huggingface.co/TheBloke/Mistral-7B-Instruct-v0.1-GGUF/resolve/main/mistral-7b-instruct-v0.1.Q4_K_M.gguf
MODEL = "mistral-7b-instruct-v0.1.Q4_K_M.gguf"
KEY = "C0ffee15C00150C0ffee15900dBadC0de15Dead101Cafe91f790Cafe7e57C0de"
URL = "https://news.ycombinator.com/rss"
base_prompt = (
    "these are Hacker News headlines: \n\n{text}\nReturn the titles and urls of "
    "the news only talking about AI, specially language models, gguf, vector databases,"
    " Llama 2, Mistral and other AI related topics."
)
instruction = """Only return news that talk about these topics"
    " and nothing else. Say there are no news if none of these topics are found."""
shot_text = (
    "- Why Open Source? (url: https://getconvoy.io/blog/why-open-source/)\n"
    "- Associated Press clarifies standards around generative AI "
    "(url: https://www.niemanlab.org/2023/08/not-a-replacement-of-journalists"
    "-in-any-way-ap-clarifies-standards-around-generative-ai/)\n"
    "- Android devices with backdoored firmware found in US schools \n"
    "(url: https://www.securityweek.com/android-devices-with-backdoored-"
    "firmware-found-in-us-schools/\n)"
    "- Unix is both a technology and an idea (url: https://utcc.utoronto.ca/~cks/"
    "space/blog/unix/UnixTechnologyAndIdea)\n"
    "- How will AI lean next? (url: \n"
    "https://www.newyorker.com/science/annals-of-artificial-intelligence/"
    "how-will-ai-learn-next)\n"
    "- SlowLlama: Finetune llama2-70B and codellama on MacBook Air without \n"
    "quantization (url: https://github.com/okuvshynov/slowllama)"
)
shot_response = (
    "Here are the news about AI topics:\n\n"
    "- Associated Press clarifies standards around generative AI "
    "(url: https://www.niemanlab.org/2023/08/not-a-replacement-of-journalists"
    "-in-any-way-ap-clarifies-standards-around-generative-ai/)"
    "- How will AI lean next? (url: \n"
    "https://www.newyorker.com/science/annals-of-artificial-intelligence/"
    "how-will-ai-learn-next)\n"
    "- SlowLlama: Finetune llama2-70B and codellama on MacBook Air without \n"
    "quantization (url: https://github.com/okuvshynov/slowllama)"
)

# get and format data
feed = feedparser.parse(URL)
text = ""
for entry in feed.entries:
    text += f"- {entry.title} (url: {entry.link})\n"

PROMPT = f"""<s>[INST] {base_prompt.replace("{text}", shot_text)}s {instruction} [/INST]
{shot_response}
[INST] {base_prompt.replace("{text}", text)} [/INST]
"""

print("News headlines:")
print("------------------------")
print(text)
print("------------------------")
print("Checking the news for interesting topics ...")

# run the inference query
payload = {
    "prompt": PROMPT,
    "model": {"name": MODEL, "ctx": 4096},
    "temperature": 0.5,
    "top_p": 0.35,
    "repeat_penalty": 1.2,
}
headers = {"Authorization": f"Bearer {KEY}"}
url = "http://localhost:5143/completion"
response = requests.post(url, headers=headers, json=payload)
data = response.json()
print("Model response:")
print(data["text"])
print("Raw response:")
print(data)
