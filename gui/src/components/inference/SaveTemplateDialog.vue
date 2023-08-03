<template>
  <div class="flex flex-col">
    <div class="p-float-label">
      <InputText inputId="tname" v-model="tname" class="" v-focus @keyup.enter="save" />
      <label for="tname">Name</label>
    </div>
    <div class="mt-3">
      <button class="btn success" :disabled="tname.length == 0" @click="save">Save template</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import InputText from 'primevue/inputtext';
import { db, loadTemplates } from '@/state';
import { template } from "@/state";

const emit = defineEmits(["pick"]);
const tname = ref("");

async function save() {
  await db.setTemplate(tname.value, template.content);
  await loadTemplates();
  emit("pick")
}
</script>