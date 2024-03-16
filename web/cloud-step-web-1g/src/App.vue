<script setup lang="ts">
import Control from './components/Control.vue'
import Login from './components/Login.vue'
import request from './utils/request'
import {ref,onMounted} from 'vue'

const logined = ref(false)

function checkLogined(){
    request.request<any>(
        {
            url: '../check',
            method: 'GET',
            headers: {
                    "Token": window.sessionStorage.getItem('token')
                }
        }
    ).then((res)=>{
      if(res.data.code === 0){
        logined.value = true
      }else{
        logined.value = false
      }
    })
}

onMounted(()=>{
    checkLogined()
})

</script>

<template>
  <Login v-if="!logined" @check="checkLogined"/>
  <Control v-if="logined"/>
</template>

<style scoped>

</style>
