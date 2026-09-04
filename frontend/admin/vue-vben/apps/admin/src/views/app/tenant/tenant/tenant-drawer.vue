<script lang="ts" setup>
import { computed, ref } from 'vue';

import dayjs from 'dayjs';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import {
  Button,
  Descriptions,
  DescriptionsItem,
  Popconfirm,
  Progress,
  notification,
} from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  fetchListPlans,
  fetchTenantUsage,
  PaginationQuery,
  tenantAuditStatusList,
  tenantStatusList,
  tenantTypeList,
  useCreateTenantWithAdminUser,
  useTenantExists,
  useUpdateTenant,
  useUserExists,
  useCleanupTenantData,
} from '#/api';

const { mutateAsync: doCreateTenant } = useCreateTenantWithAdminUser();
const { mutateAsync: doUpdateTenant } = useUpdateTenant();
const { mutateAsync: tenantExists } = useTenantExists();
const { mutateAsync: userExists } = useUserExists();
const { mutateAsync: cleanupMut } = useCleanupTenantData();

const data = ref();
const usageData = ref<any>(null);
const cleanupLoading = ref(false);

// 将日期组件的值（dayjs/Date/字符串）归一化为 protobuf Timestamp 可解析的 RFC3339 字符串。
// 后端 protojson 要求 Timestamp 必须是 RFC3339，直接传 "2026-09-30 00:00:00" 等形式会报 CODEC。
function toProtoTimestamp(v?: unknown): string | undefined {
  if (v === undefined || v === null || v === '') return undefined;
  const d = dayjs(v as any);
  return d.isValid() ? d.toISOString() : String(v);
}

// 与后端等保口令策略一致：≥8 位，小写/大写/数字/符号四类取三。
function isValidPassword(pwd: string): boolean {
  if (typeof pwd !== 'string' || pwd.length < 8) return false;
  let classes = 0;
  if (/[a-z]/.test(pwd)) classes++;
  if (/[A-Z]/.test(pwd)) classes++;
  if (/\d/.test(pwd)) classes++;
  if (/[^A-Za-z0-9]/.test(pwd)) classes++;
  return classes >= 3;
}

// 辅助函数：根据配额类型返回标签/当前值/百分比。
// eslint-disable-next-line @typescript-messages/no-explicit-any
function getQuotaLabel(qt: any): string {
  if (qt === 1) return $t('page.tenant.usageUserCount');
  if (qt === 2) return $t('page.tenant.usageStorage');
  if (qt === 3) return $t('page.tenant.usageApiCalls');
  return $t('page.tenant.quotaType');
}
// eslint-disable-next-line @typescript-messages/no-explicit-any
function getQuotaCurrent(qt: any): number {
  const ud = usageData.value as any;
  if (!ud) return 0;
  if (qt === 1) return ud.userCount ?? 0;
  if (qt === 2) return ud.storageUsedBytes ?? 0;
  if (qt === 3) return ud.apiCallCount ?? 0;
  return 0;
}
// eslint-disable-next-line @typescript-messages/no-explicit-any
function getQuotaPercent(qt: any, limit: number): number {
  const current = getQuotaCurrent(qt);
  if (limit <= 0) return 0;
  return Math.min(100, (current / limit) * 100);
}

