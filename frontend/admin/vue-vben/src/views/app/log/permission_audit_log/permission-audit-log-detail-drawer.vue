<script lang="ts" setup>
import type { auditservicev1_PermissionAuditLog } from '#/api';

import { ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';

import { Card, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';

import {
  permissionAuditLogActionToColor,
  permissionAuditLogActionToName,
} from '#/api';
import { $t } from '#/locales';

const row = ref<auditservicev1_PermissionAuditLog>();

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      row.value =
        drawerApi.getData<{ row: auditservicev1_PermissionAuditLog }>()?.row;
    } else {
      row.value = undefined;
    }
  },
});
</script>

<template>
  <Drawer
    :title="$t('page.permissionAuditLog.moduleName')"
    class="w-full max-w-[800px]"
  >
    <div class="audit-detail">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.permissionAuditLog.sectionBasic')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.createdAt')"
          >{{ row?.createdAt || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.permissionAuditLog.action')">
            <Tag
              v-if="row?.action"
              :color="permissionAuditLogActionToColor(row.action)"
            >
              {{ permissionAuditLogActionToName(row.action) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.targetType')"
          >{{ row?.targetType || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.targetId')"
          >{{ row?.targetId || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.targetName')"
          >{{ row?.targetName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.operatorId')"
          >{{ row?.operatorId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.operatorName')"
          >{{ row?.operatorName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.tenantId')"
          >{{ row?.tenantId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.ipAddress')"
          >{{ row?.ipAddress || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.reason')"
          >{{ row?.reason || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.requestId')"
          >{{ row?.requestId || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.permissionAuditLog.oldValue')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.oldValue || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.permissionAuditLog.newValue')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.newValue || '-' }}</pre>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.permissionAuditLog.sectionOther')
          }}</span>
        </template>
        <Descriptions :column="1" bordered>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.logHash')"
          >
            <span class="hash-text">{{ row?.logHash || '-' }}</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.permissionAuditLog.signature')"
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
