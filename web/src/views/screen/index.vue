<script setup lang="ts">
//控制大屏放大与缩小
import {onMounted, ref} from "vue";
import Top from "./components/top.vue";
import Tourist from "./components/tourist.vue";
import Age from "./components/age.vue";
import Sex from "./components/sex.vue";
import Map from "./components/map.vue";
import Line from "./components/line.vue";
import Rank from "./components/rank.vue";
import Year from "./components/year.vue";
import Counter from "./components/counter.vue";

//获取数据大屏内容盒子的dom元素
let screenRef = ref()


//计算缩放比例
const getScale = (w: number = 1920, h: number = 1080) => {
  const ww = window.innerWidth / w;
  const wh = window.innerHeight / h;
  return ww < wh ? ww : wh;
}

onMounted(() => {
  screenRef.value.style.transform = `scale(${getScale()}) translate(-50%,-50%)`
})

//监听窗口变化
window.onresize = ()=>{
  screenRef.value.style.transform = `scale(${getScale()}) translate(-50%,-50%)`
}
</script>

<template>
  <div class="container">
    <!-- 数据大屏展示内容区域 -->
    <div class="screen" ref="screenRef">
      <!-- 数据大屏顶部 -->
      <div class="top">
        <Top/>
      </div>
      <div class="main">
        <div class="left">
          <Tourist class="tourist"/>
          <Sex class="sex"/>
          <Age class="age"/>
        </div>
        <div class="center">
          <Map class="map"/>
          <Line class="line"/>
        </div>
        <div class="right">
          <Rank class="rank"/>
          <Year class="year"/>
          <Counter class="counter"/>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.container {
  width: 100vw;
  height: 100vh;
  background: url('@/assets/images/screen/bg.png') no-repeat center center;
  background-size: cover;

  .screen {
    position: fixed;
    width: 1920px;
    height: 1080px;
    top: 50%;
    left: 50%;
    transform-origin: left top;


    .top {
      display: flex;
      width: 100%;
      height: 40px;
    }

    .main {
      display: flex;
      flex: 1;
      padding: 12px 42px 20px;
      box-sizing:border-box;

      .left {
        flex: 1;
        //height: 1040px;
        display: flex;
        flex-direction: column;

        .tourist {
          flex: 1.2;
        }

        .sex {
          flex: 1;

        }

        .age {
          flex: 1;
        }
      }

      .center {
        flex: 2;
        display: flex;
        flex-direction: column;

        .map {
          flex: 2;
        }

        .line {
          flex: 0.8;
        }
      }

      .right {
        flex: 1;
        display: flex;
        flex-direction: column;
        margin-left: 40px;

        .rank {
          flex: 1.2;
        }

        .year {
          flex: 1;
        }

        .counter {
          flex: 1;
        }
      }
    }
  }
}
</style>