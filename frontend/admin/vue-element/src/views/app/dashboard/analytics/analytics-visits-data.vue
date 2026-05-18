<template>
  <div ref="chartRef" style="width: 100%; height: 400px"></div>
</template>

<script lang="ts" setup>
import { onMounted, ref } from "vue";
import * as echarts from "echarts";

const chartRef = ref<HTMLDivElement>();
let chartInstance: echarts.ECharts | null = null;

onMounted(() => {
  if (chartRef.value) {
    chartInstance = echarts.init(chartRef.value);
    chartInstance.setOption({
      legend: {
        bottom: 0,
        data: ["访问", "趋势"],
      },
      radar: {
        indicator: [
          { name: "网页" },
          { name: "移动端" },
          { name: "Ipad" },
          { name: "客户端" },
          { name: "第三方" },
          { name: "其它" },
        ],
        radius: "60%",
        splitNumber: 8,
      },
      series: [
        {
          areaStyle: {
            opacity: 1,
            shadowBlur: 0,
            shadowColor: "rgba(0,0,0,.2)",
            shadowOffsetX: 0,
            shadowOffsetY: 10,
          },
          data: [
            {
              itemStyle: { color: "#b6a2de" },
              name: "访问",
              value: [90, 50, 86, 40, 50, 20],
            },
            {
              itemStyle: { color: "#5ab1ef" },
              name: "趋势",
              value: [70, 75, 70, 76, 20, 85],
            },
          ],
          itemStyle: {
            borderRadius: 10,
            borderWidth: 2,
          },
          symbolSize: 0,
          type: "radar",
        },
      ],
      tooltip: {},
    });
  }
});

window.addEventListener("resize", () => {
  chartInstance?.resize();
});
</script>
