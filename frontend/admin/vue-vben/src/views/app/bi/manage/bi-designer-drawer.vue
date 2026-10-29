<script lang="ts" setup>
/**
 * BI 设计器抽屉（vben 端）：上部基础信息（vben Form）+ 下部 @go-wind/bi-vue 画布。
 * edit 态载入行 payload；保存整份覆盖 schema，edit 走 version 乐观锁（409 提示刷新）。
 * 与 ele 端 designer-dialog.vue 同协议：open({ row? }) → success。
 */
import type { VbenFormProps } from '@vben/common-ui';

import { computed, ref, watch } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  useCreateBiDashboard,
  useUpdateBiDashboard,
} from '#/api/composables/bi_dashboard';

import { BiDesigner } from '@go-wind/bi-vue';
import '@go-wind/bi-vue/style.css';
import type { BiDashboardSchema } from '@go-wind/bi-core';

const emit = defineEmits<{ success: [] }>();

const { mutateAsync: createDashboard } = useCreateBiDashboard();
const { mutateAsync: updateDashboard } = useUpdateBiDashboard();

const data = ref();

const mode = ref<'create' | 'edit'>('create');
const schemaRef = ref<BiDashboardSchema | null>(null);
const canvasDirty = ref(false);
const initialSchema = ref<Partial<BiDashboardSchema> | undefined>(undefined);

const formOptions: VbenFormProps = {
  showDefaultActions: false,
  commonConfig: {
    componentProps: { class: 'w-auto' },
  },
  wrapperClass: 'grid grid-cols-4 gap-x-4',
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.bi.name'),
      rules: 'required',
      componentProps: { placeholder: $t('ui.placeholder.input'), allowClear: true },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.bi.code'),
      rules: 'required',
      componentProps: { placeholder: 'biz.sales.board', allowClear: true },
    },
    {
      component: 'Input',
      fieldName: 'remark',
      label: $t('page.bi.remark'),
      formItemClass: 'col-span-2',
      componentProps: { allowClear: true },
    },
    {
      component: 'Switch',
      fieldName: 'isEnabled',
      label: $t('page.bi.isEnabled'),
      defaultValue: true,
      componentProps: { class: 'w-auto' },
    },
    {
      component: 'Switch',
      fieldName: 'isPublic',
      label: $t('page.bi.isPublic'),
      defaultValue: false,
      componentProps: { class: 'w-auto' },
    },
  ],
};

const [BaseForm, baseFormApi] = useVbenForm(formOptions);

const title = computed(() =>
  mode.value === 'edit' ? $t('page.bi.editTitle') : $t('page.bi.createTitle'),
);

/** 画布 key：换行编辑时强制重挂载（initialSchema 是挂载快照，不回灌） */
const canvasKey = computed(() => `${mode.value}-${data.value?.row?.id ?? 'new'}`);

watch(
  () => data.value,
  (val) => {
    initialSchema.value = undefined;
    schemaRef.value = null;
    const row = val?.row;
    if (row?.payload) {
      try {
        initialSchema.value = JSON.parse(row.payload);
      } catch (error) {
        // 脏 payload 不阻塞打开：记原始错误，按空白画布编辑
        console.error('[bi-designer] payload 解析失败，按空白画布打开', error);
      }
    }
  },
);

function handleSchemaChange(schema: BiDashboardSchema) {
  schemaRef.value = schema;
  canvasDirty.value = true;
}

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[96%]',
  confirmText: $t('common.save') || $t('common.button.save'),
  cancelText: $t('common.cancel') || $t('common.button.cancel'),
  onBeforeClose() {
    if (!canvasDirty.value) return true;
    return false;
  },
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }
    const schema = schemaRef.value;
    if (!schema) {
      notification.warning({ message: $t('page.bi.designerNotReady') });
      return;
    }
    const payload = JSON.stringify(schema);
    const chartCount = schema.charts?.length ?? 0;
    const themeKey = schema.theme?.key ?? '';

    drawerApi.setState({ loading: true });

    const values = await baseFormApi.getValues();

    try {
      if (mode.value === 'edit' && data.value?.row?.id) {
        await updateDashboard({
          id: data.value.row.id,
          values: {
            ...values,
            // 乐观锁：回传打开时读到的版本，冲突时后端 409
            version: data.value.row.version ?? 1,
            payload,
            chartCount,
            themeKey,
          },
        });
        notification.success({ message: $t('ui.notification.update_success') });
      } else {
        await createDashboard({
          data: {
            ...values,
            kind: 'DASHBOARD',
            payload,
            chartCount,
            themeKey,
            isBuiltIn: false,
            sortOrder: 100,
          } as any,
        });
        notification.success({ message: $t('ui.notification.create_success') });
      }
      canvasDirty.value = false;
      emit('success');
    } catch (error) {
      console.error('save bi dashboard failed:', error);
      const msg = String((error as Error)?.message || '');
      notification.error({
        message: msg.includes('modified by another session')
          ? $t('page.bi.saveConflict')
          : $t('ui.notification.update_failed'),
      });
    } finally {
      drawerApi.close();
      drawerApi.setState({ loading: false });
    }
  },

  async onOpenChange(isOpen) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();
      mode.value = data.value?.create ? 'create' : 'edit';
      canvasDirty.value = false;

      const row = data.value?.row;
      await baseFormApi.setValues({
        name: row?.name ?? '',
        code: row?.code ?? '',
        remark: row?.remark ?? '',
        isEnabled: row?.isEnabled ?? true,
        isPublic: row?.isPublic ?? false,
      });
      drawerApi.setState({ loading: false });
    }
  },
});

/** 打开设计器（create / edit 两态）；页面持 ref 直接调用 */
function open(options: { create?: boolean; row?: any }) {
  mode.value = options?.create ? 'create' : 'edit';
  drawerApi.setData({
    create: !!options?.create,
    row: options?.row,
  });
  drawerApi.open();
}

defineExpose({ open });
</script>

<template>
  <Drawer :title="title" class="flex flex-col">
    <div class="flex h-full flex-col gap-2">
      <BaseForm />
      <!-- 画布容器：任何指针/键盘交互视为脏 -->
      <div
        class="min-h-[420px] flex-1"
        style="height: calc(100vh - 320px)"
        @pointerdown.capture="canvasDirty = true"
        @keydown.capture="canvasDirty = true"
      >
        <BiDesigner
          :key="canvasKey"
          :demo="mode === 'create'"
          :initial-schema="initialSchema"
          style="display: block; height: 100%"
          @change="handleSchemaChange"
        />
      </div>
    </div>
  </Drawer>
</template>
