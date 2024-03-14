<script setup lang="ts">
import HelloWorld from './components/HelloWorld.vue'


const open = function () {
  layui.use(function () {
    var layer = layui.layer;
    layer.open({
      type: 1, // page 层类型
      area: ['500px', '300px'],
      title: 'Hello layer',
      shade: 0.6, // 遮罩透明度
      shadeClose: true, // 点击遮罩区域，关闭弹层
      maxmin: true, // 允许全屏最小化
      anim: 0, // 0-6 的动画形式，-1 不开启
      content: '<div style="padding: 32px;">一个普通的页面层，传入了自定义的 HTML</div>'
    });
  }
  )
}
</script>

<template>
  <button type="button" class="layui-btn layui-bg-blue" @click="open()">蓝色按钮</button>
  <HelloWorld msg="Vite + Vue" />
</template>

<style scoped>
.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
  transition: filter 300ms;
}

.logo:hover {
  filter: drop-shadow(0 0 2em #646cffaa);
}

.logo.vue:hover {
  filter: drop-shadow(0 0 2em #42b883aa);
}
</style>
