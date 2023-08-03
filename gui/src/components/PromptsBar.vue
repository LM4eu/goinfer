<template>
  <div class="flex flex-col space-y-1 overflow-y-auto">
    <div v-for="prompt in prompts" class="flex flex-row group">
      <div class="justify-start w-2/3 ml-2 truncate">
        <button class="btn" @click="loadPrompt(prompt)">
          {{ prompt }}
        </button>
      </div>
      <div class="flex flex-row justify-end w-1/3 mr-1 transition-opacity duration-200 opacity-0 group-hover:opacity-100">
        <confirm-delete @delete="db.delPrompt(prompt); loadPrompts()"></confirm-delete>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { prompts, db, loadPrompts } from '@/state';
import { loadPrompt } from "@/state";
import ConfirmDelete from '@/widgets/ConfirmDelete.vue';
</script>