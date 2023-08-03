<template>
  <sw-topbar :topbar="topBar" class="z-10 flex items-center w-full h-16" breakpoint="lg">
    <template #mobile-back>
      <i-ion-arrow-back-outline class="inline-flex ml-2 text-3xl" v-if="!isHome"></i-ion-arrow-back-outline>
    </template>
    <template #mobile-branding>
      <div class="inline-flex flex-row items-center h-full pt-1 ml-2 text-2xl truncate">
        <div class="text-lg"> <button class="mr-5 border-none btn" @click="toggleModelCollapse">
            <template v-if="lmState.isLoadingModel">
              Loading model ..
            </template>
            <template v-else-if="lmState.isModelLoaded">
              {{ lmState.model }} ctx: {{ lmState.ctx }}
            </template>
            <template v-else>
              Pick a model&nbsp;<i-carbon:network-4></i-carbon:network-4>
            </template>
          </button>
          <OverlayPanel ref="modelCollapse">
            <models-picker @close="toggleModelCollapse"></models-picker>
          </OverlayPanel>
        </div>
      </div>
    </template>
    <template #branding>
      <div class="flex flex-row items-center h-full ml-3 cursor-pointer">
        <div class="text-lg">
          <button class="mr-5 border-none btn" @click="toggleModelCollapse">
            <template v-if="lmState.isLoadingModel">
              Loading model ..
            </template>
            <template v-else-if="lmState.isModelLoaded">
              <div class="flex flex-row items-center space-x-2">
                <div><i-iconoir:network-alt class="text-xl"></i-iconoir:network-alt></div>
                <div>{{ lmState.model }} <span class="txt-light"> ctx: {{ lmState.ctx }}</span></div>
              </div>
            </template>
            <template v-else>
              <div class="flex flex-row items-center space-x-2">
                <div>Pick a model</div>
                <div><i-iconoir:network-alt class="text-xl"></i-iconoir:network-alt></div>
              </div>
            </template>
          </button>
          <OverlayPanel ref="modelCollapse">
            <models-picker @close="toggleModelCollapse"></models-picker>
          </OverlayPanel>
        </div>
      </div>
    </template>
    <template #menu>
      <div class="flex flex-row items-center justify-end w-full h-full space-x-3">
        <button v-show="lmState.isRunning == true || lmState.isStreaming == true"
          class="flex flex-row items-center justify-center w-48 mr-2 btn bord-light txt-light block-lighter"
          @click="abort()">
          <i-icomoon-free:stop class="mr-2"></i-icomoon-free:stop>
          <div>Stop</div>
        </button>
        <div class="pr-5 text-lg cursor-pointer txt-lighter dark:txt-light" @click="user.toggleDarkMode()">
          <i-fa-solid:moon v-if="!user.isDarkMode.value"></i-fa-solid:moon>
          <i-fa-solid:sun v-else></i-fa-solid:sun>
        </div>
      </div>
    </template>
    <template #mobile-menu>
      <div class="flex flex-col p-3 pb-5 space-y-3 lighter border-y-2 bord-primary">
        <div>
          <button class="border-none btn" @click="$router.push('/page'); topBar.closeMenu()">Page 1</button>
        </div>
        <div class="text-lg cursor-pointer" @click=" user.toggleDarkMode(); topBar.closeMenu()">
          <template v-if="!user.isDarkMode.value">
            <i-fa-solid:moon></i-fa-solid:moon>&nbsp;Dark mode
          </template>
          <template v-else>
            <i-fa-solid:sun></i-fa-solid:sun>&nbsp;Light mode
          </template>
        </div>
      </div>
    </template>
  </sw-topbar>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { SwTopbar, useTopbar } from "@snowind/header";
import { user, lmState } from "@/state";
import { useRouter } from 'vue-router';
import OverlayPanel from 'primevue/overlaypanel';
import ModelsPicker from '@/components/ModelsPicker.vue';
import { abort } from "@/services/api";

const router = useRouter()
const topBar = useTopbar(router);
const modelCollapse = ref();
const taskCollapse = ref();

const toggleModelCollapse = (event) => {
  modelCollapse.value.toggle(event);
}
const toggleTaskCollapse = (event) => {
  taskCollapse.value.toggle(event);
}

const isHome = computed<boolean>(() => router.currentRoute.value.path == "/");
</script>

<style lang="sass">
#mobile-menu
  @apply absolute left-0 z-40 flex flex-col w-full space-y-3 text-xl top-16 lighter
*
  -webkit-tap-highlight-color: rgba(0, 0, 0, 0)
</style>