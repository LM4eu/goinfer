<template>
  <div class="flex flex-col h-main">
    <div class="flex flex-row w-full txt-light">
      <button class="w-1/3 p-2 border border-t-0 bord-lighter" :class="activeBar == 'prompts' ? 'border-b-0' : ''"
        @click="openTab('prompts')">Prompts</button>
      <button class="w-1/3 p-2 border border-t-0 bord-lighter" :class="activeBar == 'templates' ? 'border-b-0' : ''"
        @click="openTab('templates')">Templates</button>
      <button class="w-1/3 p-2 border border-t-0 border-x-0 bord-lighter"
        :class="activeBar == 'tasks' ? 'border-b-0' : ''" @click="openTab('tasks')">Tasks</button>
    </div>
    <div class="h-full overflow-y-auto">
      <prompts-bar class="mt-3" v-if="activeBar == 'prompts'"></prompts-bar>
      <templates-bar class="mt-3" v-else-if="activeBar == 'templates'"></templates-bar>
      <tasks-bar v-else></tasks-bar>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import PromptsBar from './PromptsBar.vue';
import TemplatesBar from "@/components/TemplatesBar.vue";
import TasksBar from './TasksBar.vue';

type TabType = "prompts" | "templates" | "tasks";

const activeBar = ref<TabType>("prompts");

function openTab(t: TabType) {
  activeBar.value = t
}
</script>