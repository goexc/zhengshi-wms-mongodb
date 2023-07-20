<script setup lang="ts">
import {onMounted} from "vue";
import useSettingStore from "@/store/modules/setting.ts";

const settingStore = useSettingStore()

onMounted(()=>{
  //设置主题、颜色
  settingStore.initSetting()
})
</script>

<template>
 <router-view></router-view>
</template>

<style scoped lang="scss">

</style>
