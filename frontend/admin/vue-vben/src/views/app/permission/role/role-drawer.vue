<script lang="ts" setup>
import { computed, h, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { useFormCloseGuard } from '#/composables/use-form-close-guard';
import { $t } from '@vben/locales';

import { Modal, notification } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import AiGenerateButton from '#/components/AiGenerateButton.vue';
import { type permissionservicev1_PermissionGroup as PermissionGroup } from '#/api';
import {
  buildPermissionTree,
  fetchGetRole,
  fetchListOrgUnits,
  fetchListPermissionGroups,
  fetchListPermissions,
  fetchListRoles,
  PaginationQuery,
  roleDataScopeList,
  statusList,
  useCreateRole,
  useUpdateRole,
} from '#/api';
import { deepClone, filterIds, isIdString } from '#/utils';
import PermissionCheckTree from '../permission/permission-check-tree.vue';
import RoleFieldPermCard from './role-field-perm-card.vue';

const { mutateAsync: createRole } = useCreateRole();
const { mutateAsync: updateRole } = useUpdateRole();

const data = ref();
const groups = ref<PermissionGroup[]>([]);

// 分配权限勾选树：数据与装载态由抽屉 onOpenChange 拉取（对齐权限点抽屉的
// PermissionCheckTree 用法），提交时勾选 key 数组即权限 ID。
const permTreeData = ref<any[]>([]);
const permTreeLoading = ref(false);

// ========== 权限继承（父子角色） ==========

// 父角色候选树（排除自身及后代防环，后端另有环检测兜底）
const roleTreeOptions = ref<any[]>([]);

// 父角色权限上限集（null=无父角色，不收窄）；ID 已是字符串（uint64 → proto3 JSON）
const permCapIds = ref<null | string[]>(null);

// 权限树全量数据源，父角色切换时按 cap 过滤重建
const permTreeSource = ref<{ groups: any[]; permissions: any[] } | null>(null);

/** 收集自身及全部后代角色 ID（按 parentId 嵌套） */
function collectSelfAndDescendantIds(items: any[], selfId?: string): string[] {
  if (!selfId) return [];
  const byParent = new Map<string, string[]>();
  for (const it of items || []) {
    const pid = it.parentId || '';
    if (!byParent.has(pid)) byParent.set(pid, []);
    byParent.get(pid)!.push(it.id);
  }
  const result = new Set<string>([selfId]);
  const stack = [selfId];
  while (stack.length > 0) {
    const cur = stack.pop()!;
    for (const child of byParent.get(cur) || []) {
      if (!result.has(child)) {
        result.add(child);
        stack.push(child);
      }
    }
  }
  return [...result];
}

/** 角色列表按 parentId 嵌套为 TreeSelect 数据（森林，缺父孤行落为根），排除 excludeIds */
function buildRoleTreeOptions(items: any[], excludeIds: string[]): any[] {
  const excluded = new Set(excludeIds);
  const nodes = new Map<string, any>();
  for (const it of items || []) {
    const id = it.id;
    if (excluded.has(id)) continue;
    nodes.set(id, {
      value: id,
      label: it.name || it.code || String(id),
      pid: it.parentId || '',
      children: [] as any[],
    });
  }
  const roots: any[] = [];
  for (const node of nodes.values()) {
    const parent = node.pid ? nodes.get(node.pid) : undefined;
    if (parent) {
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }
  const strip = (list: any[]): any[] =>
    list.map((n: any) => {
      const { pid: _pid, ...rest } = n;
      if (rest.children.length > 0) {
        rest.children = strip(rest.children);
      } else {
        delete rest.children;
      }
      return rest;
    });
  return strip(roots);
}

/**
 * 按父角色权限上限集（cap）过滤权限树数据源——在 buildPermissionTree 之前调用。
 * 只保留 cap 内的权限点，以及（递归）仍含至少一个保留权限点的权限组。
 * cap 为 null/undefined/空数组表示无父角色（根角色），原样返回全量。
 */
function filterPermissionSourceByCap(
  permsGroups: any[],
  permissions: any[],
  capIds: null | string[] | undefined,
): { groups: any[]; permissions: any[] } {
  if (!capIds || capIds.length === 0) return { groups: permsGroups, permissions };
  const cap = new Set(capIds);
  const kept = (permissions || []).filter((p) => cap.has(String(p.id)));
  const filterGroups = (list: any[]): any[] =>
    (list || [])
      .map((g) => ({ ...g, children: filterGroups(g.children || []) }))
      .filter(
        (g) => g.children.length > 0 || kept.some((p) => String(p.groupId) === String(g.id)),
      );
  return { groups: filterGroups(permsGroups), permissions: kept };
}

/** 按父角色上限集重建权限树；返回本次生效的 cap（拉取失败按全量展示，后端保存兜底） */
async function rebuildPermTree(parentId: null | string): Promise<null | string[]> {
  const src = permTreeSource.value;
  if (!src) return null;
  let cap: null | string[] = null;
  if (parentId) {
    try {
      const parent = await fetchGetRole({ id: parentId });
      cap = ((parent as any)?.permissions ?? []) as string[];
    } catch (error) {
      console.error('load parent role permission cap failed', error);
    }
  }
  permCapIds.value = cap;
  const filtered = filterPermissionSourceByCap(src.groups, src.permissions, cap);
  permTreeData.value =
    filtered.groups.length === 0 && filtered.permissions.length > 0
      ? filtered.permissions.map((p) => ({
          key: p.id,
          title: p.name || p.code || String(p.id),
        }))
      : buildPermissionTree(filtered.groups, filtered.permissions);
  return cap;
}

/** 切换父角色：重建权限树并把已勾权限收敛到新 cap 内 */
async function handleParentChange(pid: string) {
  const cap = await rebuildPermTree(pid || null);
  if (pid && cap) {
    const cur = await baseFormApi.getValues();
    if (Array.isArray(cur?.permissions)) {
      await baseFormApi.setFieldValue(
        'permissions',
        cur.permissions.filter((k: any) => isIdString(k) && cap!.includes(k)),
      );
    }
  }
}

/** 加载父角色候选下拉（启用角色全集，排除自身及后代） */
async function loadRoleTreeOptions() {
  try {
    const result = await fetchListRoles(
      new PaginationQuery({
        paging: { page: 1, pageSize: 1000 },
        formValues: { status: 'ON' },
      }),
    );
    roleTreeOptions.value = buildRoleTreeOptions(
      result.items || [],
      collectSelfAndDescendantIds(result.items || [], data.value?.row?.id),
    );
  } catch (error) {
    console.error('load parent role options failed', error);
    roleTreeOptions.value = [];
  }
}

async function loadPermTree() {
  permTreeLoading.value = true;
  try {
    const groupData = await fetchListPermissionGroups(
      new PaginationQuery({
        formValues: { status: 'ON' },
      }),
    );
    groups.value = groupData.items ?? [];
    const permData = await fetchListPermissions(
      new PaginationQuery({
        formValues: { status: 'ON' },
      }),
    );
    // 仅装载全量数据源，渲染树由 rebuildPermTree 按 cap 过滤生成
    permTreeSource.value = { groups: groups.value, permissions: permData.items ?? [] };
  } catch (error) {
    console.error('load permission tree failed', error);
    permTreeSource.value = null;
    permTreeData.value = [];
  } finally {
    permTreeLoading.value = false;
  }
}

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
    // react 基准抽屉标签列为 120px，这里放宽到 130 容纳「字段权限（User）」整行不折行
    labelWidth: 130,
    // 标签列内左对齐（font 尺寸三端本就同为 14px，字重交由 FormLabel 默认）
    labelClass: 'justify-start',
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
      // 保留角色码：platform: / template: 命名空间与内置租户管理员码承载登录态
      // 身份判定语义（后端按码值精确匹配填管理员标志），禁止用户自建角色占用。
      // 前端拦截只是即时反馈，后端 Create/Update 仍有一道校验兜底。
      rules: z
        .string()
        .min(1, { message: $t('ui.formRules.required') })
        .refine(
          (v) => !/^(platform:|template:)/.test(v) && v !== 'tenant:manager',
          { message: $t('page.role.reservedCode') },
        ),
    },
    {
      component: 'InputNumber',
      fieldName: 'sortOrder',
      defaultValue: 1,
      label: $t('ui.table.sortOrder'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        min: 0,
        precision: 0,
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
      // 权限继承：父角色（可选）。候选树排除自身及后代（防环），数据由
      // loadRoleTreeOptions 装载；切换时 handleParentChange 收窄权限树。
      component: 'TreeSelect',
      fieldName: 'parentId',
      label: $t('page.role.parentId'),
      componentProps: () => ({
        placeholder: $t('page.role.parentIdPlaceholder'),
        allowClear: true,
        showSearch: true,
        treeNodeFilterProp: 'label',
        treeDefaultExpandAll: true,
        treeData: roleTreeOptions.value,
        fieldNames: { children: 'children', label: 'label', value: 'value' },
        onChange: (value: any) => {
          handleParentChange(value || '');
        },
      }),
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
      // 与 react/ele 对齐：AI 生成钮进 label、textarea 占满整行（放 suffix 会把
      // 输入框压掉约 90px 宽），3 行起步比默认的两行高（ele rows:3 同款）
      labelRender: () =>
        h('span', { class: 'inline-flex items-center' }, [
          $t('ui.table.description'),
          h(AiGenerateButton, {
            scene: 'DESCRIPTION',
            onGenerate: (content: string) =>
              baseFormApi.setFieldValue('description', content),
          }),
        ]),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
        rows: 3,
      },
    },
    {
      // 分配权限：直接复用权限点抽屉的 PermissionCheckTree（对齐 ele 权限配置
      // 段的形态——段头顶格左对齐 + 全宽圆角边框盒内滚动 + 空态虚线占位）。
      // hideLabel 让该行脱离 130px 标签列、通栏铺满，段头由组件 title 承担。
      component: PermissionCheckTree,
      fieldName: 'permissions',
      hideLabel: true,
      componentProps: () => ({
        title: $t('page.role.permissions'),
        hint:
          permCapIds.value != null ? $t('page.role.permCapHint') : '',
        treeData: permTreeData.value,
        loading: permTreeLoading.value,
        emptyText: $t('common.noData'),
      }),
    },
    {
      // 字段权限：勾选 = 对该角色用户隐藏（黑名单语义）。对齐 react 基准——
      // 「字段权限」段头顶格 + 一张边框卡内 提示语 + User/Role 两组竖排勾选。
      // value 与后端各资源 proto 字段 json_name 逐字一致（当前 User/Role 两资源）。
      component: RoleFieldPermCard,
      fieldName: 'fieldPerm',
      hideLabel: true,
      componentProps: () => ({
        title: $t('page.role.fieldPerm.title'),
        hint: $t('page.role.fieldPerm.hint'),
        userOptions: [
          { label: $t('page.role.fieldPerm.field.email'), value: 'email' },
          { label: $t('page.role.fieldPerm.field.mobile'), value: 'mobile' },
          { label: $t('page.role.fieldPerm.field.telephone'), value: 'telephone' },
          { label: $t('page.role.fieldPerm.field.address'), value: 'address' },
          { label: $t('page.role.fieldPerm.field.region'), value: 'region' },
          { label: $t('page.role.fieldPerm.field.lastLoginAt'), value: 'lastLoginAt' },
          { label: $t('page.role.fieldPerm.field.lastLoginIp'), value: 'lastLoginIp' },
        ],
        roleOptions: [
          {
            label: $t('page.role.fieldPerm.field.permissions'),
            value: 'permissions',
          },
        ],
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

    // 保存收敛确认：编辑模式下超出父角色上限（树上不可见）的存量勾选会被剥离，
    // 先向操作者确认（编辑时校验语义的前端引导；后端亦有子集校验兜底）。
    if (!data.value?.create && permCapIds.value) {
      const stored = Array.isArray(data.value.row?.permissions)
        ? data.value.row.permissions.filter((v: any) => isIdString(v))
        : [];
      const outOfCap = stored.filter(
        (id: string) => !permCapIds.value!.includes(id),
      );
      if (outOfCap.length > 0) {
        const confirmed = await new Promise<boolean>((resolve) => {
          Modal.confirm({
            title: $t('page.role.permTrimConfirmTitle'),
            content: $t('page.role.permTrimConfirmContent', { count: outOfCap.length }),
            okText: $t('ui.confirm'),
            cancelText: $t('ui.cancel'),
            onOk: () => resolve(true),
            onCancel: () => resolve(false),
          });
        });
        if (!confirmed) {
          return; // 用户取消收敛，维持现状不提交
        }
      }
    }

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
      finalValues.permissions = filterIds(values.permissions);
    }

    // 父角色始终随表单提交：'0'=根角色（清空父角色须显式 0 进 updateMask，
    // 否则后端三态语义会把它当"未声明"静默跳过）。uint64 经 proto3 JSON 是字符串。
    finalValues.parentId = finalValues.parentId || '0';

    // 仅 SELECTED_UNITS 档提交授权单元集（含清空场景）；
    // 其余档位不携带该字段，后端维持既有集不替换。
    if (finalValues.dataScope === 'SELECTED_UNITS') {
      finalValues.orgUnits = Array.isArray(values.orgUnits)
        ? filterIds(values.orgUnits)
        : [];
    } else {
      delete finalValues.orgUnits;
    }

    // 字段权限始终随表单提交（含清空场景）：抽屉所见即保存后的最终态。
    // fieldPerm 是表单专用中间字段（{ user, role } 两组勾选），不是 Role proto
    // 字段，提交前必须折算成 fieldPermissions 多资源数组并移除。
    finalValues.fieldPermissions = (
      [
        ['User', finalValues.fieldPerm?.user],
        ['Role', finalValues.fieldPerm?.role],
      ] as const
    )
      .filter(([, fields]) => Array.isArray(fields) && fields.length > 0)
      .map(([resource, fields]) => ({
        resource,
        hiddenFields: fields.filter((v: unknown) => typeof v === 'string'),
      }));
    delete finalValues.fieldPerm;


    try {
      await (data.value?.create
        ? createRole(finalValues)
        : updateRole({ id: data.value.row.id, values: finalValues }));

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

      // 为表单赋值。注意：setValues 必须只调一次——连调两次时后一次会清掉
      // 前一次刚设置的值（vee-validate setValues 非合并语义），因此把字段权限
      // 的中间态 fieldPerm（User/Role 两组）合并在同一次调用里。
      const row = data.value?.row;
      const findHiddenFields = (resource: string) => {
        const entry = Array.isArray(row?.fieldPermissions)
          ? row.fieldPermissions.find((e: any) => e?.resource === resource)
          : undefined;
        return Array.isArray(entry?.hiddenFields)
          ? entry.hiddenFields.filter((v: any) => typeof v === 'string')
          : [];
      };
      await baseFormApi.setValues({
        ...(row ?? {}),
        fieldPerm: {
          user: findHiddenFields('User'),
          role: findHiddenFields('Role'),
        },
      });

      // 分配权限勾选树（打开抽屉时拉取，对齐权限点抽屉），随后按初始父角色
      // （编辑回填值）收窄，并把超出上限的存量勾选收敛为提交所见即所得。
      await Promise.all([loadPermTree(), loadRoleTreeOptions()]);
      const initialParentId = data.value?.create ? '' : String(row?.parentId || '');
      await rebuildPermTree(initialParentId || null);
      if (!data.value?.create && permCapIds.value && Array.isArray(row?.permissions)) {
        const eff = row.permissions.filter(
          (id: any) => isIdString(id) && permCapIds.value!.includes(id),
        );
        await baseFormApi.setFieldValue('permissions', eff);
      }
      await closeGuard.syncBaseline();

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
