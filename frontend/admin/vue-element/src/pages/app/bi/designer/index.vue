<template>
  <div class="bi-designer-page">
    <BiDesigner
      :demo="true"
      :initial-schema="loadedSchema"
      @change="onSchemaChange"
    />
  </div>
</template>

<script setup lang="ts">
/**
 * BI 设计器页面：@go-wind/bi-vue BiDesigner 画布。
 * 变更落 localStorage（MVP 存储）；后续接后端 dashboard CRUD。
 */
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { BiDesigner, provideBiEnv, setBiLocale } from "@go-wind/bi-vue";
import "@go-wind/bi-vue/style.css";
import type { BiDashboardSchema } from "@go-wind/bi-core";
import { useI18n } from "@/core/i18n";

defineOptions({ name: "BiDesignerPage" });

const router = useRouter();
const { locale } = useI18n();

provideBiEnv({});

onMounted(() => {
  setBiLocale(locale.value.startsWith("zh") ? "zh-CN" : "en-US");
});

const loadedSchema = ref<Partial<BiDashboardSchema> | undefined>(undefined);

onMounted(() => {
  const saved = localStorage.getItem("bi:schema:default");
  if (saved) {
    try {
      loadedSchema.value = JSON.parse(saved);
    } catch {
      loadedSchema.value = undefined;
    }
  }
});

let saveTimer: ReturnType<typeof setTimeout> | null = null;

function onSchemaChange(schema: BiDashboardSchema) {
  // 防抖 1s 落盘
  if (saveTimer) clearTimeout(saveTimer);
  saveTimer = setTimeout(() => {
    localStorage.setItem("bi:schema:default", JSON.stringify(schema));
  }, 1000);
}

function goDashboard() {
  void router.push("/bi/dashboard");
}

defineExpose({ goDashboard });
</script>

<style scoped lang="scss">
.bi-designer-page {
  height: 100%;
  padding: 12px 16px;
  box-sizing: border-box;
}
</style>
