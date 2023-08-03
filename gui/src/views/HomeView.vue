<template>
  <div class="fixed left-0 flex flex-row w-full h-full top-16 ">
    <div class="flex-shrink-0 w-72">
      <side-bars></side-bars>
    </div>
    <div class="flex-shrink w-full max-w-6xl overflow-y-auto border-x bord-lighter">
      <prompt-template></prompt-template>
    </div>
    <div class="flex-grow flex-shrink-0 p-3 pt-8 min-w-72">
      <infer-params></infer-params>
      <infer-stats class=""></infer-stats>
      <infer-result class="mt-3"></infer-result>
    </div>
  </div>
</template>

<script setup lang="ts">
import PromptTemplate from '@/components/inference/PromptTemplate.vue';
import InferParams from '@/components/inference/InferParams.vue';
import InferResult from '@/components/inference/InferResult.vue';
import InferStats from '@/components/inference/InferStats.vue';
import SideBars from '@/components/SideBars.vue';
</script>