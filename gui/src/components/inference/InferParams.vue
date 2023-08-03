<template>
  <div id="params" class="flex flex-col w-full 3xl:max-w-[28rem]">
    <div id="pform" class="grid w-full grid-cols-2 3xl:grid-cols-3 gap-x-3 gap-y-8">
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.temp" inputId="temp" :min="0" :max="2" :step="0.1" showButtons />
          <label for="temp">Temp</label>
        </span>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.tfs" inputId="tfs" :min="0" :max="2" :step="0.1" showButtons />
          <label for="tfs">Tfs</label>
        </span>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.topK" inputId="topK" :min="0" :max="100" showButtons />
          <label for="topK">TopK</label>
        </span>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.topP" inputId="topP" :min="0" :max="1" :step="0.05" showButtons />
          <label for="topP">TopP</label>
        </span>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.freqPenalty" inputId="freqPenalty" :min="0" :max="2" :step="0.1"
            showButtons />
          <label for="freqPenalty">Freq Penalty</label>
        </span>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.presPenalty" inputId="presPenalty" :min="0" :max="2" :step="0.1"
            showButtons />
          <label for="presPenalty">Pres Penalty</label>
        </span>
      </div>
    </div>
    <div class="flex flex-row mt-8 mr-8 space-x-3">
      <div class="p-float-label">
        <InputText inputId="stop" v-model="inferParams.stop" class="w-full max-w-[8rem] 3xl:max-w-[16rem]" />
        <label for="stop">Stop words</label>
      </div>
      <div>
        <span class="p-float-label">
          <InputNumber class="w-8" v-model="inferParams.threads" inputId="threads" showButtons />
          <label for="threads">threads</label>
        </span>
      </div>
    </div>
    <div class="mt-5">
      <div class="p-float-label">
        <InputText inputId="tokens" class="hidden w-full" />
        <label for="tokens">Tokens</label>
      </div>
      <div class="mt-5 mr-8">
        <Slider v-model="inferParams.tokens" class="w-full" :min="64" :max="lmState.ctx" />
      </div>
      <div class="flex flex-row">
        <div class="p-3 txt-semilight">64</div>
        <div class="flex justify-center flex-grow p-3">
          {{ inferParams.tokens }}
        </div>
        <div class="p-3 txt-semilight">{{ lmState.ctx }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import InputNumber from 'primevue/inputnumber';
import InputText from 'primevue/inputtext';
import Slider from 'primevue/slider';
import { lmState } from '@/state';
import { inferParams } from '@/state';
</script>

<style lang="sass">
#params
  & > #pform > div
    @apply min-w-[8rem]  
  .p-inputtext
    &:not(.w-full)
      width: 4rem !important
  .p-inputnumber-button
    background-color: #e2e8f0 !important
    color: black !important
    border: #e2e8f0 !important
  .p-slider-range
    @apply light
  .p-slider-handle
    @apply border bord-primary
</style>