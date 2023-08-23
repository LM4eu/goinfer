import requests
import feedparser


# in this example we use the model:
# https://huggingface.co/s3nh/mamba-gpt-3b-v3-GGML/resolve/main/mamba-gpt-3b-v3.ggmlv3.q8_0.bin
# converted to gguf with Llama.cpp
MODEL = "mamba-gpt-3b-v3.gguf.q8_0"
KEY = "7aea109636aefb984b13f9b6927cd174425a1e05ab5f2e3935ddfeb183099465"
URL = "https://news.ycombinator.com/rss"
SYSTEM = (
    "Below is an instruction that describes a task. Write a response that "
    "appropriately completes the request.\n\n"
)
BASE_TEMPLATE = "### Instruction: {prompt}\n\n### Response:\n"
TEMPLATE = SYSTEM + BASE_TEMPLATE
base_prompt = (
    "these are Hacker News headlines: \n\n{text}\nReturn the titles and urls of "
    "the news only talking about language models, ggml, vector databases, Llama 2 "
    "and AI topics in general. Only return news that talk about these topics, "
    "and nothing else. Say there are no news if none of these topics are found."
)
shot_text = (
    "- Why Open Source? (url: https://getconvoy.io/blog/why-open-source/)\n"
    "- Associated Press clarifies standards around generative AI "
    "(url: https://www.niemanlab.org/2023/08/not-a-replacement-of-journalists"
    "-in-any-way-ap-clarifies-standards-around-generative-ai/)\n"
    "- Unix is both a technology and an idea (url: https://utcc.utoronto.ca/~cks/"
    "space/blog/unix/UnixTechnologyAndIdea)\n"
)
shot_response = (
    "Here are the news about AI topics:\n\n"
    "- Associated Press clarifies standards around generative AI "
    "(url: https://www.niemanlab.org/2023/08/not-a-replacement-of-journalists"
    "-in-any-way-ap-clarifies-standards-around-generative-ai/)"
)

# get and format data
feed = feedparser.parse(URL)
text = ""
for entry in feed.entries:
    text += f"- {entry.title} (url: {entry.link})\n"

# build the one shot prompt
shot_prompt = base_prompt.replace("{text}", shot_text)
shot = shot_prompt + "\n\n### Response:\n" + shot_response + "\n\n### Instruction:\n"
prompt = shot + base_prompt
PROMPT = prompt.replace("{text}", text)
end = TEMPLATE.replace("{prompt}", PROMPT)

print("News headlines:")
print("------------------------")
print(text)
print("------------------------")
print("Checking the news for interesting topics ...")

# load a language model
payload = {
    "model": MODEL,
    "ctx": 4096,
}
headers = {"Authorization": f"Bearer {KEY}"}
url = "http://localhost:5143/model/load"
response = requests.post(url, headers=headers, json=payload)
if response.status_code != 204:
    raise Exception(f"Error loading model ${response.content}")

# run the inference query
payload = {
    "prompt": PROMPT.replace("{text}", text),
    "template": TEMPLATE,
    "temperature": 0.5,
    "tfs_z": 2.0,
    "repeat_penalty": 1.2,
    "n_predict": 2048,
    "stop": ["### Instruction:"],
}
url = "http://localhost:5143/completion"
response = requests.post(url, headers=headers, json=payload)
print(response.text)
