<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import type {
  monitor_alertservicev1_MonitorAlertRule as MonitorAlertRule,
} from '#/api';

import { reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListMonitorAlertRules,
  PaginationQuery,
  useCreateMonitorAlertRule,
  useDeleteMonitorAlertRule,
  useEvaluateMonitorAlerts,
  useUpdateMonitorAlertRule,
} from '#/api';

/**
 * 监控告警规则：指标阈值 → 触发通知（联动通知域）。
 *
 * 渠道与目标显式写在规则上（不经路由表）。评估每 5 分钟自动跑一轮
 * （系统级常驻任务），「立即评估」手动触发同一内核；
 * 告警→恢复成对通知，持续越限按冷却间隔重发。
 */

const { mutateAsync: createRule } = useCreateMonitorAlertRule();
const { mutateAsync: updateRule } = useUpdateMonitorAlertRule();
const { mutateAsync: deleteRule } = useDeleteMonitorAlertRule();
const { mutateAsync: evaluateAlerts } = useEvaluateMonitorAlerts();

const metricOptions = [
  { value: 'GO_GOROUTINES', label: $t('page.monitorAlert.metricMap.GO_GOROUTINES') },
  { value: 'GO_MEM_ALLOC_MB', label: $t('page.monitorAlert.metricMap.GO_MEM_ALLOC_MB') },
  { value: 'DB_OPEN_CONNECTIONS', label: $t('page.monitorAlert.metricMap.DB_OPEN_CONNECTIONS') },
  { value: 'DB_PING_FAIL', label: $t('page.monitorAlert.metricMap.DB_PING_FAIL') },
  { value: 'REDIS_DB_SIZE', label: $t('page.monitorAlert.metricMap.REDIS_DB_SIZE') },
];

const opOptions = [
  { value: 'GE', label: $t('page.monitorAlert.opMap.GE') },
  { value: 'LE', label: $t('page.monitorAlert.opMap.LE') },
];

const channelOptions = [
  { value: 'EMAIL', label: $t('page.monitorAlert.channelMap.EMAIL') },
  { value: 'WEBHOOK', label: $t('page.monitorAlert.channelMap.WEBHOOK') },
];

function metricToName(metric?: string) {
  return metricOptions.find((o) => o.value === metric)?.label ?? '-';
}

function opToName(op?: string) {
  return opOptions.find((o) => o.value === op)?.label ?? '-';
}

