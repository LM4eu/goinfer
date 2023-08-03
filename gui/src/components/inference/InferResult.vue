<template>
  <div id="infer-result">
    <div v-if="inferResults" class="flex flex-col space-y-2">
      <div class="flex flex-row items-center mt-2 mb-5">
        <div class="w-1/2 text-2xl" v-if="inferResults.totalTokens > 0">{{ inferResults.totalTokens }} <span
            class="text-base">tokens</span></div>
        <div class="w-1/2 text-2xl" v-if="inferResults.tokensPerSecond > 0">{{ inferResults.tokensPerSecond }} <span
            class="text-base">t/s</span></div>
      </div>
      <div v-if="inferResults.thinkingTimeFormat?.length ?? 0 > 0">
        <span class="txt-light">Thinking time</span>: {{ inferResults.thinkingTimeFormat }}
      </div>
      <div v-if="inferResults.emitTimeFormat?.length ?? 0 > 0">
        <span class="txt-light">Emit time</span>:
        {{ inferResults.emitTimeFormat }}
      </div>
      <div v-if="inferResults.totalTimeFormat?.length ?? 0 > 0">
        <span class="txt-light">Total time</span>:
        {{ inferResults.totalTimeFormat }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { inferResults } from '@/state';
</script>