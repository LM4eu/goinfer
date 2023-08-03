<template>
  <div class="flex flex-col">
    <div v-for="(elem, i) in elems" class="px-8 py-4 cursor-pointer bord-lighter"
      :class="i == (elems.length - 1) ? '' : 'border-b'" @click="$emit('pick', elem)">
      {{ elem }}
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps({
  elems: {
    type: Object as () => Array<string>,
    required: true
  }
});

defineEmits(["pick"])
</script>