function channelToName(channel?: string) {
  return channelOptions.find((o) => o.value === channel)?.label ?? '-';
}

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.monitorAlert.name'),
      componentProps: { placeholder: $t('ui.placeholder.input'), allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<MonitorAlertRule> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: { isHover: true, keyField: 'id' },
  stripe: true,
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        const values = (formValues ?? {}) as Record<string, any>;
        return fetchListMonitorAlertRules(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: { name: values.name },
          }),
        );
      },
    },
  },
  columns: [
    { title: $t('page.monitorAlert.name'), field: 'name', minWidth: 150 },
    {
      title: $t('page.monitorAlert.metric'),
      field: 'metric',
      slots: { default: 'metric' },
      width: 150,
    },
    {
      title: $t('page.monitorAlert.condition'),
      field: 'condition',
      slots: { default: 'condition' },
      width: 140,
    },
    {
      title: $t('page.monitorAlert.cooldownMinutes'),
      field: 'cooldownMinutes',
      width: 110,
      formatter: ({ cellValue }) => `${cellValue ?? 30} min`,
    },
    {
      title: $t('page.monitorAlert.channel'),
      field: 'channel',
      slots: { default: 'channel' },
      width: 90,
    },
    {
      title: $t('page.monitorAlert.target'),
      field: 'target',
      minWidth: 200,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.monitorAlert.lastFiring'),
      field: 'lastFiring',
      slots: { default: 'lastFiring' },
      width: 100,
    },
    {
      title: $t('page.monitorAlert.lastAlertedAt'),
      field: 'lastAlertedAt',
      formatter: 'formatDateTime',
      width: 170,
    },
    {
      title: $t('page.monitorAlert.isEnabled'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      width: 90,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 160,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// ============ 创建/编辑 ============
const editOpen = ref(false);
const editMode = ref<'create' | 'edit'>('create');
const editSaving = ref(false);
const editingId = ref<number>();

const form = reactive({
  channel: 'EMAIL',
  cooldownMinutes: 30,
  isEnabled: true,
  metric: undefined as string | undefined,
  name: '',
  op: 'GE',
  remark: '',
  target: '',
  threshold: undefined as number | undefined,
});

function resetForm() {
  form.name = '';
  form.metric = undefined;
  form.op = 'GE';
  form.threshold = undefined;
  form.cooldownMinutes = 30;
  form.channel = 'EMAIL';
  form.target = '';
  form.isEnabled = true;
  form.remark = '';
}

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  resetForm();
  editOpen.value = true;
}

function openEdit(row: MonitorAlertRule) {
  editMode.value = 'edit';
  editingId.value = row.id;
  form.name = row.name || '';
  form.metric = row.metric;
  form.op = row.op || 'GE';
  form.threshold = row.threshold;
  form.cooldownMinutes = row.cooldownMinutes ?? 30;
  form.channel = row.channel || 'EMAIL';
  form.target = row.target || '';
  form.isEnabled = !!row.isEnabled;
  form.remark = row.remark || '';
  editOpen.value = true;
}

function handleMetricChange(value: any) {
  // 切到布尔指标时清掉无意义的比较条件，避免把残留值提交给服务端
  if (value === 'DB_PING_FAIL') {
    form.op = 'GE';
    form.threshold = undefined;
  }
}

async function handleSave() {
  if (!form.name) {
    message.error($t('page.monitorAlert.requiredName'));
    return;
  }
  if (!form.metric) {
    message.error($t('page.monitorAlert.requiredMetric'));
    return;
  }
  const isPingFail = form.metric === 'DB_PING_FAIL';
  if (!isPingFail && !form.op) {
    message.error($t('page.monitorAlert.requiredOp'));
    return;
  }
  if (!isPingFail && (form.threshold === undefined || form.threshold === null)) {
    message.error($t('page.monitorAlert.requiredThreshold'));
    return;
  }
  if (!form.target) {
    message.error($t('page.monitorAlert.requiredTarget'));
    return;
  }

  editSaving.value = true;
  try {
    // CRUD 的请求体必须包 data；DB_PING_FAIL 是布尔指标，op/threshold 不给（服务端忽略）。
    // form 侧 metric/op 是宽 string，经断言收窄到 proto 枚举字面量
    const data = {
      channel: form.channel,
      cooldownMinutes: form.cooldownMinutes,
      isEnabled: form.isEnabled,
      metric: form.metric,
      name: form.name,
      op: isPingFail ? undefined : form.op,
      remark: form.remark,
      target: form.target,
      threshold: isPingFail ? undefined : form.threshold,
    };
    const ruleData = data as MonitorAlertRule;
    if (editMode.value === 'create') {
      await createRule({ data: ruleData });
      message.success($t('page.monitorAlert.createSuccess'));
    } else if (editingId.value) {
      await updateRule({
        data: ruleData,
        id: editingId.value,
        updateMask: 'name,metric,op,threshold,cooldownMinutes,channel,target,isEnabled,remark',
      });
      message.success($t('page.monitorAlert.updateSuccess'));
    }
    editOpen.value = false;
    await gridApi.reload();
  } catch (error: any) {
    console.error('[monitor-alert] save failed', error);
    message.error(error?.message || $t('page.monitorAlert.saveFailed'));
  } finally {
    editSaving.value = false;
  }
}

// ============ 删除 ============
async function handleDelete(row: MonitorAlertRule) {
  try {
    await deleteRule({ id: row.id });
    notification.success({ message: $t('page.monitorAlert.deleteSuccess') });
    await gridApi.reload();
  } catch (error: any) {
    console.error('[monitor-alert] delete failed', error);
    notification.error({
      message: error?.message || $t('page.monitorAlert.deleteFailed'),
    });
  }
}

// ============ 立即评估 ============
const evaluateOpen = ref(false);
const evaluateSaving = ref(false);
const evaluateOutcomes = ref<any[]>([]);

function openEvaluate() {
  evaluateOutcomes.value = [];
  evaluateOpen.value = true;
  handleEvaluate();
}

async function handleEvaluate() {
  evaluateSaving.value = true;
  try {
    // body:"*" 自定义路由，请求体扁平：包一层 { data } 会被 protojson 当未知字段丢掉
    const resp = await evaluateAlerts({});
    evaluateOutcomes.value = (resp.outcomes ?? []) as any[];
  } catch (error: any) {
    console.error('[monitor-alert] evaluate failed', error);
    message.error(error?.message || $t('page.monitorAlert.evaluateFailed'));
  } finally {
    evaluateSaving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.monitorAlerts')">
      <template #toolbar-tools>
        <a-button class="mr-2" @click="openEvaluate">
          {{ $t('page.monitorAlert.evaluateNow') }}
        </a-button>
        <a-button type="primary" @click="openCreate">
          {{ $t('page.monitorAlert.create') }}
        </a-button>
      </template>
      <template #metric="{ row }">
        <a-tag v-if="row.metric" color="geekblue">
          {{ metricToName(row.metric) }}
        </a-tag>
        <template v-else>-</template>
      </template>
      <template #condition="{ row }">
        <template v-if="row.metric === 'DB_PING_FAIL'">
          {{ $t('page.monitorAlert.pingFailCondition') }}
        </template>
        <template v-else>{{ opToName(row.op) }} {{ row.threshold ?? '-' }}</template>
      </template>
      <template #channel="{ row }">
        <a-tag v-if="row.channel">{{ channelToName(row.channel) }}</a-tag>
        <template v-else>-</template>
      </template>
      <template #lastFiring="{ row }">
        <a-tag v-if="row.lastFiring" color="red">
          {{ $t('page.monitorAlert.firingNow') }}
        </a-tag>
        <a-tag v-else color="green">{{ $t('page.monitorAlert.firingNo') }}</a-tag>
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="row.isEnabled ? 'green' : 'default'">
          {{
            row.isEnabled
              ? $t('page.monitorAlert.enabledOn')
              : $t('page.monitorAlert.enabledOff')
          }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button type="link" size="small" @click.stop="openEdit(row)">
          {{ $t('page.monitorAlert.edit') }}
        </a-button>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('page.monitorAlert.deleteConfirm')"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small">
            {{ $t('page.monitorAlert.delete') }}
          </a-button>
        </a-popconfirm>
      </template>
    </Grid>

    <!-- 创建/编辑 -->
    <a-modal
      v-model:open="editOpen"
      :title="
        editMode === 'create'
          ? $t('page.monitorAlert.create')
          : $t('page.monitorAlert.edit')
      "
      :confirm-loading="editSaving"
      :mask-closable="false"
      destroy-on-close
      @ok="handleSave"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item :label="$t('page.monitorAlert.name')" required>
          <a-input
            v-model:value="form.name"
            :maxlength="100"
            :placeholder="$t('ui.placeholder.input')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.monitorAlert.metric')" required>
          <a-select
            v-model:value="form.metric"
            :options="metricOptions"
            :placeholder="$t('ui.placeholder.select')"
            @change="handleMetricChange"
          />
        </a-form-item>
        <template v-if="form.metric !== 'DB_PING_FAIL'">
          <a-form-item :label="$t('page.monitorAlert.op')" required>
            <a-select v-model:value="form.op" :options="opOptions" />
          </a-form-item>
          <a-form-item :label="$t('page.monitorAlert.threshold')" required>
            <a-input-number
              v-model:value="form.threshold"
              :precision="2"
              :step="1"
              style="width: 100%"
            />
          </a-form-item>
        </template>
        <a-form-item
          :help="$t('page.monitorAlert.cooldownHint')"
          :label="$t('page.monitorAlert.cooldownMinutes')"
        >
          <a-input-number
            v-model:value="form.cooldownMinutes"
            :max="1440"
            :min="1"
            :precision="0"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item :label="$t('page.monitorAlert.channel')" required>
          <a-select v-model:value="form.channel" :options="channelOptions" />
        </a-form-item>
        <a-form-item
          :help="$t('page.monitorAlert.targetHint')"
          :label="$t('page.monitorAlert.target')"
          required
        >
          <a-input
            v-model:value="form.target"
            :maxlength="500"
            :placeholder="$t('ui.placeholder.input')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.monitorAlert.isEnabled')">
          <a-switch v-model:checked="form.isEnabled" />
        </a-form-item>
        <a-form-item :label="$t('page.monitorAlert.remark')">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 立即评估 -->
    <a-modal
      v-model:open="evaluateOpen"
      :title="$t('page.monitorAlert.evaluateResultTitle')"
      :footer="null"
      width="680px"
      destroy-on-close
    >
      <a-empty
        v-if="evaluateOutcomes.length === 0"
        :description="$t('page.monitorAlert.evaluateEmpty')"
      />
      <a-table
        v-else
        :columns="[
          { title: $t('page.monitorAlert.name'), dataIndex: 'name', minWidth: 140 },
          { title: $t('page.monitorAlert.evaluateState'), dataIndex: 'firing', key: 'firing', width: 90 },
          { title: $t('page.monitorAlert.evaluateValue'), dataIndex: 'currentValue', key: 'currentValue', width: 110 },
          { title: $t('page.monitorAlert.evaluateNotified'), dataIndex: 'notified', key: 'notified', width: 90 },
          { title: $t('page.monitorAlert.evaluateReason'), dataIndex: 'reason', ellipsis: true },
        ]"
        :data-source="evaluateOutcomes"
        :loading="evaluateSaving"
        :pagination="false"
        row-key="ruleId"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'firing'">
            <a-tag :color="record.firing ? 'red' : 'green'">
              {{
                record.firing
                  ? $t('page.monitorAlert.firingNow')
                  : $t('page.monitorAlert.firingNo')
              }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'currentValue'">
            {{ record.currentValue != null ? Number(record.currentValue).toFixed(2) : '-' }}
          </template>
          <template v-else-if="column.key === 'notified'">
            {{ record.notified ? $t('page.monitorAlert.notifiedYes') : '-' }}
          </template>
        </template>
      </a-table>
    </a-modal>
  </Page>
</template>
