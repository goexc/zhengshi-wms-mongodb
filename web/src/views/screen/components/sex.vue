<script setup lang="ts">
import * as echarts from 'echarts'
import {EChartOption} from "echarts";
import {onMounted, ref} from "vue";

defineOptions({
  name: 'Sex'
})

//图表实例
const echartsRef = ref()
const initEcharts = (data: { man: number, woman: number }) => {
  //初始化echarts实例
  let ec = echarts.init(echartsRef.value)

  //配置
  let option: EChartOption = {
    xAxis: {
      type: "value",
      show: false
    },
    grid: {
      left: 0,
      top: "30px",
      bottom: 0,
      right: 0
    },
    yAxis: [
      {
        type: "category",
        position: "left",
        data: ["男生"],
        axisTick: {
          show: false
        },
        axisLine: {
          show: false
        },
        axisLabel: {
          show: false
        }
      },
      {
        type: "category",
        position: "right",
        data: ["女士"],
        axisTick: {
          show: false
        },
        axisLine: {
          show: false
        },
        axisLabel: {
          show: false,
          padding: [0, 0, 40, -60],
          fontSize: 12,
          lineHeight: 60,
          color: "rgba(255, 255, 255, 0.9)",
          formatter: "{value}" + data.woman * 100 + "%",
          rich: {
            a: {
              color: "transparent",
              lineHeight: 30,
              fontFamily: "digital",
              fontSize: 12
            }
          }
        }
      }
    ],
    series: [
      {
        type: "bar",
        barWidth: 20,
        data: [data.man],
        z: 20,
        itemStyle: {
          borderRadius: 10,
          color: "#007AFE"
        },
        label: {
          show: true,
          color: "#E7E8ED",
          position: "insideLeft",
          offset: [0, -20],
          fontSize: 12,
          formatter: () => {
            return `男士 ${data.man * 100}%`;
          }
        }
      },
      {
        type: "bar",
        barWidth: 20,
        data: [1],
        barGap: "-100%",
        itemStyle: {
          borderRadius: 10,
          color: "#FF4B7A"
        },
        label: {
          show: true,
          color: "#E7E8ED",
          position: "insideRight",
          offset: [0, -20],
          fontSize: 12,
          formatter: () => {
            return `女士 ${data.woman * 100}%`;
          }
        }
      }
    ]
  };


  // ec.setOption(option)
  ec.setOption(option)

}

onMounted(() => {
  initEcharts({man:0.8, woman:0.2})
})
</script>

<template>
  <div class="box">
    <div class="title">
      <p>性别比例</p>
      <img src="@/assets/images/screen/title.png" alt="">
    </div>
    <div class="sex">
      <div class="man">
        <img src="@/assets/images/screen/man.png" alt="">
      </div>
      <div class="women">
        <img src="@/assets/images/screen/woman.png" alt="">
      </div>
    </div>
    <div class="echarts" ref="echartsRef"></div>
  </div>
</template>

<style scoped lang="scss">
.box {
  width: 100%;
  height: 100%;

  background: url("@/assets/images/screen/main-lt.png") no-repeat center center;
  background-size: 100% 100%;

  margin-top: 20px;
  //padding: 12px 42px 20px;

  color: white;

  .title {
    margin-left: 20px;

    color: white;
    font-size: 20px;
    font-family: YouSheBiaoTiHei, serif;
  }

  .sex {
    display: flex;
    justify-content: center;
    //justify-content: space-evenly;
    //flex-direction: row;
    //margin-top: 20px;

    .man {
      margin: 20px;

      display: flex;
      justify-content: center;
      align-items: center;
      width: 111px;
      height: 116px;
      background: url("@/assets/images/screen/man-bg.png");
    }

    .women {
      margin: 20px;

      display: flex;
      justify-content: center;
      align-items: center;
      width: 111px;
      height: 116px;
      background: url("@/assets/images/screen/woman-bg.png");
    }
  }

  .echarts {
    height: 100px;
    margin: 0 30px;
  }

}
</style>