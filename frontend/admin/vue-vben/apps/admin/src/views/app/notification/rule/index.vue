<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import type {
  notificationservicev1_Channel as RuleChannel,
  notificationservicev1_EventType as RuleEventType,
  notificationservicev1_NotificationRule as NotificationRule,
} from '#/api';

import { reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListNotificationRules,
  NOTIFICATION_RULE_UPDATE_MASK,
  notificationRuleAsyncToColor,
  notificationRuleAsyncToName,
  notificationRuleChannelOptionList,
  notificationRuleChannelToColor,
  notificationRuleChannelToName,
  notificationRuleEnabledToColor,
  notificationRuleEnabledToName,
  notificationRuleEventTypeList,
  notificationRuleEventTypeToName,
  PaginationQuery,
  useCreateNotificationRule,
  useDeleteNotificationRule,
  useTestDispatchNotification,
  useUpdateNotificationRule,
} from '#/api';

/**
 * 通知路由规则管理（业务事件 → 投递渠道 + 派发方式）
 *
 * 这张表是投递时的唯一真相：改完这一行，下一次投递立刻按新路由走（服务端不缓存）。
 * 两个决定来自同一行，所以「渠道」与「派发方式」必须一起看：异步 = 请求入队即回，
 * 结论由队列回写台账；同步 = 当场拨号，接口的回话就是投递结论。
 *
 * 删掉一行 = 「这个事件不再通知」：服务端不会在下次启动补回来（播种只在空表时执行），
 * 但那条事件的调用会拿到「没有启用路由规则」的报错（不静默丢通知）。
 */

