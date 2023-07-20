<script setup lang="ts">

import {useUserStore} from "@/store/modules/user.ts";
import {ref} from "vue";

const userStore = useUserStore()
const title = ref<string>(import.meta.env.VITE_TITLE)
</script>

<template>
  <div>
    <el-card>
      <div class="box">
        <el-image class="avatar" :src="userStore.account.avatar"/>
        <div class="user">
          <h2>{{ userStore.account.name }}</h2>
          <p>{{ title }}</p>
        </div>
      </div>
    </el-card>
    <div class="bottom">
      <svg-icon name="welcome" width="600px" height="400px"></svg-icon>
    </div>
  </div>


</template>

<style scoped lang="scss">
.box{
  display: flex;
  .avatar {
    width: 100px;
    height: 100px;
    border-radius: 100px;
  }
  .user{
    margin-left: 10px;
    h2{
      font-size: 30px;
      font-weight: 900;
      margin-bottom: 30px;
    }
    p{
      color: #cdd0d6;
      font-style: italic;
    }
  }
}

.bottom{
  display: flex;
  justify-content: center;
}
</style>