<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '@/adapter/form';
import {
  tenantAuditStatusList,
  tenantStatusList,
  tenantTypeList,
  useTenantStore,
  useUserListStore,
} from '@/stores';

const tenantStore = useTenantStore();
const userListStore = useUserListStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: t('pages.tenant.moduleName') })
    : $t('ui.modal.update', { moduleName: t('pages.tenant.moduleName') }),
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
      label: t('pages.tenant.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: t('pages.tenant.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Select',
      fieldName: 'type',
      label: t('pages.tenant.type'),
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
      label: t('pages.tenant.auditStatus'),
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
          default: () => t('pages.tenant.adminSetting'),
        };
      },
    },

    {
      component: 'Input',
      fieldName: 'user.username',
      label: t('pages.tenant.adminUserName'),
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
      label: t('pages.tenant.adminPassword'),
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
      label: t('pages.tenant.adminPasswordConfirm'),
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
      label: t('pages.tenant.adminMobile'),
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
      label: t('pages.tenant.adminEmail'),
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
      message: t('pages.notification.password_mismatch'),
    });
    setLoading(false);
    return;
  }

  // 检查租户编码是否存在
  try {
    await tenantStore.tenantExists(values.code, values.name);
  } catch {
    notification.error({
      message: t('pages.tenant.tenant_code_exists'),
    });
    setLoading(false);
    return;
  }

  // 检查用户名是否存在
  try {
    await userListStore.userExists(values.user.username);
  } catch {
    notification.error({
      message: t('pages.tenant.notification.user_username_exists'),
    });
    setLoading(false);
    return;
  }

  try {
    await tenantStore.createTenantWithAdminUser({
      tenant: {
        name: values.name,
        code: values.code,
        type: values.type,
        auditStatus: values.auditStatus,
        status: values.status,
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
    await tenantStore.updateTenant(data.value.row.id, values);

    notification.success({
      message: $t('ui.notification.update_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_failed'),
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
  </Drawer>
</template>
