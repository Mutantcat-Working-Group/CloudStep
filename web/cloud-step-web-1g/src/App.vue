<script setup lang="ts">
import Control from './components/Control.vue'
import Login from './components/Login.vue'
import {ref} from 'vue'

const logined = ref(false)

</script>

<template>
  <Login v-if="!logined"/>
  <Control v-if="logined"/>
</template>

<style scoped>

</style>
