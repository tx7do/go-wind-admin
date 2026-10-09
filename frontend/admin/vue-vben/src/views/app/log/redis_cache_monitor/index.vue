<script lang="ts" setup>
import type {
  redis_cacheservicev1_RedisCacheMonitorInfo,
  redis_cacheservicev1_SlowLogEntry,
} from '#/api/generated/admin/service/v1';

import { computed, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { formatDateTime } from '@vben/utils';

import {
  Card,
  Descriptions,
  DescriptionsItem,
  Empty,
  Table,
  Tag,
} from 'ant-design-vue';

import { useRedisCacheMonitorInfo } from '#/api';
import { $t } from '#/locales';

const { data } = useRedisCacheMonitorInfo();

const expandedSections = ref<Record<string, boolean>>({});
function toggleSection(key: string) {
  expandedSections.value = {
    ...expandedSections.value,
    [key]: !expandedSections.value[key],
  };
}

// data 是 vue-query 异步解析的 Ref：必须在 setup 时用 computed 派生，
// 否则快照到的是 undefined，数据返回后页面也不会更新（永远显示空态）。
const info = computed(
  () =>
    (data.value as redis_cacheservicev1_RedisCacheMonitorInfo | undefined) ??
    undefined,
);
const sections = computed(() => {
  const raw = info.value?.sections ?? [];
  return raw as {
    entries: { key: string; value: string }[];
    name: string;
  }[];
});
const dbSize = computed(() => info.value?.dbSize ?? 0);
const slowlog = computed(
  () => (info.value?.slowlog ?? []) as redis_cacheservicev1_SlowLogEntry[],
);

const slowLogColumns = [
  {
    title: $t('page.redisCacheMonitor.slowlogId'),
    dataIndex: 'id',
    width: 80,
  },
  {
    title: $t('page.redisCacheMonitor.createdAt'),
    dataIndex: 'createdAt',
    width: 180,
    customRender: ({ text }: { text: unknown }) => formatDateTime(text as any),
  },
  {
    title: $t('page.redisCacheMonitor.durationUsec'),
    dataIndex: 'durationUsec',
    width: 120,
  },
  {
    title: $t('page.redisCacheMonitor.clientAddr'),
    dataIndex: 'clientAddr',
    width: 200,
  },
  {
    title: $t('page.redisCacheMonitor.clientName'),
    dataIndex: 'clientName',
    width: 120,
  },
  {
    title: $t('page.redisCacheMonitor.args'),
    dataIndex: 'args',
  },
];
</script>

<template>
  <Page :auto-content-height="true">
    <div class="redis-cache-monitor-page">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.redisCacheMonitor.dbSizeCardTitle')
          }}</span>
        </template>
        <Descriptions :column="1" bordered>
          <DescriptionsItem :label="$t('page.redisCacheMonitor.dbSize')">
            <Tag :color="dbSize > 0 ? 'orange' : 'default'">
              {{ String(dbSize) }}
            </Tag>
          </DescriptionsItem>
        </Descriptions>
      </Card>

      <Card v-if="sections.length === 0" class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.redisCacheMonitor.infoCardTitle')
          }}</span>
        </template>
        <Empty :description="$t('page.redisCacheMonitor.noInfo')" />
      </Card>

      <Card
        v-for="(section, idx) in sections"
        :key="`${section.name}-${idx}`"
        class="mb-4"
      >
        <template #title>
          <div
            class="section-header"
            @click="toggleSection(`${section.name}-${idx}`)"
          >
            <span class="card-title">{{ section.name }}</span>
            <Tag>{{ section.entries.length }}</Tag>
            <span class="toggle-hint">
              {{
                expandedSections[`${section.name}-${idx}`]
                  ? $t('page.redisCacheMonitor.collapse')
                  : $t('page.redisCacheMonitor.expand')
              }}
            </span>
          </div>
        </template>
        <!-- 收起态用轻量提示行，不用 Empty 插画——Empty 语义是"无数据"，
             大片收起时会让整页看起来像加载失败/空表 -->
        <div
          v-if="!expandedSections[`${section.name}-${idx}`]"
          class="collapsed-hint"
        >
          {{ $t('page.redisCacheMonitor.collapsed') }}
        </div>
        <Empty
          v-else-if="section.entries.length === 0"
          :description="$t('page.redisCacheMonitor.noEntries')"
        />
        <Descriptions v-else :column="1" bordered>
          <DescriptionsItem
            v-for="(entry, eIdx) in section.entries"
            :key="eIdx"
            :label="entry.key"
          >
            <span class="entry-value">{{ entry.value }}</span>
          </DescriptionsItem>
        </Descriptions>
      </Card>

      <Card>
        <template #title>
          <span class="card-title">{{
            $t('page.redisCacheMonitor.slowlogCardTitle')
          }}</span>
        </template>
        <Empty
          v-if="slowlog.length === 0"
          :description="$t('page.redisCacheMonitor.noSlowlog')"
        />
        <Table
          v-else
          :columns="slowLogColumns"
          :data-source="slowlog"
          :pagination="false"
          row-key="id"
          size="small"
        />
      </Card>

      <div class="disclaimer">
        {{ $t('page.redisCacheMonitor.disclaimer') }}
      </div>
    </div>
  </Page>
</template>

<style scoped>
.redis-cache-monitor-page {
  padding: 16px;
}
.card-title {
  font-weight: 600;
}
.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.toggle-hint {
  font-size: 12px;
  color: var(--muted-foreground);
  margin-left: auto;
}
.collapsed-hint {
  padding: 4px 0;
  font-size: 12px;
  color: var(--muted-foreground);
}
.entry-value {
  word-break: break-all;
}
.disclaimer {
  margin-top: 16px;
  font-size: 12px;
  color: var(--muted-foreground);
}
</style>
