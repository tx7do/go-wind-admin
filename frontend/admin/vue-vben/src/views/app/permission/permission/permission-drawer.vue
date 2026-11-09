<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { useFormCloseGuard } from '#/composables/use-form-close-guard';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  buildMenuTree,
  buildPermissionGroupTree,
  fetchListApis,
  fetchListMenus,
  fetchListPermissionGroups,
  PaginationQuery,
  statusList,
  useCreatePermission,
  useUpdatePermission,
} from '#/api';
import { deepClone, extractLeafIds, filterIds } from '#/utils';

import PermissionCheckTree from './permission-check-tree.vue';

const { mutateAsync: createPermission } = useCreatePermission();
const { mutateAsync: updatePermission } = useUpdatePermission();

const data = ref();

// 菜单/API 勾选树数据：抽屉打开时拉取，提交时用 treeData 做叶子交集/数字过滤，
// 剥离父节点（菜单目录/API 模块分组）的 key，避免父子联动下父 ID 混入提交。
const menuTreeData = ref<any[]>([]);
const menuTreeLoading = ref(false);
const apiTreeData = ref<any[]>([]);
const apiTreeLoading = ref(false);

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.permission.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.permission.moduleName') }),
);

/** 菜单树 → 勾选树节点（key=菜单 ID 字符串，title=meta.title 兜底 name） */
function toMenuCheckTree(menus: any[]): any[] {
  return (menus ?? []).map((menu) => ({
    key: menu.id,
    title: menu.meta?.title || menu.name || '',
    ...(Array.isArray(menu.children) && menu.children.length > 0
      ? { children: toMenuCheckTree(menu.children) }
      : {}),
  }));
}

/** API 列表 → 按模块分组的勾选树（父节点 key=`module_${mod}` 非法 ID，提交时被过滤） */
function toApiCheckTree(apis: any[]): any[] {
  const moduleMap = new Map<string, any[]>();
  for (const item of apis ?? []) {
    if (!item) continue;
    const mod = item.module || $t('page.permissionGroup.moduleOther');
    if (!moduleMap.has(mod)) {
      moduleMap.set(mod, []);
    }
    const title = [item.method, item.path].filter(Boolean).join(' ');
    moduleMap.get(mod)!.push({
      key: item.id,
      title: item.description ? `${title} - ${item.description}` : title,
    });
  }
  return [...moduleMap.entries()].map(([mod, children]) => ({
    key: `module_${mod}`,
    title: mod,
    children,
  }));
}

async function loadMenuTree() {
  menuTreeLoading.value = true;
  try {
    const res = await fetchListMenus(
      new PaginationQuery({ formValues: { status: 'ON' } }),
    );
    menuTreeData.value = toMenuCheckTree(buildMenuTree(res.items ?? []));
  } catch (error) {
    console.error('load menu tree failed', error);
    menuTreeData.value = [];
  } finally {
    menuTreeLoading.value = false;
  }
}

async function loadApiTree() {
  apiTreeLoading.value = true;
  try {
    const res = await fetchListApis(new PaginationQuery({}));
    apiTreeData.value = toApiCheckTree(res.items ?? []);
  } catch (error) {
    console.error('load api tree failed', error);
    apiTreeData.value = [];
  } finally {
    apiTreeLoading.value = false;
  }
}

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
      component: PermissionCheckTree,
      fieldName: 'menuIds',
      // hideLabel 脱离空标签列：段头（组件 title）顶格靠左、树盒通栏
      hideLabel: true,
      componentProps: () => ({
        title: $t('page.permission.menuIds'),
        emptyText: $t('page.permission.menuIdsPlaceholder'),
        treeData: menuTreeData.value,
        loading: menuTreeLoading.value,
      }),
    },
    {
      component: PermissionCheckTree,
      fieldName: 'apiIds',
      hideLabel: true,
      componentProps: () => ({
        title: $t('page.permission.apiIds'),
        emptyText: $t('page.permission.apiIdsPlaceholder'),
        treeData: apiTreeData.value,
        loading: apiTreeLoading.value,
      }),
    },
  ],
});

// 关闭守卫：遮罩/ESC/X/取消统一，脏数据先确认（docs/form_drawer_close_guard.md）
const closeGuard = useFormCloseGuard(baseFormApi);

const [Drawer, drawerApi] = useVbenDrawer({
  onBeforeClose: () => closeGuard.onBeforeClose(),
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

    // 菜单树父子联动会自动勾选父菜单，其父节点 key 会混入 checkedKeys，
    // 用叶子交集剥离；API 树父节点 key 是 `module_${mod}` 非法 ID 字符串，
    // filterIds 即可剥离。两字段始终随表单提交（含清空场景）。
    finalValues.menuIds = extractLeafIds(
      Array.isArray(finalValues.menuIds) ? finalValues.menuIds : [],
      menuTreeData.value,
    );
    finalValues.apiIds = filterIds(
      Array.isArray(finalValues.apiIds) ? finalValues.apiIds : [],
    );

    try {
      await (data.value?.create
        ? createPermission(finalValues)
        : updatePermission({ id: data.value.row.id, values: finalValues }));

      notification.success({
        message: data.value?.create
          ? $t('ui.notification.create_success')
          : $t('ui.notification.update_success'),
      });
      closeGuard.markClean();
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

  async onOpenChange(isOpen) {
    if (isOpen) {
      // 获取传入的数据
      data.value = drawerApi.getData<Record<string, any>>();

      // 为表单赋值
      await baseFormApi.setValues(data.value?.row);
      await closeGuard.syncBaseline();

      // 加载两棵勾选树（对齐 react 基准：打开抽屉时拉取）
      loadMenuTree();
      loadApiTree();

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