const { mutateAsync: createRule } = useCreateNotificationRule();
const { mutateAsync: updateRule } = useUpdateNotificationRule();
const { mutateAsync: deleteRule } = useDeleteNotificationRule();
const { mutateAsync: testDispatch } = useTestDispatchNotification();

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'eventType',
      label: $t('page.notificationRule.eventType'),
      componentProps: {
        options: notificationRuleEventTypeList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'channel',
      label: $t('page.notificationRule.channel'),
      componentProps: {
        options: notificationRuleChannelOptionList,
        placeholder: $t('ui.placeholder.select'),
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<NotificationRule> = {
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
        try {
          return await fetchListNotificationRules(
            new PaginationQuery({
              paging: { page: page.currentPage, pageSize: page.pageSize },
              // 枚举字段只从下拉取完整值，不给模糊输入框（仓库铁律：ID/枚举不进 contains 搜索）
              formValues: {
                channel: values.channel,
                eventType: values.eventType,
              },
            }),
          );
        } catch (error: any) {
          // 不吞错：原始错误对象进控制台，用户那句翻译不包含服务端的原因
          console.error('[notification-rule] list failed', error);
          notification.error({
            message: error?.message || $t('page.notificationRule.fetchFailed'),
          });
          return { items: [], total: 0 };
        }
      },
    },
  },
  columns: [
    {
      title: $t('page.notificationRule.eventType'),
      field: 'eventType',
      slots: { default: 'eventType' },
      minWidth: 140,
    },
    {
      title: $t('page.notificationRule.channel'),
      field: 'channel',
      slots: { default: 'channel' },
      width: 110,
    },
    {
      title: $t('page.notificationRule.isAsync'),
      field: 'isAsync',
      slots: { default: 'isAsync' },
      width: 110,
    },
    {
      title: $t('page.notificationRule.isEnabled'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      width: 100,
    },
    {
      title: $t('page.notificationRule.remark'),
      field: 'remark',
      minWidth: 220,
      showOverflow: 'tooltip',
      formatter: ({ cellValue }) => cellValue || '-',
    },
    {
      title: $t('page.notificationRule.updatedAt'),
      field: 'updatedAt',
      formatter: 'formatDateTime',
      width: 170,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 230,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// 业务事件四个取值，中文标签输不全，下拉开搜索按标签过滤
function filterSelectOption(input: string, option: any) {
  return String(option?.label ?? '')
    .toLowerCase()
    .includes(input.toLowerCase());
}

// ============ 创建/编辑 ============
const editOpen = ref(false);
const editMode = ref<'create' | 'edit'>('create');
const editSaving = ref(false);
const editingId = ref<number>();

const form = reactive<{
  channel?: RuleChannel;
  eventType?: RuleEventType;
  isAsync: boolean;
  isEnabled: boolean;
  remark: string;
}>({
  channel: 'EMAIL',
  eventType: undefined,
  isAsync: true,
  isEnabled: true,
  remark: '',
});

function resetForm() {
  form.eventType = undefined;
  form.channel = 'EMAIL';
  form.isAsync = true;
  form.isEnabled = true;
  form.remark = '';
}

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  resetForm();
  editOpen.value = true;
}

function openEdit(row: NotificationRule) {
  editMode.value = 'edit';
  editingId.value = row.id;
  form.eventType = row.eventType;
  form.channel = row.channel;
  form.isAsync = !!row.isAsync;
  form.isEnabled = !!row.isEnabled;
  form.remark = row.remark || '';
  editOpen.value = true;
}

async function handleSave() {
  if (!form.eventType) {
    message.error($t('page.notificationRule.requiredEventType'));
    return;
  }
  if (!form.channel) {
    message.error($t('page.notificationRule.requiredChannel'));
    return;
  }

  editSaving.value = true;
  try {
    // CRUD 的请求体必须包 data；掩码固定四列，eventType 不在其中（见 NOTIFICATION_RULE_UPDATE_MASK）
    const data = {
      channel: form.channel,
      eventType: form.eventType,
      isAsync: form.isAsync,
      isEnabled: form.isEnabled,
      remark: form.remark,
    };
    if (editMode.value === 'create') {
      await createRule({ data });
      message.success($t('page.notificationRule.createSuccess'));
    } else if (editingId.value) {
      await updateRule({
        data,
        id: editingId.value,
        updateMask: NOTIFICATION_RULE_UPDATE_MASK,
      });
      message.success($t('page.notificationRule.updateSuccess'));
    }
    editOpen.value = false;
    await gridApi.reload();
  } catch (error: any) {
    console.error('[notification-rule] save failed', error);
    message.error(error?.message || $t('page.notificationRule.saveFailed'));
  } finally {
    editSaving.value = false;
  }
}

// ============ 删除 ============
async function handleDelete(row: NotificationRule) {
  try {
    await deleteRule({ id: row.id });
    notification.success({ message: $t('page.notificationRule.deleteSuccess') });
    await gridApi.reload();
  } catch (error: any) {
    console.error('[notification-rule] delete failed', error);
    notification.error({
      message: error?.message || $t('page.notificationRule.deleteFailed'),
    });
  }
}

// ============ 测试投递 ============
const testOpen = ref(false);
const testSaving = ref(false);
const testRule = ref<NotificationRule>();
const testTargetValue = ref('');

function openTestDispatch(row: NotificationRule) {
  testRule.value = row;
  testTargetValue.value = '';
  testOpen.value = true;
}

async function handleTestDispatch() {
  const id = testRule.value?.id;
  if (!id) return;

  testSaving.value = true;
  try {
    // 这个 RPC 是 body:"*" 的自定义路由，请求体扁平：{ id, target }。
    // 照 CRUD 那样包一层 { data: {...} } 会被 protojson 当未知字段丢掉——
    // 接口照样 200，服务端收到的却是空 id，报出来的错看着像前端没传 ID。
    const resp = await testDispatch({
      id,
      target: testTargetValue.value || undefined,
    });
    // 异步规则回的是 SENDING：这次只是「已交给队列」，说「已投递」会把它读成已经送达
    notification.success({
      message:
        resp.status === 'SENDING'
          ? $t('page.notificationRule.testDispatchQueued', {
              id: resp.deliveryId,
            })
          : $t('page.notificationRule.testDispatchSent', {
              id: resp.deliveryId,
            }),
    });
    testOpen.value = false;
  } catch (error: any) {
    console.error('[notification-rule] test dispatch failed', error);
    notification.error({
      message:
        error?.message || $t('page.notificationRule.testDispatchFailed'),
    });
  } finally {
    testSaving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('page.notificationRule.moduleName')">
      <template #toolbar-tools>
        <a-button type="primary" class="mr-2" @click="openCreate">
          {{ $t('page.notificationRule.create') }}
        </a-button>
      </template>
      <template #eventType="{ row }">
        {{ notificationRuleEventTypeToName(row.eventType) || '-' }}
      </template>
      <template #channel="{ row }">
        <a-tag v-if="row.channel" :color="notificationRuleChannelToColor(row.channel)">
          {{ notificationRuleChannelToName(row.channel) }}
        </a-tag>
        <template v-else>-</template>
      </template>
      <template #isAsync="{ row }">
        <a-tag :color="notificationRuleAsyncToColor(row.isAsync)">
          {{ notificationRuleAsyncToName(row.isAsync) }}
        </a-tag>
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="notificationRuleEnabledToColor(row.isEnabled)">
          {{ notificationRuleEnabledToName(row.isEnabled) }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button type="link" size="small" @click.stop="openEdit(row)">
          {{ $t('page.notificationRule.edit') }}
        </a-button>
        <!-- 站内信规则不给测试按钮：它需要一条真实的消息本体，填个收件人 ID 测不出东西 -->
        <a-tooltip
          v-if="row.channel !== 'INTERNAL'"
          :title="$t('page.notificationRule.testDispatchHint')"
        >
          <a-button type="link" size="small" @click.stop="openTestDispatch(row)">
            {{ $t('page.notificationRule.testDispatch') }}
          </a-button>
        </a-tooltip>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('page.notificationRule.deleteConfirm')"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small">
            {{ $t('page.notificationRule.delete') }}
          </a-button>
        </a-popconfirm>
      </template>
    </Grid>

    <!-- 创建/编辑 -->
    <a-modal
      v-model:open="editOpen"
      :title="
        editMode === 'create'
          ? $t('page.notificationRule.create')
          : $t('page.notificationRule.edit')
      "
      :confirm-loading="editSaving"
      :mask-closable="false"
      destroy-on-close
      @ok="handleSave"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item :label="$t('page.notificationRule.eventType')" required>
          <!-- 编辑态禁改：改事件类型等于换一条路由，删旧建新才说得清 -->
          <a-select
            v-model:value="form.eventType"
            :disabled="editMode === 'edit'"
            :filter-option="filterSelectOption"
            :placeholder="$t('ui.placeholder.select')"
            show-search
          >
            <a-select-option
              v-for="item in notificationRuleEventTypeList"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :label="$t('page.notificationRule.channel')" required>
          <a-select
            v-model:value="form.channel"
            :placeholder="$t('ui.placeholder.select')"
          >
            <a-select-option
              v-for="item in notificationRuleChannelOptionList"
              :key="item.value"
              :value="item.value"
            >
              {{ item.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item
          :label="$t('page.notificationRule.isAsync')"
          :help="$t('page.notificationRule.isAsyncHint')"
        >
          <a-switch v-model:checked="form.isAsync" />
        </a-form-item>
        <a-form-item
          :label="$t('page.notificationRule.isEnabled')"
          :help="$t('page.notificationRule.isEnabledHint')"
        >
          <a-switch v-model:checked="form.isEnabled" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationRule.remark')">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 测试投递 -->
    <a-modal
      v-model:open="testOpen"
      :title="
        $t('page.notificationRule.testDispatchTitle', {
          event: notificationRuleEventTypeToName(testRule?.eventType),
        })
      "
      :confirm-loading="testSaving"
      destroy-on-close
      @ok="handleTestDispatch"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item
          :label="$t('page.notificationRule.testTarget')"
          :help="$t('page.notificationRule.testTargetHint')"
        >
          <a-input
            v-model:value="testTargetValue"
            :placeholder="$t('page.notificationRule.testTargetPlaceholder')"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </Page>
</template>
