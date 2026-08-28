<script lang="ts" setup>
import type { permissionservicev1_PolicyEvaluationLog } from '#/api';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';

import { Card, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';

import { $t } from '#/locales';

const row = ref<permissionservicev1_PolicyEvaluationLog>();

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      row.value =
        drawerApi.getData<{ row: permissionservicev1_PolicyEvaluationLog }>()
          ?.row;
    } else {
      row.value = undefined;
    }
  },
});

// 评估上下文是 JSON 字符串，pretty 展示便于阅读
const prettyContext = computed(() => {
  if (!row.value?.evaluationContext) return '-';
  try {
    return JSON.stringify(
      JSON.parse(row.value.evaluationContext),
      null,
      2,
    );
  } catch {
    return row.value.evaluationContext;
  }
});
</script>

<template>
  <Drawer
    :title="$t('page.policyEvaluationLog.moduleName')"
    class="w-full max-w-[860px]"
  >
    <div class="audit-detail">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.policyEvaluationLog.sectionBasic')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.createdAt')"
          >{{ row?.createdAt || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.policyEvaluationLog.result')">
            <Tag :color="row?.result ? 'green' : 'red'">
              {{
                row?.result
                  ? $t('page.policyEvaluationLog.resultPass')
                  : $t('page.policyEvaluationLog.resultDenied')
              }}
            </Tag>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.policyId')"
          >{{ row?.policyId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.permissionId')"
          >{{ row?.permissionId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.userId')"
          >{{ row?.userId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.membershipId')"
          >{{ row?.membershipId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.requestMethod')"
          >{{ row?.requestMethod || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.requestPath')"
          >{{ row?.requestPath || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.tenantId')"
          >{{ row?.tenantId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.ipAddress')"
          >{{ row?.ipAddress || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.traceId')"
          >{{ row?.traceId || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.policyEvaluationLog.effectDetails')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.effectDetails || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.policyEvaluationLog.evaluationContext')
          }}</span>
        </template>
        <pre class="detail-pre">{{ prettyContext }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.policyEvaluationLog.scopeSql')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.scopeSql || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.policyEvaluationLog.sectionOther')
          }}</span>
        </template>
        <Descriptions :column="1" bordered>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.logHash')"
          >
            <span class="hash-text">{{ row?.logHash || '-' }}</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.policyEvaluationLog.signature')"
          >
            <span class="hash-text">{{ row?.signature || '-' }}</span>
          </DescriptionsItem>
        </Descriptions>
      </Card>
    </div>
  </Drawer>
</template>

<style scoped>
.audit-detail {
  padding-right: 10px;
}

.detail-pre {
  margin: 0;
  padding: 8px 12px;
  max-height: 260px;
  overflow: auto;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  background-color: rgb(128 128 128 / 10%);
}

.hash-text {
  font-size: 12px;
  word-break: break-all;
}
</style>
