<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  buildMenuTree,
  buildPermissionGroupTree,
  convertApiToTree,
  fetchListApis,
  fetchListMenus,
  fetchListPermissionGroups,
  PaginationQuery,
  statusList,
  useCreatePermission,
  useUpdatePermission,
} from '#/api';
import { deepClone, extractLeafIds } from '#/utils';

const { mutateAsync: createPermission } = useCreatePermission();
const { mutateAsync: updatePermission } = useUpdatePermission();

const data = ref();

// 保存两个树当前的 treeData，供提交时用 extractLeafIds 做叶子交集过滤，
// 剥离父节点（菜单目录/API 模块分组）的 ID，避免父子联动下父 ID 混入提交。
const menuTreeData = ref<any[]>([]);
const apiTreeData = ref<any[]>([]);

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.permission.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.permission.moduleName') }),
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
      label: $t('page.permission.name'),
      rules: 'required',
      componentProps() {
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
        };
      },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.permission.code'),
      rules: 'required',
      componentProps() {
        return {
          placeholder: $t('ui.placeholder.input'),
          allowClear: true,
        };
      },
    },
    {
      component: 'ApiTreeSelect',
      fieldName: 'groupId',
      label: $t('page.permission.groupId'),
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        class: 'w-full',
        showSearch: true,
        treeDefaultExpandAll: true,
        numberToString: false,
        allowClear: true,
        childrenField: 'children',
        labelField: 'name',
        valueField: 'id',
        treeNodeFilterProp: 'label',
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: any) => {
          return buildPermissionGroupTree(data);
        },
        api: async () => {
          const fieldValue = baseFormApi.form.values;
          const result = await fetchListPermissionGroups(
            new PaginationQuery({
              formValues: {
                parentId: fieldValue.groupId,
                status: 'ON',
              },
            }),
          );
          return result.items;
        },
      },
    },
    {
      component: 'RadioGroup',
      fieldName: 'status',
      defaultValue: 'ON',
      label: $t('ui.table.status'),
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: statusList,
      },
    },
    {
      component: 'ApiTree',
      fieldName: 'menuIds',
      componentProps: {
        title: $t('page.permission.menuIds'),
        showSearch: true,
        treeDefaultExpandAll: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'meta.title',
        valueField: 'id',
        resultField: 'items',
        api: async () => {
          return await fetchListMenus(
            new PaginationQuery({
              formValues: { status: 'ON' },
            }),
          );
        },
        afterFetch: (data: any) => {
          const tree = buildMenuTree(data.items);
          // 保存当前 treeData，供提交时用 extractLeafIds 做叶子交集过滤
          menuTreeData.value = tree;
          return tree;
        },
      },
    },
    {
      component: 'ApiTree',
      fieldName: 'apiIds',
      componentProps: {
        title: $t('page.permission.apiIds'),
        toolbar: true,
        search: true,
        checkable: true,
        numberToString: false,
        loadingSlot: 'suffixIcon',
        childrenField: 'children',
        labelField: 'title',
        valueField: 'key',
        api: async () => {
          const data = await fetchListApis(new PaginationQuery({}));
          const tree = convertApiToTree(data.items ?? []);
          // 保存当前 treeData，供提交时用 extractLeafIds 做叶子交集过滤
          apiTreeData.value = tree;
          return tree;
        },
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
      finalValues.apiIds !== null &&
      Array.isArray(finalValues.apiIds) &&
      finalValues.apiIds.length > 0
    ) {
      // 用叶子交集剥离父节点（API 模块分组）ID：父子联动下勾选父分组下全部子 API
      // 会使父分组被自动勾选，其 key（分组 ID，数字）混入 checkedKeys，
      // filterNumbers 无法过滤数字父 key，会被后端当作 API ID 处理。
      finalValues.apiIds = extractLeafIds(values.apiIds, apiTreeData.value);
    }

    if (
      finalValues.menuIds !== null &&
      Array.isArray(finalValues.menuIds) &&
      finalValues.menuIds.length > 0
    ) {
      // 同上，剥离菜单目录父节点 ID
      finalValues.menuIds = extractLeafIds(values.menuIds, menuTreeData.value);
    }


    try {
      await (data.value?.create
        ? createPermission(finalValues)
        : updatePermission({ id: data.value.row.id, values: finalValues }));

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

      // 为表单赋值
      baseFormApi.setValues(data.value?.row);

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
