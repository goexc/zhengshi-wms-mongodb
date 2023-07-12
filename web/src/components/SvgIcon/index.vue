<script setup lang="ts">
defineProps({
  //xlink:href属性值前缀
  prefix: {
    type: String,
    default:'#icon-'
  },
  //图标名字
  name: String,
  //接收父组件传递颜色
  color: {
    type:String,
    default:''
  },
  //接收父组件传递过来的图标宽度
  width: {
    type:String,
    default: '16px'
  },
  //接收父组件传递过来的图标高度
  height: {
    type:String,
    default: '16px'
  },
})
</script>

<template>
  <svg :fill="color" :width="width" :height="height">
    <use :xlink:href="prefix+name">      </use>
  </svg>
</template>

<style scoped>

</style>