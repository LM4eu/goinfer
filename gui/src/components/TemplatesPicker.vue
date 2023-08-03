<template>
  <div>
    <overlay-list v-if="isReady" :elems="_templates.names" @pick="loadTemplate($event); emit('close')"></overlay-list>
  </div>
</template>

<script setup lang="ts">
import { onBeforeMount, reactive, ref } from 'vue';
import OverlayList from '@/widgets/OverlayList.vue';
import { loadTemplate } from './inference/state';
import { db } from '@/state';

const emit = defineEmits(["close"]);
const _templates = reactive({ names: new Array<string>() });
const isReady = ref(false);

async function getTemplates() {
  _templates.names = await db.listTemplatesNames();
  isReady.value = true
}

onBeforeMount(() => getTemplates())
</script>