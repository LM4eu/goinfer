<template>
  <div>
    <button class="btn txt-semilight" @click="toggleOpen($event)">
      <template v-if="formatMode == 'Html'">
        <i-dashicons:html class="text-3xl"></i-dashicons:html>
      </template>
      <template v-else-if="formatMode == 'Text'">
        <i-cil:text class="text-2xl"></i-cil:text>
      </template>
      <template v-else-if="formatMode == 'Markdown'">
        <i-ant-design:file-markdown-outlined class="text-2xl"></i-ant-design:file-markdown-outlined>
      </template>
    </button>
    <OverlayPanel ref="opened">
      <div class="flex flex-col mt-2 space-y-2">
        <div class="flex flex-row items-center p-2 space-x-3 cursor-pointer"
          :class="formatMode == 'Html' ? '' : 'txt-semilight'" @click="selectMode('Html')">
          <div>
            <i-dashicons:html class="text-3xl"></i-dashicons:html>
          </div>
          <div>
            Html
          </div>
        </div>
        <div class="flex flex-row items-center p-2 space-x-3 cursor-pointer"
          :class="formatMode == 'Text' ? '' : 'txt-semilight'" @click="selectMode('Text')">
          <div>
            <i-cil:text class="text-2xl"></i-cil:text>
          </div>
          <div>
            Text
          </div>
        </div>
        <div class="flex flex-row items-center p-2 space-x-3 cursor-pointer"
          :class="formatMode == 'Markdown' ? '' : 'txt-semilight'" @click="selectMode('Markdown')">
          <div>
            <i-ant-design:file-markdown-outlined class="text-2xl"></i-ant-design:file-markdown-outlined>
          </div>
          <div>
            Markdown
          </div>
        </div>
        <!-- div class="p-2 cursor-pointer" :class="formatMode == 'Json' ? '' : 'txt-semilight'" @click="selectMode('Json')">
          <i-carbon:json class="text-2xl"></i-carbon:json>
        </div -->
      </div>
    </OverlayPanel>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import OverlayPanel from 'primevue/overlaypanel';
import { FormatMode } from "@/interfaces";
import { formatMode } from "@/state";

const emit = defineEmits(["select"]);

const opened = ref();

function selectMode(mode: FormatMode) {
  formatMode.value = mode;
  emit('select', mode);
  toggleOpen(false)
}

function toggleOpen(evt) {
  opened.value.toggle(evt);
}
</script>