<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import {
  type identityservicev1_OrgUnit as OrgUnit,
  type identityservicev1_Position as Position,
} from '#/generated/api/admin/service/v1';
import {
  genderList,
  useOrgUnitStore,
  usePositionStore,
  useRoleStore,
  userStatusList,
  useUserListStore,
} from '#/stores';
import { useUserViewStore } from '#/views/app/opm/user/user-view.state';

const userListStore = useUserListStore();
const roleStore = useRoleStore();
const orgUnitStore = useOrgUnitStore();
const positionStore = usePositionStore();
const userViewStore = useUserViewStore();

const data = ref();

const orgUnitList = ref<OrgUnit[]>([]);
const positionList = ref<Position[]>([]);

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.user.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.user.moduleName') }),
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
      fieldName: 'username',
      label: $t('page.user.table.username'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: z.string().min(1, { message: $t('ui.formRules.required') }),
      dependencies: {
        disabled: () => !data.value?.create,
        triggerFields: ['username'],
      },
    },
    {
      component: 'VbenInputPassword',
      fieldName: 'password',
      label: $t('page.user.table.password'),
      componentProps: {
        passwordStrength: true,
        placeholder: $t('ui.placeholder.input'),
      },
      // rules: 'required',
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'roleIds',
      label: $t('page.user.form.role'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        showSearch: true,
        multiple: true,
        treeDefaultExpandAll: false,
        allowClear: true,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result = await roleStore.listRole(undefined, {
            // parent_id: 0,
            status: 'ON',
            tenant_id: userViewStore.currentTenantId ?? 0,
            type__not: 'TEMPLATE',
          });

          return result.items;
        },
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'orgUnitIds',
      label: $t('page.user.form.orgUnit'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        numberToString: true,
        showSearch: true,
        multiple: true,
        treeDefaultExpandAll: true,
        allowClear: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        api: async () => {
          const result = await orgUnitStore.listOrgUnit(undefined, {
            status: 'ON',
            tenant_id: userViewStore.currentTenantId ?? 0,
          });
          orgUnitList.value = result.items ?? [];
          return result.items;
        },
        onChange: async (orgUnitId: any) => {
          console.log('org onChange:', orgUnitId);

          if (!orgUnitId) {
            await baseFormApi.setValues(
              {
                orgUnitId: undefined,
                positionId: undefined,
              },
              false,
            );
          }
        },
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'positionIds',
      label: $t('page.user.form.position'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        showSearch: true,
        allowClear: true,
        multiple: true,
        api: async () => {
          const result = await positionStore.listPosition(undefined, {
            status: 'ON',
          });
          positionList.value = result.items ?? [];
          return result.items;
        },
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: { name: string; path: string }[]) => {
          return data.map((item: Position) => ({
            label: item.name,
            value: item.id,
          }));
        },
      },
    },

    {
      component: 'Select',
      fieldName: 'gender',
      label: $t('page.user.table.gender'),
      defaultValue: 'SECRET',
      componentProps: {
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        allowClear: true,
        showSearch: true,
        options: genderList,
        placeholder: $t('ui.placeholder.select'),
      },
    },

    {
      component: 'Input',
      fieldName: 'nickname',
      label: $t('page.user.table.nickname'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'realname',
      label: $t('page.user.table.realname'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'email',
      label: $t('page.user.table.email'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'mobile',
      label: $t('page.user.table.mobile'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },

    {
      component: 'RadioGroup',
      fieldName: 'status',
      label: $t('ui.table.status'),
      defaultValue: 'NORMAL',
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: userStatusList,
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

    // 加载条设置为加载状态
    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();

    console.log(getTitle.value, Object.keys(values));

    try {
      await (data.value?.create
        ? userListStore.createUser(values)
        : userListStore.updateUser(data.value.row.id, values));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });
    } catch {
      notification.error({
        message: data.value?.create
          ? $t('ui.notification.create_failed')
          : $t('ui.notification.update_failed'),
      });
    } finally {
      // 关闭窗口
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen: boolean) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值
      if (data.value.row !== undefined) {
        if (data.value?.row?.orgUnitId !== undefined) {
          data.value.row.orgUnitId = data.value?.row?.orgUnitId.toString();
        }
        baseFormApi.setValues(data.value?.row);
      }

      setLoading(false);

      console.log('onOpenChange', data.value, data.value?.create);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ confirmLoading: loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
