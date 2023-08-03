<template>
  <div>
    <overlay-list v-if="!selectCtx" :elems="models" @pick="pickModel($event)"></overlay-list>
    <!-- div class="flex flex-col" v-if="!selectCtx">
      <div v-for="(model, i) in models" class="px-8 py-4 cursor-pointer bord-lighter"
        :class="i == (models.length - 1) ? '' : 'border-b'" @click="pick(model)">
        {{ model }}
      </div>
    </div -->
    <div v-else class="flex flex-col p-3 space-y-3">
      <div class="text-xl">Context window size</div>
      <div class="flex flex-row items-center space-x-2 text-xs">
        <div>Presets:</div>
        <chip-text class="cursor-pointer" :class="ctx == 256 ? 'success' : 'lighter'" @click="preset(256)">256</chip-text>
        <chip-text class="cursor-pointer" :class="ctx == 512 ? 'success' : 'lighter'" @click="preset(512)">512</chip-text>
        <chip-text class="cursor-pointer" :class="ctx == 1024 ? 'success' : 'lighter'"
          @click="preset(1024)">1024</chip-text>
        <chip-text class="cursor-pointer" :class="ctx == 2048 ? 'success' : 'lighter'"
          @click="preset(2048)">2048</chip-text>
        <chip-text class="cursor-pointer" :class="ctx == 4096 ? 'success' : 'lighter'"
          @click="preset(4096)">4096</chip-text>
      </div>
      <div>
        <InputText :v-model.number="ctx" class="w-full" />
      </div>
      <div>
        <Slider v-model="ctx" class="w-full" :min="32" :max="4096" :step="64" />
      </div>
      <div>
        <button class="w-full mt-3 btn success" @click="post()">Load model</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import InputText from 'primevue/inputtext';
import Slider from 'primevue/slider';
import ChipText from "@/widgets/ChipText.vue";
import { selectModel } from '@/services/api';
import { models } from '@/state';
import OverlayList from '@/widgets/OverlayList.vue';

const emit = defineEmits(["close"]);

const selectCtx = ref(false);
const selectedModel = ref("");
const ctx = ref<number>(1024);

function preset(n: number) {
  ctx.value = n;
}

function pickModel(m: string) {
  selectedModel.value = m;
  selectCtx.value = true;
}

async function post() {
  emit("close")
  await selectModel(selectedModel.value, ctx.value);
  console.log("Model loaded");
  selectCtx.value = false;
}
</script>