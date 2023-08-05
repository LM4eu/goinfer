# Goinfer

Inference server for local language models. Based on [Llama.cpp](https://github.com/ggerganov/llama.cpp)

- Switch between models at runtime
- Run inference queries
- Websockets and http support

## Install

### Binary

Get a binary file in the releases section (Linux only)

### From source

Clone the repository, cd into it and install [go-llama.cpp](https://github.com/go-skynet/go-llama.cpp):

```bash
git clone --recurse-submodules https://github.com/go-skynet/go-llama.cpp
cd go-llama.cpp
make libbinding.a
```

## Configure

Create a config `goinfer.config.json` file at the root:

```json
{
  "models_dir": "/home/me/path/to/models/dir",
  "origins": [
    "http://localhost:5173",
    "http://localhost:3000",
  ]
}
```

Parameters:

- `models_dir` *string* **required**: the absolute path to the models directory
- `origins` *[]string*: a list of authorized CORS urls

## Run

### Binary

```bash
./goinfer
```

### From source

```bash
go run main.go
```

### Options

- `-v`: run in verbose mode
- `-nows`: disable the websockets

## Api

### Models

#### State

Get the current models state:

- `/model/state` *GET*: the current state of the models on the server

Example response:

```json
{
  "isModelLoaded": false,
  "loadedModel": "",
  "models": [
    "orca-mini-3b.ggmlv3.q8_0",
    "tulu-7b.ggmlv3.q5_1",
    "orca-mini-7b.ggmlv3.q5_1",
    "open-llama-7B-open-instruct.ggmlv3.q5_1",
    "WizardLM-13B-1.0.ggmlv3.q4_0"
  ],
  "ctx": 1024
}
```

Response properties:

- `isModelLoaded` *boolean*: is a model currently loaded in the server memory
- `loadedModel` *string*: the name of the currently loaded model, empty string if no model is loaded
- `models`: *[]string*: a list of the available model names in the models directory
- `ctx`: *int*: the size of the context window, in number of tokens (default *1024*)

#### Load model

Load a model into the server memory:

- `/model/load` *POST*: payload:
  - `model` *string* **required**: the model name to load
  - `ctx` *int*: context window size, default *1024*
  - `embeddings` *boolean*: use embeddings, default *false*
  - `gpuLayers` *int*: number of layers to run on GPU, default *0*

Example payload:

```js
{
  "model": "orca-mini-3b.ggmlv3.q8_0"
}
// or
{
  "model": "WizardLM-13B-1.0.ggmlv3.q4_0",
  "ctx": 2048
}
```

The response will be a `204` status code when the model is loaded

### Inference

Once a model is loaded we can start making inference requests

#### Infer

Run an inference request

- `/infer` *POST*: payload:
  - `prompt` *string* **required**: the prompt text
  - `template` *string*: the template to use, default *{prompt}*
  - `threads` *int*: the number of threads to use, default *4*
  - `tokens` *int*: the maximum number of tokens to predict, default *512*
  - `topK` *int*: the top K sampling param (limit the next token selection to the K most probable tokens), default *40*
  - `topP` *int*: the top P sampling param (limit the next token selection to a subset of tokens with a cumulative probability above a threshold P), default *0.95*
  - `temp` *int*: the temperature param (randomness of the generated text), default *0.2*
  - `frequencyPenalty` *int*: the frequency penalty param , default *0*
  - `presencePenalty` *int*: the presence penalty param , default *0*
  - `tfs` *float64*: the tail free sampling param (to reduce the probabilities of less likely tokens to appear), default *1.0* (disabled)
  - `stop` *[]string*: the stop tokens param to stop inference if met, default *[]*
  
Example post payload:

```json
{
  "prompt": "List the planets in the solar system",
  "template": "### Instruction: {prompt}\n\n### Response:",
  "temp": 0,
  "tfs": 2
}
```

A `{prompt}` slot is available for custom templates: this is where the prompt text
will be placed

#### Websockets

If the websockets option is enabled (default) the tokens emitted by the language model
will be send through websockets as they come. To connect client side to the websockets
server:

```js
const ws = new WebSocket('ws://localhost:5142/ws');

  ws.onopen = () => {
    console.log('WebSocket connected');
  };
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    const msg = data.msg;
    // use the message here
    console.log(event.data)
  };
  ws.onerror = (event) => {
    console.log('WebSocket error', event);
  };
  ws.onclose = () => {
    console.log('WebSocket disconnected');
  };
```

#### Abort

Abort a running inference

- `/infer/abort` *GET*: will return a `204` status code if the inference was aborted, and a `202` in case of nothing to abort

## Examples

### Models

Show models state:

```bash
curl http://localhost:5143/model/state  | python -m json.tool
```

Output:
```json
{
    "isModelLoaded": false,
    "loadedModel": "",
    "models": [
        "orca-mini-3b.ggmlv3.q8_0",
        "tulu-7b.ggmlv3.q5_1",
        "orca-mini-7b.ggmlv3.q5_1",
        "open-llama-7B-open-instruct.ggmlv3.q5_1",
        "WizardLM-13B-1.0.ggmlv3.q4_0"
    ]
}
```

Load a model:

```bash
curl -X POST -H "Content-Type: application/json" -d \
'{"model": "orca-mini-3b.ggmlv3.q8_0"}' http://localhost:5143/model/load
```

The response has no content and a `204` status code when the model was loaded successfully

### Inference

Run an inference request:

```bash
curl -X POST -H "Content-Type: application/json" -d \
'{"prompt": "List the planets in the solar system", "template": \
"### Instruction: {prompt}\n### Response:"}' \
http://localhost:5143/infer  | python -m json.tool
```

Response:

```json
{
    "text": " \n Sure, here are the planets in our solar system in order from the sun:\n\n1. Mercury\n2. Venus\n3. Earth\n4. Mars\n5. Jupiter\n6. Saturn\n7. Uranus\n8. Neptune",
    "thinkingTime": 0.759748491,
    "thinkingTimeFormat": "759.748491ms",
    "emitTime": 8.728113106,
    "emitTimeFormat": "8.728113106s",
    "totalTime": 9.487861597,
    "totalTimeFormat": "9.487861597s",
    "tokensPerSecond": 6.42,
    "totalTokens": 56,
}
```