<script lang="ts" setup>
import type { auditservicev1_LoginAuditLog } from '#/api';

import { ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';

import { Card, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';

import {
  getLoginAuditLogStatusColor,
  getLoginAuditLogActionTypeColor,
  getLoginAuditLogRiskLevelColor,
  loginAuditLogStatusToName,
  loginAuditLogActionTypeToName,
  loginAuditLogRiskLevelToName,
} from '#/api';
import { $t } from '#/locales';

const row = ref<auditservicev1_LoginAuditLog>();

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      row.value = drawerApi.getData<{ row: auditservicev1_LoginAuditLog }>()?.row;
    } else {
      row.value = undefined;
    }
  },
});
</script>

<template>
  <Drawer :title="$t('page.loginAuditLog.moduleName')" class="w-full max-w-[800px]">
    <div class="login-audit-detail">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.loginAuditLog.sectionBasic')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.createdAt')"
          >{{ row?.createdAt || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.loginAuditLog.status')">
            <Tag
              v-if="row?.status"
              :color="getLoginAuditLogStatusColor(row.status)"
            >
              {{ loginAuditLogStatusToName(row.status) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.username')"
          >{{ row?.username || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.userId')"
          >{{ row?.userId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.loginAuditLog.actionType')">
            <Tag
              v-if="row?.actionType"
              :color="getLoginAuditLogActionTypeColor(row.actionType)"
            >
              {{ loginAuditLogActionTypeToName(row.actionType) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.loginMethod')"
          >{{ row?.loginMethod || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.loginAuditLog.riskLevel')">
            <Tag
              v-if="row?.riskLevel"
              :color="getLoginAuditLogRiskLevelColor(row.riskLevel)"
            >
              {{ loginAuditLogRiskLevelToName(row.riskLevel) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.riskScore')"
          >{{ row?.riskScore ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.ipAddress')"
          >{{ row?.ipAddress || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.tenantName')"
          >{{ row?.tenantName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.geoLocation')"
          >
            {{ row?.geoLocation?.province || '' }}
            {{ row?.geoLocation?.city || '' }}
          </DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.loginAuditLog.sectionDevice')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.platform')"
          >{{ row?.deviceInfo?.platform || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.osName')"
          >{{ row?.deviceInfo?.osName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.browserName')"
          >{{ row?.deviceInfo?.browserName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.userAgent')"
          >{{ row?.deviceInfo?.userAgent || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.loginAuditLog.sectionOther')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.mfaStatus')"
          >{{ row?.mfaStatus || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.sessionId')"
          >{{ row?.sessionId || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.failureReason')"
          >{{ row?.failureReason || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.riskFactors')"
          >
            <Tag
              v-for="factor in row?.riskFactors"
              :key="factor"
              class="tag-item"
            >
              {{ factor }}
            </Tag>
            <span v-if="!row?.riskFactors?.length">-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.requestId')"
          >{{ row?.requestId || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.traceId')"
          >{{ row?.traceId || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.logHash')"
          >{{ row?.logHash || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.loginAuditLog.signature')"
          >{{ row?.signature || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>
    </div>
  </Drawer>
</template>

<style scoped>
.login-audit-detail {
  padding-right: 10px;
}

.tag-item {
  margin: 0 4px 4px 0;
}
</style>
