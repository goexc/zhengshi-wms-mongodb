<script setup lang="ts">
import useLayoutSettingStore from "@/store/modules/layout";
import {nextTick, ref, watch} from "vue";

defineOptions({
  name:"LayoutMain"
})
let layoutSettingStore = useLayoutSettingStore()

//控制当前组件是否销毁、重建
const flag = ref<boolean>(true)

//监听刷新属性是否变化
watch(()=>layoutSettingStore.refresh, ()=>{
  flag.value = false
  //组件销毁和重建
  nextTick(()=>{
    flag.value = true
  })
})

</script>

<template>
  <div class="main">
    <!-- 路由组件出口位置 -->
    <router-view v-slot="{ Component }">
      <transition name="fade">
        <!--   渲染layout一级路由组件的子路由   -->
        <component :is="Component" v-if="flag"/>
      </transition>
    </router-view>
  </div>

</template>

<style scoped lang="scss">
.main{
//  margin-bottom: 60px;
//  min-width: 1000px;
}
.fade-enter-from{
  opacity: 0;
  transform: scale(0);
}
.fade-enter-active{
transition: all 0.3s;
}
.fade-enter-to{
  opacity: 1;
  transform: scale(1);
}
</style>