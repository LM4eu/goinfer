#!/bin/bash

# https://github.com/emk/llama-server-local-model-config/blob/main/gpt-oss-20b/run.sh
# https://github.com/ggml-org/llama.cpp/discussions/15396
# https://docs.unsloth.ai/basics/gpt-oss-how-to-run-and-fine-tune#recommended-settings

# Only one GPU: Offloads all MoE layers to the CPU
# -ot ".ffn_.*_exps.=CPU"
#
# More VRAM: Offloads only up and down projection MoE layers
# -ot ".ffn_(up|down)_exps.=CPU"
#
# Even more VRAM: Offloads only up projection MoE layers
# -ot ".ffn_(up)_exps.=CPU"
#
# Customize: Offload gate, up and down MoE layers but only from the 6th layer onwards.
# -ot "\.(6|7|8|9|[0-9][0-9]|[0-9][0-9][0-9])\.ffn_(gate|up|down)_exps.=CPU"
#
# 	--n-cpu-moe 1

# current directory
cd ${0%/*}

../../llama.cpp/build/bin/llama-server   \
	--threads -1                         \
	--host 0.0.0.0 --port 8080           \
	--no-context-shift                   \
	--no-warmup                          \
	--no-mmap                            \
	-hf ggml-org/gpt-oss-120b-GGUF       \
	--alias openai/gpt-oss-120b          \
	--temp 1.0  --top-k 0.0              \
	--min-p 0.0 --top-p 1.0              \
	-c 16384                             \
	--batch-size 2048 --ubatch-size 2048 \
	-ngl 999                             \
	-ot "\.(6|7|8|9)\.ffn_up_exps.=CPU"  \
	--jinja                              \
	--reasoning-format auto              \
	--chat-template-kwargs '{"reasoning_effort": "high"}' \
	--grammar-file run-llama-gpt-roo.grammar              \

