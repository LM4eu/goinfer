<template>
  <div class="flex flex-col">
    <div v-for="t in templates" class="flex flex-row items-center space-y-1 group">
      <div class="justify-start w-2/3 ml-2 truncate overflow-ellipsis">
        <button class="btn" @click="loadTemplate(t)">
          {{ t }}
        </button>
      </div>
      <div class="flex flex-row justify-end w-1/3 mr-1 transition-opacity duration-200 opacity-0 group-hover:opacity-100">
        <confirm-delete @delete="db.delTemplate(t); loadTemplates()"></confirm-delete>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeMount, ref } from 'vue';
import ConfirmDelete from '@/widgets/ConfirmDelete.vue';
import { loadTemplate } from '@/state';
import { db, loadTemplates, templates } from '@/state';

onBeforeMount(() => loadTemplates())
</script>