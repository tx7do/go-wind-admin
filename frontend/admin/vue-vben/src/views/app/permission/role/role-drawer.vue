<script lang="ts" setup>
import { computed, h, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import AiGenerateButton from '#/components/AiGenerateButton.vue';
import { type permissionservicev1_PermissionGroup as PermissionGroup } from '#/api';
import {
  buildPermissionTree,
  fetchListOrgUnits,
  fetchListPermissionGroups,
  fetchListPermissions,
  PaginationQuery,
  roleDataScopeList,
  statusList,
  useCreateRole,
  useUpdateRole,
} from '#/api';
import { deepClone, filterNumbers } from '#/utils';

const { mutateAsync: createRole } = useCreateRole();
const { mutateAsync: updateRole } = useUpdateRole();

const data = ref();
const groups = ref<PermissionGroup[]>([]);

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.role.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.role.moduleName') }),
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
      label: $t('page.role.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.role.code'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'InputNumber',
      fieldName: 'sortOrder',
      defaultValue: 1,
      label: $t('ui.table.sortOrder'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'RadioGroup',
      fieldName: 'status',
      label: $t('ui.table.status'),
      defaultValue: 'ON',
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: statusList,
      },
    },
    {
      component: 'Select',
      fieldName: 'dataScope',
      label: $t('page.role.dataScope'),
      defaultValue: 'ALL',
      rules: 'selectRequired',
      componentProps: {
        options: roleDataScopeList,
        placeholder: $t('ui.placeholder.select'),
      },
    },
    {
      component: 'ApiTree',
      fieldName: 'orgUnits',
      dependencies: {
        // 仅 SELECTED_UNITS 档展示自定义授权单元集配置
        show: (values) => values.dataScope === 'SELECTED_UNITS',
        triggerFields: ['dataScope'],
      },
      componentProps: {
        title: $t('page.role.orgUnits'),
        treeDefaultExpandAll: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        api: async () => {
          const result = await fetchListOrgUnits(
            new PaginationQuery({
              formValues: { status: 'ON' },
            }),
          );
          return result.items;
        },
      },
    },
    {
      component: 'Textarea',
      fieldName: 'description',
      label: $t('ui.table.description'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      // AI 内容生成（表单助手）：生成结果回填 description
      suffix: () =>
        h(
          'div',
          // shrink-0：flex 行内不被 textarea 挤压换行；self-start 顶部对齐
          { class: 'ml-1 shrink-0 self-start pt-1' },
          h(AiGenerateButton, {
            scene: 'DESCRIPTION',
            onGenerate: (content: string) =>
              baseFormApi.setFieldValue('description', content),
          }),
        ),
    },
    {
      component: 'ApiTree',
      fieldName: 'permissions',
      componentProps: {
        title: $t('page.role.permissions'),
        showSearch: true,
        treeDefaultExpandAll: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'title',
        valueField: 'key',
        resultField: 'items',
        api: async () => {
          const groupData = await fetchListPermissionGroups(
            new PaginationQuery({
              formValues: { status: 'ON' },
            }),
          );
          groups.value = groupData.items ?? [];

          return await fetchListPermissions(
            new PaginationQuery({
              formValues: { status: 'ON' },
            }),
          );
        },
        afterFetch: (data: any) => {
          return buildPermissionTree(groups.value, data.items);
        },
      },
    },
    {
      // 字段权限：勾选 = 对该角色用户隐藏（黑名单语义，User 资源试点）。
      // value 与后端 identity User proto 字段 json_name 逐字一致。
      component: 'CheckboxGroup',
      fieldName: 'userHiddenFields',
      defaultValue: [],
      label: $t('page.role.fieldPerm.title'),
      componentProps: {
        options: [
          { label: $t('page.role.fieldPerm.field.email'), value: 'email' },
          { label: $t('page.role.fieldPerm.field.mobile'), value: 'mobile' },
          { label: $t('page.role.fieldPerm.field.telephone'), value: 'telephone' },
          { label: $t('page.role.fieldPerm.field.address'), value: 'address' },
          { label: $t('page.role.fieldPerm.field.region'), value: 'region' },
          { label: $t('page.role.fieldPerm.field.lastLoginAt'), value: 'lastLoginAt' },
          { label: $t('page.role.fieldPerm.field.lastLoginIp'), value: 'lastLoginIp' },
        ],
      },
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {

    // 校验输入的数据
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    setLoading(true);

    // 获取表单数据
    const values = await baseFormApi.getValues();
    // @ts-ignore JSON.stringify
    const finalValues = deepClone(values);

    if (
      finalValues.permissions !== null &&
      Array.isArray(finalValues.permissions) &&
      finalValues.permissions.length > 0
    ) {
      finalValues.permissions = filterNumbers(values.permissions);
    }

    // 仅 SELECTED_UNITS 档提交授权单元集（含清空场景）；
    // 其余档位不携带该字段，后端维持既有集不替换。
    if (finalValues.dataScope === 'SELECTED_UNITS') {
      finalValues.orgUnits = Array.isArray(values.orgUnits)
        ? filterNumbers(values.orgUnits)
        : [];
    } else {
      delete finalValues.orgUnits;
    }

    // 字段权限始终随表单提交（含清空场景）：抽屉所见即保存后的最终态。
    // userHiddenFields 是表单专用中间字段，不是 Role proto 字段，提交前必须移除。
    finalValues.fieldPermissions =
      Array.isArray(finalValues.userHiddenFields) &&
      finalValues.userHiddenFields.length > 0
        ? [
            {
              resource: 'User',
              hiddenFields: finalValues.userHiddenFields.filter(
                (v: any) => typeof v === 'string',
              ),
            },
          ]
        : [];
    delete finalValues.userHiddenFields;


    try {
      await (data.value?.create
        ? createRole(finalValues)
        : updateRole({ id: data.value.row.id, values: finalValues }));

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
      drawerApi.close();
      setLoading(false);
    }
  },

  onOpenChange(isOpen) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值。注意：setValues 必须只调一次——连调两次时后一次会清掉
      // 前一次刚设置的值（vee-validate setValues 非合并语义），因此把字段权限
      // 的中间态 userHiddenFields 合并在同一次调用里。
      const row = data.value?.row;
      const fpEntry = Array.isArray(row?.fieldPermissions)
        ? row.fieldPermissions.find((e: any) => e?.resource === 'User')
        : undefined;
      baseFormApi.setValues({
        ...(row ?? {}),
        userHiddenFields: Array.isArray(fpEntry?.hiddenFields)
          ? fpEntry.hiddenFields.filter((v: any) => typeof v === 'string')
          : [],
      });

      // setLoading(true);
      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ loading });
}
</script>

<template>
  <Drawer :title="getTitle">
    <BaseForm />
  </Drawer>
</template>
