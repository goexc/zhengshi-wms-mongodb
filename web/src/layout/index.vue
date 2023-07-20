<script setup lang="ts">
import LayoutLogo from '@/layout/logo/index.vue'
import LayoutMenu from '@/layout/menu/index.vue'
import LayoutMain from '@/layout/main/index.vue'
import LayoutTabBar from '@/layout/tabbar/index.vue'
//获取用户相关的小仓库
import {useRoute} from "vue-router";
import useLayoutSettingStore from "@/store/modules/layout";
import {useAuthStore} from "@/store/modules/auth.ts";
import {computed} from "vue";

defineOptions({
  name:"Layout"
})

let authStore = useAuthStore()
const route = useRoute()
const layoutSettingStore = useLayoutSettingStore()
// const menus = computed(() => authStore.showMenuListGet);
const menus = computed(() => authStore.authMenuListGet);
</script>

<template>
  <div class="layout_container">
    <!-- 左侧菜单栏 -->
    <div class="layout_slider" :class="{ collapse: !!layoutSettingStore.collapse }">
      <LayoutLogo></LayoutLogo>
      <!-- 展示菜单 -->
      <!--滚动组件-->
      <el-scrollbar class="scrollbar">
        <!--菜单组件-->
        <el-menu
            mode="vertical"
            :default-active="route.path"
            :router="true"
            :collapse="layoutSettingStore.collapse"
            :collapse-transition="false"
        >
          <!--    动态生成菜单      -->
<!--          <LayoutMenu :menus="userStore.menuRoutes"></LayoutMenu>-->
          <LayoutMenu :menus="menus"></LayoutMenu>
        </el-menu>
      </el-scrollbar>
    </div>
    <!-- 顶部导航 -->
    <div class="layout_tabbar" :class="{ collapse: !!layoutSettingStore.collapse }">
      <LayoutTabBar></LayoutTabBar>
    </div>
    <!-- 内容展示区域 -->
    <div class="layout_main" :class="{ collapse: !!layoutSettingStore.collapse }">
      <LayoutMain></LayoutMain>
    </div>
  </div>
</template>

<style scoped lang="scss">
.layout_container {
  width: 100%;
  height: 100vh;
  white-space: nowrap; //文字不换行

  .layout_slider {
    width: $base-menu-width;
    height: 100vh;
    background: $base-menu-background;
    color: white;
    //transition: all 0.3s;
    border-right: 1px solid #e4e7ed;

    &.collapse {
      width: $base-menu-min-width;
    }

    .scrollbar {
      width: 100%;
      height: calc(100vh - $base-menu-logo-height);
      .el-menu{
        border-right: 0;
      }
    }
  }

  .layout_tabbar {
    position: fixed; //固定定位
    top: 0;
    left: $base-menu-width;
    width: calc(100% - $base-menu-width);
    height: $base-tabbar-height;

    &.collapse {
      width: calc(100vw - $base-menu-min-width );
      left: $base-menu-min-width;
    }
  }

  .layout_main {
    position: absolute;
    top: $base-tabbar-height;
    left: $base-menu-width;
    width: calc(100% - $base-menu-width);
    height: calc(100vh - $base-tabbar-height);
    padding: 20px;
    overflow: auto;

    &.collapse {
      width: calc(100vw - $base-menu-min-width );
      left: $base-menu-min-width;
    }
  }
}
</style>