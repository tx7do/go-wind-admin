<template>
  <!-- 图片模板 -->
  <template v-if="col.cellType === 'image'">
    <template v-if="field">
      <template v-if="Array.isArray(row[field])">
        <ElImage
          v-for="(item, idx) in row[field]"
          :key="idx"
          :src="item"
          :preview-src-list="row[field]"
          :initial-index="Number(idx)"
          :preview-teleported="true"
          :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
        />
      </template>
      <template v-else>
        <ElImage
          :src="row[field]"
          :preview-src-list="[row[field]]"
          :preview-teleported="true"
          :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
        />
      </template>
    </template>
  </template>

  <!-- 标签模板 -->
  <template v-else-if="col.cellType === 'tag'">
    <ElTag :type="getTagType(row[field], col)">
      {{ (col.labelMap ?? {})[row[field]] ?? row[field] }}
    </ElTag>
  </template>

  <!-- 开关模板 -->
  <template v-else-if="col.cellType === 'switch'">
    <ElSwitch
      v-if="field"
      v-model="row[field]"
      :active-value="col.activeValue ?? 1"
      :inactive-value="col.inactiveValue ?? 0"
      :inline-prompt="true"
      :active-text="col.activeText ?? ''"
      :inactive-text="col.inactiveText ?? ''"
      :validate-event="false"
      @change="(val: any) => emit('modify', { row, field, value: val })"
    />
  </template>

  <!-- 日期模板 -->
  <template v-else-if="col.cellType === 'date'">
    {{ row[field] ? useDateFormat(row[field], col.dateFormat ?? "YYYY-MM-DD HH:mm:ss").value : "" }}
  </template>

  <!-- 链接模板 -->
  <template v-else-if="col.cellType === 'link'">
    <ElLink type="primary" :href="row[field]" target="_blank">
      {{ row[field] }}
    </ElLink>
  </template>

  <!-- 价格模板 -->
  <template v-else-if="col.cellType === 'price'">
    {{ `${col.pricePrefix ?? ""}${row[field]}` }}
  </template>

  <!-- 百分比模板 -->
  <template v-else-if="col.cellType === 'percent'">{{ row[field] }}%</template>

  <!-- 图标模板 -->
  <template v-else-if="col.cellType === 'icon'">
    <ElIcon><component :is="row[field]" /></ElIcon>
  </template>

  <!-- 操作列模板 -->
  <template v-else-if="col.cellType === 'tool'">
    <template v-for="(btn, idx) in col.buttons" :key="idx">
      <AccessControl :codes="btn.auth ? (Array.isArray(btn.auth) ? btn.auth : [btn.auth]) : undefined">
        <ElButton
          v-if="btn.visible === undefined || btn.visible(row)"
          v-bind="{ link: true, size: 'small', ...btn.attrs }"
          @click="emit('operate', { name: btn.name, row, $index: rowIndex })"
        >
          {{ btn.text ?? btn.name }}
        </ElButton>
      </AccessControl>
    </template>
  </template>

  <!-- 默认显示 -->
  <template v-else>
    {{ field ? row[field] : "" }}
  </template>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useDateFormat } from "@vueuse/core";
import { ElImage, ElTag, ElSwitch, ElLink, ElButton, ElIcon } from "element-plus";
import { AccessControl } from "@/core/access";
import type { ProTableColumn } from "./types";

const props = defineProps<{
  col: ProTableColumn;
  row: any;
  rowIndex: number;
}>();

const emit = defineEmits<{
  modify: [data: { row: any; field: string; value: any }];
  operate: [data: { name: string; row: any; $index: number }];
}>();

// 确保 prop 始终为 string，避免 undefined 索引类型错误
const field = computed(() => props.col.prop ?? "");

function getTagType(
  value: any,
  col: ProTableColumn
): "primary" | "success" | "warning" | "danger" | "info" {
  if (col.tagType) return col.tagType as any;
  return value ? "success" : "danger";
}
</script>
