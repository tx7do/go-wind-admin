<script lang="ts" setup>
import type { auditservicev1_OperationAuditLog } from '#/api';

import { ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';

import { Card, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';

import {
  operationAuditLogActionToColor,
  operationAuditLogActionToName,
  successToColor,
} from '#/api';
import { $t } from '#/locales';

const row = ref<auditservicev1_OperationAuditLog>();

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      row.value = drawerApi.getData<{ row: auditservicev1_OperationAuditLog }>()?.row;
    } else {
      row.value = undefined;
    }
  },
});
</script>

<template>
  <Drawer :title="$t('page.operationAuditLog.moduleName')" class="w-full max-w-[800px]">
    <div class="audit-detail">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.operationAuditLog.sectionBasic')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.createdAt')"
          >{{ row?.createdAt || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.operationAuditLog.action')">
            <Tag
              v-if="row?.action"
              :color="operationAuditLogActionToColor(row.action)"
            >
              {{ operationAuditLogActionToName(row.action) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.resourceType')"
          >{{ row?.resourceType || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.resourceId')"
          >{{ row?.resourceId || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.operationAuditLog.success')">
            <Tag :color="successToColor(row?.success ?? false)">
              {{
                row?.success
                  ? $t('enum.successStatus.success')
                  : $t('enum.successStatus.failed')
              }}
            </Tag>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.sensitiveLevel')"
          >{{ row?.sensitiveLevel || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.username')"
          >{{ row?.username || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.userId')"
          >{{ row?.userId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.tenantName')"
          >{{ row?.tenantName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.ipAddress')"
          >{{ row?.ipAddress || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.geoLocation')"
          >
            {{ row?.geoLocation?.province || '' }}
            {{ row?.geoLocation?.city || '' }}
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.failureReason')"
          >{{ row?.failureReason || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.traceId')"
          >{{ row?.traceId || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.requestId')"
          >{{ row?.requestId || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.operationAuditLog.beforeData')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.beforeData || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.operationAuditLog.afterData')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.afterData || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.operationAuditLog.sectionOther')
          }}</span>
        </template>
        <Descriptions :column="1" bordered>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.logHash')"
          >{{ row?.logHash || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.operationAuditLog.signature')"
          >{{ row?.signature || '-' }}</DescriptionsItem>
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
</style>
