<template>
  <template v-if="!confirmMode">
    <button class="px-1 btn" @click="confirmMode = true">
      <i-mingcute:delete-2-line class="text-2xl txt-light"></i-mingcute:delete-2-line>
    </button>
  </template>
  <template v-else>
    <button class="px-1 btn" @click="confirmMode = false">
      <i-akar-icons:cross class="text-xl txt-light"></i-akar-icons:cross>
    </button>
    <button class="px-1 btn" @click="emit('delete'); confirmMode = false">
      <i-mingcute:check-fill class="text-2xl txt-danger"></i-mingcute:check-fill>
    </button>
  </template>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const emit = defineEmits(["delete"]);

const confirmMode = ref(false);
</script>