async function handleCleanup() {
  // eslint-disable-next-line @typescript-messages/no-explicit-any
  const rowId = (data.value?.row as any)?.id;
  if (!rowId) return;
  try {
    cleanupLoading.value = true;
    await cleanupMut({ id: rowId });
    notification.success({ message: $t('page.tenant.cleanupSuccess') });
    drawerApi.close();
  } catch (err: any) {
    notification.error({
      message: err?.message || $t('page.tenant.cleanupFailed'),
    });
  } finally {
    cleanupLoading.value = false;
  }
}

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.tenant.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.tenant.moduleName') }),
);
// const isCreate = computed(() => data.value?.create);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  // 所有表单项共用，可单独在表单内覆盖
  commonConfig: {
    // 所有表单项
    componentProps: {
      class: 'w-full',
    },
  },
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.tenant.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.tenant.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: $t('page.tenant.type'),
      defaultValue: 'PAID',
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: tenantTypeList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'Select',
      fieldName: 'auditStatus',
      label: $t('page.tenant.auditStatus'),
      defaultValue: 'APPROVED',
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        options: tenantAuditStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
      rules: 'selectRequired',
    },
    {
      component: 'Select',
      fieldName: 'status',
      defaultValue: 'ON',
      label: $t('ui.table.status'),
      rules: 'selectRequired',
      componentProps: {
        options: tenantStatusList,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
      },
    },
    {
      component: 'Textarea',
      fieldName: 'remark',
      label: $t('ui.table.remark'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'ApiSelect',
      fieldName: 'subscriptionPlan',
      label: $t('page.tenant.subscriptionPlan'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        showSearch: true,
        allowClear: true,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        labelField: 'name',
        valueField: 'id',
        api: async () => {
          const result = await fetchListPlans(
            new PaginationQuery({
              formValues: {},
            }),
          );
          return result.items;
        },
      },
    },

    {
      component: 'DatePicker',
      fieldName: 'expiredAt',
      label: $t('page.tenant.expiredAt'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
      },
    },

    {
      component: 'Divider',
      fieldName: 'divider1',
      hideLabel: true,
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
      renderComponentContent() {
        return {
          default: () => $t('page.tenant.adminSetting'),
        };
      },
    },

    {
      component: 'Input',
      fieldName: 'user.username',
      label: $t('page.tenant.adminUserName'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'VbenInputPassword',
      fieldName: 'password',
      label: $t('page.tenant.adminPassword'),
      rules: 'required',
      componentProps: {
        passwordStrength: true,
        placeholder: $t('ui.placeholder.input'),
      },
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'VbenInputPassword',
      fieldName: 'passwordConfirm',
      label: $t('page.tenant.adminPasswordConfirm'),
      rules: 'required',
      componentProps: {
        passwordStrength: true,
        placeholder: $t('ui.placeholder.input'),
      },
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'Input',
      fieldName: 'user.mobile',
      label: $t('page.tenant.adminMobile'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
    },

    {
      component: 'Input',
      fieldName: 'user.email',
      label: $t('page.tenant.adminEmail'),
      rules: 'required',
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      dependencies: {
        show: (_values) => {
          return data.value?.create;
        },
        triggerFields: ['type'],
      },
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    console.log('onConfirm');

    // 校验输入的数据
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    // 创建模式下，密码需满足后端等保复杂度，提前拦截给出可操作提示
    if (data.value?.create && !isValidPassword(values.password)) {
      notification.error({
        message: $t('page.tenant.passwordComplexity'),
      });
      setLoading(false);
      return;
    }

    console.log(getTitle.value, values);

    await (data.value?.create
      ? createTenantWithAdminUser(values)
      : updateTenant(values));
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值
      baseFormApi.setValues(data.value?.row);

      // 编辑模式下加载该租户的用量与配额对比数据
      // eslint-disable-next-line @typescript-messages/no-explicit-any
      const rowId = (data.value?.row as any)?.id;
      if (!data.value?.create && rowId) {
        fetchTenantUsage({ id: rowId } as any)
          .then((res: any) => {
            usageData.value = res ?? null;
          })
          .catch(() => {
            usageData.value = null;
          });
      } else {
        usageData.value = null;
      }

      setLoading(false);

      console.log('onOpenChange', data.value, data.value?.create);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}

// async function createTenant(values: any) {
//   console.log('createTenant', values);
//
//   try {
//     await tenantStore.createTenant(values);
//
//     notification.success({
//       message: $t('ui.notification.create_success'),
//     });
//   } catch {
//     notification.error({
//       message: $t('ui.notification.create_failed'),
//     });
//   } finally {
//     // 关闭窗口
//     drawerApi.close();
//     setLoading(false);
//   }
// }

async function createTenantWithAdminUser(values: any) {
  console.log('createTenantWithAdminUser', values);

  // 检查密码和确认密码是否一致
  if (values.password !== values.passwordConfirm) {
    notification.error({
      message: $t('page.notification.password_mismatch'),
    });
    setLoading(false);
    return;
  }

  // 检查租户编码是否存在
  try {
    await tenantExists({ code: values.code, name: values.name });
  } catch {
    notification.error({
      message: $t('page.tenant.tenant_code_exists'),
    });
    setLoading(false);
    return;
  }

  // 检查用户名是否存在
  try {
    await userExists({ username: values.user.username });
  } catch {
    notification.error({
      message: $t('page.tenant.notification.user_username_exists'),
    });
    setLoading(false);
    return;
  }

  try {
    await doCreateTenant({
      tenant: {
        name: values.name,
        code: values.code,
        type: values.type,
        auditStatus: values.auditStatus,
        status: values.status,
        planId: values.subscriptionPlan,
        expiredAt: toProtoTimestamp(values.expiredAt),
        remark: values.remark,
      },
      user: values.user,
      password: values.password,
    });

    notification.success({
      message: $t('ui.notification.create_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.create_failed'),
    });
  } finally {
    // 关闭窗口
    drawerApi.close();
    setLoading(false);
  }
}

async function updateTenant(values: any) {
  console.log('updateTenant', values);

  try {
    // 仅透传有效的 Tenant 字段，剔除 divider1/user/password 等纯 UI 或子对象字段，
    // 避免其进入 updateMask 导致后端 field mask 校验失败
    await doUpdateTenant({
      id: data.value.row.id,
      values: {
        name: values.name,
        code: values.code,
        type: values.type,
        auditStatus: values.auditStatus,
        status: values.status,
        remark: values.remark,
        subscriptionPlan: values.subscriptionPlan,
        expiredAt: toProtoTimestamp(values.expiredAt),
      },
    });

    notification.success({
      message: $t('ui.notification.update_success'),
    });
  } catch (err: any) {
    notification.error({
      message: err?.message || $t('ui.notification.update_failed'),
    });
  } finally {
    // 关闭窗口
    drawerApi.close();
    setLoading(false);
  }
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
    <template v-if="!data?.create && usageData">
      <div class="usage-divider">{{ $t('page.tenant.usageSection') }}</div>
      <Descriptions :column="1" size="small" bordered>
        <DescriptionsItem :label="$t('page.tenant.usageUserCount')">
          {{ usageData.userCount ?? 0 }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('page.tenant.usageStorage')">
          {{ usageData.storageUsedBytes ?? 0 }} bytes
        </DescriptionsItem>
        <DescriptionsItem :label="$t('page.tenant.usageApiCalls')">
          {{ usageData.apiCallCount ?? 0 }}
        </DescriptionsItem>
        <DescriptionsItem :label="$t('page.tenant.usagePlan')">
          {{ usageData.planName ?? $t('page.tenant.usageNoPlan') }}
        </DescriptionsItem>
      </Descriptions>
      <div v-if="usageData.quotas && usageData.quotas.length" class="quota-list">
        <div v-for="(q, idx) in usageData.quotas" :key="idx" class="quota-item">
          <div class="quota-label">
            {{ getQuotaLabel(q.quotaType) }}: {{ getQuotaCurrent(q.quotaType) }} /
            {{ q.quotaValue ?? 0 }}
          </div>
          <Progress
            :percent="getQuotaPercent(q.quotaType, q.quotaValue ?? 0)"
            size="small"
          />
        </div>
      </div>
      <div class="cleanup-btn-wrapper">
        <Popconfirm
          :title="$t('page.tenant.cleanupConfirmTitle')"
          :content="$t('page.tenant.cleanupConfirmDesc')"
          ok-type="danger"
          @confirm="handleCleanup"
        >
          <Button danger :loading="cleanupLoading">
            {{ $t('page.tenant.cleanupData') }}
          </Button>
        </Popconfirm>
      </div>
    </template>
  </Drawer>
</template>

<style scoped>
.usage-divider {
  margin: 24px 0 16px;
  font-weight: 600;
  font-size: 14px;
}
.quota-list {
  margin-top: 16px;
}
.quota-item {
  margin-bottom: 12px;
}
.quota-label {
  margin-bottom: 4px;
  font-size: 12px;
}
.cleanup-btn-wrapper {
  margin-top: 24px;
}
</style>
