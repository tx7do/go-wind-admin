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
      series: [
        {
          animationDelay() {
            return Math.random() * 400;
          },
          animationEasing: "exponentialInOut",
          animationType: "scale",
          center: ["50%", "50%"],
          color: ["#5ab1ef", "#b6a2de", "#67e0e3", "#2ec7c9"],
          data: [
            { name: "外包", value: 500 },
            { name: "定制", value: 310 },
            { name: "技术支持", value: 274 },
            { name: "远程", value: 400 },
          ].sort((a, b) => a.value - b.value),
          name: "商业占比",
          radius: "80%",
          roseType: "radius",
          type: "pie",
        },
      ],
      tooltip: {
        trigger: "item",
      },
    });
  }
});

window.addEventListener("resize", () => {
  chartInstance?.resize();
});
</script>
