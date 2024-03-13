<script setup lang="ts">
import useLayoutSettingStore from "@/store/modules/layout";
import {useRoute, useRouter} from "vue-router";
import {useUserStore} from "@/store/modules/user.ts";
import {ref} from "vue";
import useSettingStore from "@/store/modules/setting.ts";
import {reqLogout} from "@/api/user";
import {ElMessage} from "element-plus";

defineOptions({
  name: "LayoutTabBar"
})

let userStore = useUserStore()
let layoutSettingStore = useLayoutSettingStore()

const route = useRoute()
const router = useRouter()

//切换折叠、展开状态
const change = () => {
  layoutSettingStore.collapse = !layoutSettingStore.collapse
}

//刷新页面
const refresh = () => {
  layoutSettingStore.refresh = !layoutSettingStore.refresh
}

//切换全屏
const fullscreen = () => {
  //DOM对象的一个属性，用来判断是不是全屏模式。全屏：true，不是全屏：false
  let full = document.fullscreenElement
  if (!full) {
    //requestFullscreen:实现全屏模式
    document.documentElement.requestFullscreen()
  } else {
    document.exitFullscreen()
  }

}

//退出登录
const logout = async () => {
  //1.向服务器发出退出登录的请求
  let res = await reqLogout()
  if(res.code !== 200){
    ElMessage.error(res.msg)
    return
  }
  //2.清空仓库中的相关数据[token|name|avatar]
  await userStore.setToken('')
  //3.跳转到登录页面
  await router.push({path: '/login', query: {redirect: route.path}})
}

//预定义主题颜色
const settingStore = useSettingStore()

// const color = ref<string>('rgba(255, 69, 0, 0.68)')
const color = ref<string>(settingStore.color)
const predefineColors = ref<string[]>([
  '#ff4500',
  '#ff8c00',
  '#ffd700',
  '#90ee90',
  '#00ced1',
  '#1e90ff',
  '#c71585',
  'rgba(255, 69, 0, 0.68)',
  'rgb(255, 120, 0)',
  'hsv(51, 100, 98)',
  'hsva(120, 40, 94, 0.5)',
  'hsl(181, 100%, 37%)',
  'hsla(209, 100%, 56%, 0.73)',
  '#c7158577',
])


//夜间模式
const night = ref<boolean>(settingStore.theme === 'dark')

//切换主题
const changeTheme = () => {
  settingStore.changeTheme(night.value)
}

//更改主题颜色
const changeColor = () => {
  settingStore.changeColor(color.value)
}

</script>

<template>
  <div class="tabbar">
    <div class="tabbar_left">
      <el-icon @click="change" size="22">
        <component :is="layoutSettingStore.collapse?'Expand':'Fold'"/>
      </el-icon>
      <el-breadcrumb separator="/" separator-icon="ArrowRight">
        <el-breadcrumb-item v-for="(r, index) in route.matched" v-show="r.name!='layout'" :to="r.path" :key="index">
          {{ r.meta.title }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="tabbar_right">
      <el-button size="default" circle icon="Refresh" plain @click="refresh"></el-button>
      <el-button size="default" circle icon="FullScreen" plain @click="fullscreen"></el-button>
      <el-popover
          placement="bottom"
          title="主题设置"
          :width="300"
          trigger="hover"
      >
        <template #reference>
          <el-button size="default" circle icon="Setting" plain></el-button>
        </template>
        <el-form>
          <el-form-item label="主题颜色">
            <el-color-picker
                v-model.trim="color"
                show-alpha
                :predefine="predefineColors"
                @change="changeColor"
            />
          </el-form-item>
          <el-form-item label="夜间模式">
            <el-switch
                v-model.trim="night"
                active-icon="MoonNight"
                inactive-icon="Sunny"
                inline-prompt
                @change="changeTheme"
            />
          </el-form-item>
        </el-form>
      </el-popover>
      <el-dropdown>
    <span class="el-dropdown-link">
      <img :src="userStore.account.avatar" alt="" style="width: 32px;height: 32px;border-radius: 32px">
      {{ userStore.account.name }}
      <el-icon class="el-icon--right">
        <arrow-down/>
      </el-icon>
    </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item @click="logout">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<style scoped lang="scss">
.tabbar {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: space-between;
  background: url("@/assets/images/tabbar-bg.avif") no-repeat top center;
  //线性渐变
  //background-image: linear-gradient(to right, white, rgb(242,242,242));

  .tabbar_left {
    display: flex;
    align-items: center;

    .el-icon {
      margin: 0 14px;
    }
  }

  .tabbar_right {
    display: flex;
    align-items: center;

    .el-dropdown-link {
      display: flex;
      align-items: center;
      margin-right: 10px;

      img {
        margin: 0 10px;
      }
    }
  }
}
</style>