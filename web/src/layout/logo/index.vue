<script setup lang="ts">
//引入标题与logo配置文件
import setting from "@/setting";
import useLayoutSettingStore from "@/store/modules/layout";

defineOptions({
  name:"LayoutLogo"
})

const layoutSettingStore = useLayoutSettingStore()

</script>

<template>
  <div class="logo" v-if="setting.logoHidden">
    <img :src="setting.logo" alt="">
    <el-text v-if="!layoutSettingStore.collapse">{{ setting.title }}</el-text>
  </div>
</template>

<style scoped lang="scss">
.logo{
  width: 100%;
  height: $base-menu-logo-height;
  display: flex;//logo与文字 浮动排列
  align-items: center; //水平一致
  padding: 10px;

  img{
    width: 40px;
    height: 40px;
  }

  p{
    //color: white;
    font-size: $base-logo-title-fontsize;
    margin-left: 10px;
  }
}
</style>