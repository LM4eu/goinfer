<template>
  <div id="infer-block" class="px-3 mb-12 overflow-y-auto form h-main">
    <div class="flex flex-col">
      <div class="pt-3">
        <Textarea v-model="template.content" class="w-full h-48" />
      </div>
      <div class="pt-2">
        <Textarea v-model="prompt" rows="1" class="w-full" autoResize />
      </div>
      <div class="flex flex-row items-center justify-end pt-3 space-x-2 h-1/3" v-if="lmState.isModelLoaded">
        <div class="flex flex-row items-center flex-grow txt-semilight">
          <button class="px-2 btn" v-show="template.content.length > 0" @click="toggleSaveTask($event)">
            <i-carbon:task-star class="text-2xl"></i-carbon:task-star>
          </button>
          <OverlayPanel ref="saveTaskCollapse">
            <save-task-dialog class="p-3 mt-3" @pick="toggleSaveTask($event)"></save-task-dialog>
          </OverlayPanel>
          <button class="px-2 btn" v-show="template.content.length > 0" @click="toggleSaveTemplate($event)">
            <i-bi:menu-up class="text-xl"></i-bi:menu-up>
          </button>
          <OverlayPanel ref="saveTemplateCollapse">
            <save-template-dialog class="p-3 mt-3" @pick="toggleSaveTemplate($event)"></save-template-dialog>
          </OverlayPanel>
          <button class="px-2 btn" v-show="prompt.length > 0" @click="toggleSavePrompt($event)">
            <i-tabler:prompt class="text-3xl"></i-tabler:prompt>
          </button>
          <OverlayPanel ref="savePromptCollapse">
            <save-prompt-dialog class="p-3 mt-3" @pick="toggleSavePrompt($event)"></save-prompt-dialog>
          </OverlayPanel>
        </div>
        <format-bar v-if="stream.length > 0" @select="selectFormatMode($event)"></format-bar>
        <button id="clearinfer-btn" class="btn txt-semilight"
          :disabled="(lmState.isRunning || stream.length == 0) ? true : false" @click="stream = ''; clearInferResults()">
          <i-grommet-icons:clear class="text-xl"></i-grommet-icons:clear>
        </button>
        <div>
          <button id="runinfer-btn" class="flex flex-row items-center w-48 btn bord-light txt-light block-lighter"
            @click="processInfer()" :disabled="prompt.length == 0 || lmState.isRunning == true">
            <i-iconoir:play class="mr-2 text-xl"></i-iconoir:play>
            <div>Run inference</div>
          </button>
        </div>
        <!-- button v-show="lmState.isRunning == true || lmState.isStreaming == true"
          class="flex flex-row items-center justify-center w-48 btn bord-light txt-light block-lighter" @click="abort()">
          <i-icomoon-free:stop class="mr-2"></i-icomoon-free:stop>
          <div>Stop</div>
        </button -->
      </div>
      <div v-if="lmState.isRunning == true && lmState.isStreaming == false">
        <LoadingSpinner class="pt-16 text-6xl txt-lighter" />
      </div>
      <div class="mt-5 mb-8 h-2/3">
        <div v-if="formatMode == 'Html'" class="mb-8 text-justify"
          v-html="stream.replaceAll('\n', '<br />').replaceAll('\t', '&nbsp;&nbsp;')">
        </div>
        <pre class="mx-3" v-else-if="formatMode == 'Text'">{{ stream }}</pre>
        <div class="mx-3 mt-8" v-else-if="formatMode == 'Markdown'">
          <render-md :hljs="hljs" :source="stream"></render-md>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { watchDebounced } from '@vueuse/core';
import OverlayPanel from 'primevue/overlaypanel';
import Textarea from 'primevue/textarea';
import { RenderMd } from '@docdundee/vue';
import { clearInferResults, lmState, stream, formatMode } from '@/state';
import { infer } from "@/services/api";
import LoadingSpinner from '@/widgets/LoadingSpinner.vue';
import SavePromptDialog from './SavePromptDialog.vue';
import SaveTemplateDialog from './SaveTemplateDialog.vue';
import SaveTaskDialog from './SaveTaskDialog.vue';
import FormatBar from './FormatBar.vue';
import { template, prompt, inferParams, inferResults, secondsCount, countPromptTokens, countTemplateTokens } from '@/state';
import { hljs } from "@/conf";

const savePromptCollapse = ref();
const saveTemplateCollapse = ref();
const saveTaskCollapse = ref();

async function processInfer() {
  clearInferResults();
  const id = setInterval(() => {
    secondsCount.value++;
    const tps = parseFloat((inferResults.totalTokens / secondsCount.value).toFixed(1));
    inferResults.tokensPerSecond = tps;
  }, 1000);
  const res = await infer(prompt.value, template.content, inferParams);
  clearInterval(id);
  inferResults.thinkingTimeFormat = res.thinkingTimeFormat;
  inferResults.emitTimeFormat = res.emitTimeFormat;
  inferResults.totalTimeFormat = res.totalTimeFormat;
}

function toggleSavePrompt(evt) {
  savePromptCollapse.value.toggle(evt);
}

function toggleSaveTemplate(evt) {
  saveTemplateCollapse.value.toggle(evt);
}

function toggleSaveTask(evt) {
  saveTaskCollapse.value.toggle(evt);
}

function selectFormatMode(evt) {
  console.log("F", evt)
}

onMounted(() => {
  countPromptTokens();
  countTemplateTokens();
  watchDebounced(template, countTemplateTokens, { debounce: 1000, maxWait: 5000 });
  watchDebounced(prompt, countPromptTokens, { debounce: 1000, maxWait: 5000 });
})
</script>

<style lang="sass">
#runinfer-btn:disabled
  @apply opacity-25
#clearinfer-btn:disabled
  @apply opacity-0
</style>