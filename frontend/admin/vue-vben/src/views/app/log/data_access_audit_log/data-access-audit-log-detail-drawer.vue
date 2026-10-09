<script lang="ts" setup>
import type { auditservicev1_DataAccessAuditLog } from '#/api';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';

import { Card, Descriptions, DescriptionsItem, Tag } from 'ant-design-vue';

import {
  dataAccessAuditLogAccessTypeToColor,
  dataAccessAuditLogAccessTypeToName,
  dataAccessAuditLogCategoryToColor,
  dataAccessAuditLogCategoryToName,
  successToColor,
} from '#/api';
import { $t } from '#/locales';

const row = ref<auditservicev1_DataAccessAuditLog>();

const [Drawer, drawerApi] = useVbenDrawer({
  footer: false,
  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      row.value =
        drawerApi.getData<{ row: auditservicev1_DataAccessAuditLog }>()?.row;
    } else {
      row.value = undefined;
    }
  },
});

// evaluationContext 类的长 JSON 原样展示；sqlDigest/logHash 等哈希截断展示
const signatureText = computed(() => row.value?.signature || '-');
</script>

<template>
  <Drawer
    :title="$t('page.dataAccessAuditLog.moduleName')"
    class="w-full max-w-[860px]"
  >
    <div class="audit-detail">
      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.dataAccessAuditLog.sectionBasic')
          }}</span>
        </template>
        <Descriptions :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.createdAt')"
          >{{ row?.createdAt || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.dataAccessAuditLog.success')">
            <Tag :color="successToColor(row?.success ?? false)">
              {{
                row?.success
                  ? $t('enum.successStatus.success')
                  : $t('enum.successStatus.failed')
              }}
            </Tag>
          </DescriptionsItem>
          <DescriptionsItem :label="$t('page.dataAccessAuditLog.accessType')">
            <Tag
              v-if="row?.accessType"
              :color="dataAccessAuditLogAccessTypeToColor(row.accessType)"
            >
              {{ dataAccessAuditLogAccessTypeToName(row.accessType) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.tableName')"
          >{{ row?.tableName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.dataCategory')"
          >
            <Tag
              v-if="row?.dataCategory"
              :color="dataAccessAuditLogCategoryToColor(row.dataCategory)"
            >
              {{ dataAccessAuditLogCategoryToName(row.dataCategory) }}
            </Tag>
            <span v-else>-</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.sensitiveLevel')"
          >{{ row?.sensitiveLevel || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.latencyMs')"
          >{{ row?.latencyMs ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.affectedRows')"
          >{{ row?.affectedRows ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.username')"
          >{{ row?.username || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.userId')"
          >{{ row?.userId ?? '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.tenantName')"
          >{{ row?.tenantName || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.dbUser')"
          >{{ row?.dbUser || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.dataSource')"
          >{{ row?.dataSource || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.ipAddress')"
          >{{ row?.ipAddress || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.businessPurpose')"
          >{{ row?.businessPurpose || '-' }}</DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.dataId')"
          >{{ row?.dataId || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.dataAccessAuditLog.sectionSql')
          }}</span>
        </template>
        <pre class="detail-pre">{{ row?.sqlText || '-' }}</pre>
        <Descriptions class="mt-3" :column="2" bordered>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.sqlDigest')"
          >
            <span class="hash-text">{{ row?.sqlDigest || '-' }}</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.maskingRules')"
          >{{ row?.maskingRules || '-' }}</DescriptionsItem>
          <DescriptionsItem :label="$t('page.dataAccessAuditLog.dataMasked')">
            <Tag :color="row?.dataMasked ? 'green' : 'default'">
              {{
                row?.dataMasked
                  ? $t('page.dataAccessAuditLog.dataMaskedYes')
                  : $t('page.dataAccessAuditLog.dataMaskedNo')
              }}
            </Tag>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.requestId')"
          >{{ row?.requestId || '-' }}</DescriptionsItem>
        </Descriptions>
      </Card>

      <Card class="mb-4">
        <template #title>
          <span class="card-title">{{
            $t('page.dataAccessAuditLog.sectionOther')
          }}</span>
        </template>
        <Descriptions :column="1" bordered>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.logHash')"
          >
            <span class="hash-text">{{ row?.logHash || '-' }}</span>
          </DescriptionsItem>
          <DescriptionsItem
            :label="$t('page.dataAccessAuditLog.signature')"
          >
            <span class="hash-text">{{ signatureText }}</span>
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
