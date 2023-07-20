<script setup lang="ts">
//获取父组件传递过来的全部路由数组
defineProps(['menus'])

//声明组件名称，以便递归调用
defineOptions({
  name: "LayoutMenu"
})

</script>

<template>
  <template v-for="(menu) in menus" :key="menu.path">
    <template v-if="!menu.meta.hidden">
      <!--  没有子路由  -->
      <el-menu-item v-if="!menu.children" :index="menu.path">
        <el-icon>
          <component :is="menu.meta.icon"></component>
        </el-icon>
        <template #title>
          <span>{{ menu.name }}</span>
        </template>
      </el-menu-item>

      <!--  只有一个子路由  -->
      <el-menu-item v-if="menu.children&&menu.children.length===1" :index="menu.children[0].path">
        <el-icon>
          <component :is="menu.children[0].meta.icon"></component>
        </el-icon>
        <template #title>
          <span>{{ menu.children[0].name }}</span>
        </template>
      </el-menu-item>
      <!--  有多个子路由  -->
      <el-sub-menu v-if="menu.children&&menu.children.length>1" :index="menu.path">
        <template #title>
          <el-icon>
            <component :is="menu.meta.icon"></component>
          </el-icon>
          <span>{{ menu.name }}</span>
        </template>
        <!--   递归调用   -->
        <LayoutMenu :menus="menu.children"></LayoutMenu>
      </el-sub-menu>
    </template>
  </template>
</template>

<style scoped lang="scss">
.el-menu-item.is-active{
  color: #009688 !important;
  background-color: #e6f5f3 !important;

  &::before {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    width: 4px;
    content: "";
    background-color: #009688;
  }

}

</style>