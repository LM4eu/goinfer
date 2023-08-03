<template>
  <div class="overflow-y-auto">
    <Tree v-model:selectionKeys="selectedKey" :value="tasks" class="w-full" @nodeSelect="onNodeSelect"
      @nodeUnselect="onNodeUnselect">
      <template #default="slotProps">
        <template v-if="slotProps.node.label.endsWith('.yml')">
          <button class="p-0 btn" @click="onNodeSelect(slotProps.node)">{{ slotProps.node.label }}</button>
        </template>
        <template v-else>
          {{ slotProps.node.label }}
        </template>
      </template>
    </Tree>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import Tree from 'primevue/tree';
import { tasks } from "@/state";
import { loadTask } from "@/services/api";
import { loadTask as loadTaskState } from "@/state";

const selectedKey = ref();

const onNodeSelect = async (node) => {
  console.log("Select", node.path)
  const task = await loadTask(node.path);
  console.log("TASK", JSON.stringify(task, null, "  "));
  loadTaskState(task);
};

const onNodeUnselect = (node) => {
  console.log("Unselect", node)
};
</script>

<style lang="sass">
.p-tree
  border: 0 !important
</style>