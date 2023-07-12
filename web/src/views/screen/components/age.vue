<script setup lang="ts">
import {onMounted, ref} from "vue";
import * as echarts from "echarts";
import {EChartOption, ECharts} from "echarts";

defineOptions({
  name: 'Age'
})

interface ChartProp {
  value: number;
  name: string;
  percentage: string;
}

// 初始化 charts参数
let ageData:ChartProp[] = [
  {
    value: 200,
    name: "10岁以下",
    percentage: "16%"
  },
  {
    value: 110,
    name: "10 - 18岁",
    percentage: "8%"
  },
  {
    value: 150,
    name: "18 - 30岁",
    percentage: "12%"
  },
  {
    value: 310,
    name: "30 - 40岁",
    percentage: "24%"
  },
  {
    value: 250,
    name: "40 - 60岁",
    percentage: "20%"
  },
  {
    value: 260,
    name: "60岁以上",
    percentage: "20%"
  }
];

//图表实例
const echartsRef = ref()

const initEcharts = (data: ChartProp[]) => {
  const charEch: ECharts = echarts.init(echartsRef.value);
  /* echarts colors */
  const colors = ["#F6C95C", "#EF7D33", "#1F9393", "#184EA1", "#81C8EF", "#9270CA"];
  const option: EChartOption = {
    color: colors,
    tooltip: {
      show: true,
      trigger: "item",
      formatter: "{b} <br/>占比：{d}%"
    },
    legend: {
      orient: "vertical",
      right: "20px",
      top: "15px",
      itemGap: 15,
      itemWidth: 14,
      formatter: function (name) {
        let text = "";
        data.forEach((val:ChartProp) => {
          if (val.name === name) {
            text = " " + name + "　 " + val.percentage;
          }
        });
        return text;
      },
      textStyle: {
        color: "#fff"
      }
    },
    grid: {
      top: "bottom",
      left: 10,
      bottom: 10
    },
    series: [
      {
        zlevel: 1,
        name: "年龄比例",
        type: "pie",
        selectedMode: "single",
        radius: [50, 90],
        center: ["35%", "50%"],
        startAngle: 60,
        // hoverAnimation: false,
        label: {
          position: "inside",
          show: true,
          color: "#fff",
          formatter: function (params: any) {
            return params.data.percentage;
          },
          rich: {
            b: {
              fontSize: 16,
              lineHeight: 30,
              color: "#fff"
            }
          }
        },
        itemStyle: {
          shadowColor: "rgba(0, 0, 0, 0.2)",
          shadowBlur: 10
        },
        data: data.map((val, index: number) => {
          return {
            value: val.value,
            name: val.name,
            percentage: val.percentage,
            itemStyle: {
              borderWidth: 10,
              shadowBlur: 20,
              borderColor: colors[index],
              borderRadius: 10
            }
          };
        })
      },
      {
        name: "",
        type: "pie",
        selectedMode: "single",
        radius: [50, 90],
        center: ["35%", "50%"],
        startAngle: 60,
        data: [
          {
            value: 1000,
            name: "",
            label: {
              show: true,
              formatter: "{a|本日总数}",
              rich: {
                a: {
                  align: "center",
                  color: "rgb(98,137,169)",
                  fontSize: 14
                }
              },
              position: "center"
            }
          }
        ]
      }
    ]
  };
  charEch.setOption(option);

}

onMounted(() => {
  initEcharts(ageData)
})
</script>

<template>
  <div class="box">
    <div class="title">
      <p>年龄比例</p>
      <img src="@/assets/images/screen/title.png" alt="">
    </div>
    <div class="echarts" ref="echartsRef"></div>
  </div>
</template>

<style scoped lang="scss">
.box {
  width: 100%;
  height: 100%;

  background: url("@/assets/images/screen/main-lb.png") no-repeat center center;
  background-size: 100% 100%;

  margin-top: 20px;

  color: white;

  .title {
    margin-left: 20px;

    color: white;
    font-size: 20px;
    font-family: YouSheBiaoTiHei, serif;
  }

  .echarts {
    height: 250px;
  }
}
</style>