<script setup lang="ts">
import {useRouter} from "vue-router";
import  moment from "moment";
import {onBeforeUnmount, onMounted, ref} from "vue";

defineOptions({
  name: 'Top'
})

const router = useRouter()

const go = (path:string)=>{
  router.push(path)
}

const time = ref<string>(moment().format('YYYY年MM月DD日 HH:mm:ss'))
const timer = ref<number>(0)
onMounted(()=>{
 timer.value =  setInterval(()=>{
    time.value = moment().format('YYYY年MM月DD日 HH:mm:ss')
  }, 1000)
})

onBeforeUnmount(()=>{
  clearInterval(timer.value)
})
</script>

<template>
  <div class="top">
    <div class="left">
      <span class="lbtn" @click="go('/home')">首页</span>
    </div>
    <div class="center">
      <div class="title">智慧旅游可视化大数据平台</div>

    </div>
    <div class="right">
      <span class="rbtn">统计报告</span>
      <span class="time">当前时间: {{ time }}</span>
    </div>
  </div>
</template>

<style scoped lang="scss">
.top {
  width: 100%;
  height: 40px;
  display: flex;
  font-family: YouSheBiaoTiHei, serif;

  .left{
    flex: 1.5;
    background: url("@/assets/images/screen/header-left-bg.png");
    background-size: cover;

    .lbtn{
      width: 130px;
      height: 40px;
      float: right;
      background: url("@/assets/images/screen/header-btn-bg-l.png");
      background-size: 100% 100%;

      text-align: center;
      line-height: 40px;
      color: #29fcff;
      font-size: 18px;
    }
  }
  .center{
    flex: 2;

    .title{
      width: 100%;
      height: 74px;
      background: url("@/assets/images/screen/header-center-bg.png");
      background-size: 100% 100%;

      text-align: center;
      line-height: 74px;
      color: #29fcff;
      font-size: 32px;
    }
  }
  .right{
    flex: 1.5;
    background: url("@/assets/images/screen/header-right-bg.png");
    background-size: cover;

    display: flex;
    justify-content: space-between;

    .rbtn{
      width: 130px;
      height: 40px;
      background: url("@/assets/images/screen/header-btn-bg-r.png");
      background-size: 100% 100%;

      text-align: center;
      line-height: 40px;
      color: #29fcff;
      font-size: 18px;
    }

    .time{
      color: #29fcff;
      text-align: center;
      line-height: 40px;
      margin-right: 20px;
    }
  }
}
</style>