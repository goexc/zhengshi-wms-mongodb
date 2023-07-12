<script setup lang="ts">
import {onMounted, ref} from "vue";
import * as echarts from 'echarts'
//引入echarts 水球图插件
import 'echarts-liquidfill'
import {ECharts} from "echarts";

defineOptions({
  name: 'Tourist'
})

//预约人数
const reserve = ref<string>('10962人')

//获取节点
const echartsRef = ref()

//初始化echarts
const initEchats = (data:number=1): ECharts=>{
  //获取echarts类的实例
  let ec = echarts.init(echartsRef.value)
  //设置实例的配置项
  let option = {
    title: [
      {
        text: (data * 100).toFixed(0) + "%",
        left: "49%",
        top: "35%",
        textAlign: "center",
        textStyle: {
          fontSize: "14",
          fontWeight: "normal",
          color: "#ffffff",
          align: "center",
          textBorderColor: "rgba(0, 0, 0, 0)",
          textShadowColor: "#000",
          textShadowBlur: 0,
          textShadowOffsetX: 0,
          textShadowOffsetY: 1
        }
      },
      {
        text: "预约量",
        left: "49%",
        top: "25%",
        textAlign: "center",
        textStyle: {
          fontSize: "15",
          fontWeight: "normal",
          color: "#ffffff",
          align: "center",
          textBorderColor: "rgba(0, 0, 0, 0)",
          textShadowColor: "#000",
          textShadowBlur: 0,
          textShadowOffsetX: 0,
          textShadowOffsetY: 1
        }
      }
    ],
    grid: {
      top: "0",
      left: "0px",
      right: "0px",
      bottom: "0",
      containLabel: true
    },
    polar: {
      radius: ["75%", "85%"],
      center: ["50%", "50%"]
    },
    angleAxis: {
      max: 120,
      clockwise: false,
      axisLine: {
        show: false
      },
      axisTick: {
        show: false
      },
      axisLabel: {
        show: false
      },
      splitLine: {
        show: false
      },
      startAngle: 188
    },
    radiusAxis: {
      type: "category",
      show: true,
      axisLabel: {
        show: false
      },
      axisLine: {
        show: false
      },
      axisTick: {
        show: false
      }
    },
    series: [
      {
        type: "liquidFill",
        radius: "70%",
        z: 2,
        center: ["50%", "50%"],
        data: [0.4, 0.4, 0.4], // data个数代表波浪数
        itemStyle: {
          color: {
            type: "linear",
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              {
                offset: 0,
                color: "#35FAB6" // 0% 处的颜色
              },
              {
                offset: 1,
                color: "rgba(40, 209, 247,0.3)" // 100% 处的颜色
              }
            ],
            global: false // 缺省为 false
          }
        },
        outline: {
          borderDistance: 0,
          itemStyle: {
            borderWidth: 2,
            borderColor: "#31d8d5",
            shadowBlur: 20,
            shadowColor: "#50c1a7"
          }
        },
        label: {
          show: false
        },
        backgroundStyle: {
          borderWidth: 1,
          // 径向渐变，前三个参数分别是圆心 x, y 和半径，取值同线性渐变
          color: {
            type: "radial",
            x: 0.5,
            y: 0.5,
            r: 0.5,
            colorStops: [
              {
                offset: 0,
                color: "#0D2648" // 0% 处的颜色
              },
              {
                offset: 0.8,
                color: "#0D2648" // 100% 处的颜色
              },
              {
                offset: 1,
                color: "#228E7D" // 100% 处的颜色
              }
            ],
            global: false // 缺省为 false
          }
        }
      },
      {
        type: "pie",
        radius: ["80%", "80%"],
        center: ["50%", "50%"],
        z: 1,
        label: {
          show: false
        },
        silent: true,
        itemStyle: {
          borderWidth: 2,
          borderType: [8, 10],
          borderDashOffset: 15,
          borderColor: "#31d8d5",
          color: "#11144e",
          borderCap: "round"
        },
        data: [100]
      },
      {
        type: "bar",
        data: [55],
        z: 10,
        coordinateSystem: "polar",
        roundCap: true,
        color: "#31d8d5"
      }
    ]
  };

  ec.setOption(option)
  // return ec
}

onMounted(() => {
  initEchats(0.75)
})
</script>

<template>
  <div class="box">
    <div class="top">
      <p class="title">实时游客统计</p>
      <p class="bg"></p>
      <p class="right">可预约总量 <span>9999</span> 人</p>
    </div>
    <div class="number">
      <span v-for="(item, index) in reserve" :key="index">{{ item }}</span>
    </div>
    <!-- echarts展示图形的节点 -->
    <div class="echarts" ref="echartsRef"></div>
  </div>
</template>

<style scoped lang="scss">

.box {
  background: url("@/assets/images/screen/main-lt.png") no-repeat center center;
  background-size: 100% 100%;

  //margin-top: 20px;

  .top {
    margin-left: 20px;

    .title {
      color: white;
      font-size: 20px;
      font-family: YouSheBiaoTiHei,serif;
    }

    .bg {
      width: 68px;
      height: 7px;
      background: url("@/assets/images/screen/title.png") no-repeat center center;
      background-size: 100% 100%;
      margin-top: 10px;
    }

    .right {
      float: right;
      color: #ffffff;

      span {
        color: #ff8100;
      }
    }
  }

  .number {
    margin-top: 30px;
    display: flex;
    padding: 20px;

    span {
      flex: 1;
      height: 40px;
      line-height: 40px;
      text-align: center;
      color: #66ffff;
      font-size: 28px;

      background: url("@/assets/images/screen/total.png") no-repeat center center;
      background-size: 100% 100%;

    }

  }

  .echarts {
    width: 100%;
    height: 240px;
  }
}
</style>