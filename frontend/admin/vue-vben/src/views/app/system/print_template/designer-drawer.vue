<script lang="ts" setup>
import { computed, ref, watch } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { Modal, notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import {
  useCreatePrintTemplate,
  useUpdatePrintTemplate,
} from '#/api/composables/print_template';
import { useFormCloseGuard } from '#/composables/use-form-close-guard';

// go-wind-print 设计器画布（Vue3 组件，无 UI 库依赖）；样式全局引一次
import { PrintDesigner, createDefaultTemplate } from '@go-wind/print-canvas';
import '@go-wind/print-canvas/style.css';
import type {
  MultiPageTemplateData,
  PrintBusinessField,
  TemplateData,
} from '@go-wind/print-core/designer';
import { renderHtmlPages, browserCodeRenderer } from '@go-wind/print-core/browser';

const { mutateAsync: createTemplate } = useCreatePrintTemplate();
const { mutateAsync: updateTemplate } = useUpdatePrintTemplate();

const emit = defineEmits<{ success: [] }>();

const data = ref();

const mode = ref<'create' | 'edit' | 'copy'>('create');
const canvasRef = ref<any>(null);
const canvasDirty = ref(false);
const previewVisible = ref(false);
const previewLoading = ref(false);
const previewFrameRef = ref<HTMLIFrameElement | null>(null);
const initialTemplate = ref<TemplateData | MultiPageTemplateData | undefined>(
  createDefaultTemplate() as any,
);
const fields = ref<PrintBusinessField[]>([]);
const sampleText = ref('');

const moduleName = computed(() => $t('page.printTemplate.moduleName'));

const getTitle = computed(() =>
  mode.value === 'edit'
    ? $t('ui.modal.update', { moduleName: moduleName.value })
    : $t('ui.modal.create', { moduleName: moduleName.value }),
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-[96%]',
    },
  },
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.printTemplate.name'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.printTemplate.code'),
      componentProps: {
        placeholder: 'biz.order.a4',
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'Select',
      fieldName: 'paperSize',
      label: $t('page.printTemplate.paperSize'),
      defaultValue: 'A4',
      componentProps: {
        options: ['A4', 'A3', 'A5', 'Letter', 'Legal'].map((v) => ({ value: v, label: v })),
      },
    },
    {
      component: 'Select',
      fieldName: 'orientation',
      label: $t('page.printTemplate.orientation'),
      defaultValue: 'portrait',
      componentProps: {
        options: [
          { value: 'portrait', label: $t('page.printTemplate.portrait') },
          { value: 'landscape', label: $t('page.printTemplate.landscape') },
        ],
      },
    },
    {
      component: 'Switch',
      fieldName: 'isEnabled',
      label: $t('page.printTemplate.isEnabled'),
      defaultValue: true,
      componentProps: { class: 'w-auto' },
    },
    {
      component: 'InputNumber',
      fieldName: 'sortOrder',
      label: $t('page.printTemplate.sortOrder'),
      defaultValue: 100,
      componentProps: { class: 'w-auto', min: 0 },
    },
    {
      component: 'Input',
      fieldName: 'resourceKey',
      label: $t('page.printTemplate.resourceKey'),
      componentProps: {
        placeholder: 'system/user',
        allowClear: true,
      },
    },
    {
      component: 'Textarea',
      fieldName: 'sampleData',
      label: $t('page.printTemplate.sampleData'),
      formItemClass: 'col-span-2',
      componentProps: {
        rows: 2,
        placeholder: '{"order":{"no":"SO-001","items":[{"sku":"A1","qty":2}]}}',
      },
    },
  ],
});

// 关闭守卫：表单脏走基线对比；画布脏单独确认（docs/form_drawer_close_guard.md）
const closeGuard = useFormCloseGuard(baseFormApi);

async function onBeforeClose() {
  if (canvasDirty.value) {
    return new Promise<boolean>((resolve) => {
      Modal.confirm({
        title: $t('common.drawer.unsavedMessage') || $t('ui.text.do_you_want_close'),
        okText: $t('common.discardButton') || $t('common.confirm'),
        cancelText: $t('common.cancel'),
        onOk: () => {
          canvasDirty.value = false;
          resolve(true);
        },
        onCancel: () => resolve(false),
      });
    });
  }
  return closeGuard.onBeforeClose();
}

// 示例数据变化 → 重派生字段面板
watch(sampleText, (text) => {
  fields.value = fieldsFromSample(parseSample(text));
});

// —— 暗色主题：画布亮色值定义在 .designer-container 上（scoped），注入 !important 覆盖块。
// 与钉死的 1.3.11 类名耦合；fork 内核应加正式 data-theme 块替代（docs/print_template.md）。
const DARK_CSS =
  '.designer-container{--pd-bg:#0b0f19!important;--pd-surface:#111827!important;--pd-sidebar:#111827!important;--pd-sidebar-hover:#1f2937!important;--pd-field-bg:#0d1420!important;--pd-input-bg:#0d1420!important;--pd-hover-tint:rgba(255,255,255,.06)!important;--pd-canvas-bg:#0a0e17!important;--pd-text:#e6e8ef!important;--pd-text-regular:#d1d5db!important;--pd-text-muted:#9aa3b2!important;--pd-text-faint:#6b7280!important;--pd-border:#2a3444!important;--pd-border-soft:#232c3a!important;--pd-accent:#4a8dff!important;--pd-accent-soft:rgba(74,141,255,.16)!important;--pd-accent-2:#7c9dff!important;--pd-accent-secondary:#7c9dff!important;--pd-ruler-bg:#0d1420!important;--pd-ruler:#2a3444!important;--pd-ruler-cursor:#f87171!important;--pd-paper:#ffffff!important;--pd-shadow-paper:rgba(0,0,0,.55)!important;--pd-scrollbar-thumb:#2a3444!important;--pd-scrollbar-thumb-hover:#3a4557!important;--pd-menu-text:#e6e8ef!important;--pd-menu-muted:#9aa3b2!important;--pd-menu-hover:#1f2937!important;--pd-menu-divider:#232c3a!important;--pd-menu-disabled:#4b5563!important;--pd-menu-border:#232c3a!important;--pd-shadow-sm:0 1px 2px rgba(0,0,0,.5)!important;--pd-shadow-md:0 4px 12px rgba(0,0,0,.5)!important}';

function syncDarkStyle() {
  const id = 'pd-host-dark-theme';
  let style = document.getElementById(id);
  const isDark = document.documentElement.classList.contains('dark');
  if (isDark && !style) {
    style = document.createElement('style');
    style.id = id;
    style.textContent = DARK_CSS;
    document.head.appendChild(style);
  } else if (!isDark && style) {
    style.remove();
  }
}

let classObserver: MutationObserver | null = null;

function fieldsFromSample(
  data: Record<string, unknown>,
): PrintBusinessField[] {
  const out: PrintBusinessField[] = [];
  let order = 0;
  const push = (key: string, label: string, fieldType: string) => {
    out.push({ fieldKey: key, fieldLabel: label, fieldType, sortOrder: ++order });
  };
  const walk = (obj: Record<string, unknown>, prefix: string) => {
    Object.entries(obj || {}).forEach(([k, v]) => {
      const key = prefix ? `${prefix}.${k}` : k;
      if (Array.isArray(v)) {
        push(key, k, 'list');
        v.forEach((item) => {
          if (item && typeof item === 'object' && !Array.isArray(item)) {
            walk(item as Record<string, unknown>, `${key}[]`);
          }
        });
      } else if (v && typeof v === 'object') {
        walk(v as Record<string, unknown>, key);
      } else {
        push(key, k, typeof v === 'number' ? 'number' : 'text');
      }
    });
  };
  walk(data, '');
  return out;
}

function parseSample(text?: string): Record<string, unknown> {
  if (!text || !text.trim()) return {};
  try {
    const v = JSON.parse(text);
    return v && typeof v === 'object' && !Array.isArray(v) ? v : {};
  } catch {
    return {};
  }
}

function safeParseTemplate(text: string): TemplateData | MultiPageTemplateData {
  try {
    return JSON.parse(text);
  } catch {
    return createDefaultTemplate() as any;
  }
}

const [Drawer, drawerApi] = useVbenDrawer({
  class: 'w-[96%]',
  confirmText: $t('common.save') || $t('common.button.save'),
  cancelText: $t('common.cancel') || $t('common.button.cancel'),
  onBeforeClose: () => onBeforeClose(),
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }
    const json = currentTemplateJson();
    if (!json) {
      notification.warning({ message: $t('page.printTemplate.designerNotReady') });
      return;
    }

    setLoading(true);

    const values = await baseFormApi.getValues();
    let redundant: any = {};
    try {
      const tpl = JSON.parse(json);
      redundant = { paperSize: tpl.paperSize, orientation: tpl.orientation };
    } catch {
      // 保底走表单值
    }
    const payloadValues: Record<string, any> = {
      ...values,
      payload: json,
      paperSize: redundant.paperSize || values.paperSize,
      orientation: redundant.orientation || values.orientation,
      resourceKey: values.resourceKey || '',
    };

    try {
      if (mode.value === 'edit' && data.value?.row?.id) {
        await updateTemplate({ id: data.value.row.id, values: payloadValues });
        notification.success({ message: $t('ui.notification.update_success') });
      } else {
        await createTemplate({ data: { ...payloadValues, isBuiltIn: false } as any });
        notification.success({ message: $t('ui.notification.create_success') });
      }
      closeGuard.markClean();
      canvasDirty.value = false;
    } catch (error) {
      console.error('save print template failed:', error);
      notification.error({
        message:
          mode.value === 'edit'
            ? $t('ui.notification.update_failed')
            : $t('ui.notification.create_failed'),
      });
    } finally {
      drawerApi.close();
      setLoading(false);
    }
  },

  async onOpenChange(isOpen) {
    if (isOpen) {
      data.value = drawerApi.getData<Record<string, any>>();
      mode.value = data.value?.copy ? 'copy' : data.value?.create ? 'create' : 'edit';
      canvasDirty.value = false;

      syncDarkStyle();
      if (!classObserver) {
        classObserver = new MutationObserver(syncDarkStyle);
        classObserver.observe(document.documentElement, {
          attributes: true,
          attributeFilter: ['class'],
        });
      }

      const row = data.value?.row;
      await baseFormApi.setValues({
        name: mode.value === 'copy' ? `${row?.name || ''} ` + $t('page.printTemplate.copySuffix') : row?.name ?? '',
        code: mode.value === 'copy' ? '' : row?.code ?? '',
        paperSize: row?.paperSize ?? 'A4',
        orientation: row?.orientation ?? 'portrait',
        isEnabled: row?.isEnabled ?? true,
        sortOrder: row?.sortOrder ?? 100,
        resourceKey: row?.resourceKey ?? '',
        sampleData: '',
      });
      sampleText.value = '';
      fields.value = [];
      initialTemplate.value =
        mode.value === 'create'
          ? (createDefaultTemplate() as any)
          : safeParseTemplate((row?.payload || '') as string);

      await closeGuard.syncBaseline();
      setLoading(false);
    } else {
      classObserver?.disconnect();
      classObserver = null;
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ loading });
}

/** 打开设计器（create / edit / copy 三态）；页面持 ref 直接调用，不再走 connectedComponent 双实例 */
function open(options: { create?: boolean; copy?: boolean; row?: any }) {
  mode.value = options?.copy ? 'copy' : options?.create ? 'create' : 'edit';
  drawerApi.setData({
    create: !!options?.create,
    copy: !!options?.copy,
    row: options?.row,
  });
  drawerApi.open();
}

defineExpose({ open });

function markCanvasDirty() {
  canvasDirty.value = true;
}

function currentTemplateJson(): string | null {
  try {
    // 注意：Vue 画布的 getTemplateJson() 返回对象（单页裸 TemplateData，多页 {version,pages}），
    // 与 react CE（字符串）不同——统一归一化成字符串再进 payload
    const raw = canvasRef.value?.getTemplateJson();
    if (raw === null || raw === undefined) return null;
    return typeof raw === 'string' ? raw : JSON.stringify(raw);
  } catch {
    return null;
  }
}

async function handlePreview() {
  const json = currentTemplateJson();
  if (!json) {
    notification.warning({ message: $t('page.printTemplate.designerNotReady') });
    return;
  }
  previewVisible.value = true;
  previewLoading.value = true;
  try {
    const result = await renderHtmlPages(
      JSON.parse(json),
      parseSample(sampleText.value),
      window.location.origin,
      browserCodeRenderer,
    );
    window.setTimeout(() => {
      const doc = previewFrameRef.value?.contentWindow?.document;
      if (doc) {
        doc.open();
        doc.write(result.html);
        doc.close();
      }
      previewLoading.value = false;
    }, 80);
  } catch (e) {
    previewLoading.value = false;
    notification.error({ message: (e as Error).message || $t('page.printTemplate.renderFailed') });
  }
}

function handlePreviewPrint() {
  previewFrameRef.value?.contentWindow?.print();
}

/** 内容区保存（不依赖框架 footer 接线）；成功后先清脏再关，触发 onBeforeClose 直通 */
const submitLoading = ref(false);

async function handleSave() {
  const validate = await baseFormApi.validate();
  if (!validate.valid) {
    return;
  }
  const json = currentTemplateJson();
  if (!json) {
    notification.warning({ message: $t('page.printTemplate.designerNotReady') });
    return;
  }

  submitLoading.value = true;
  try {
    const values = await baseFormApi.getValues();
    let redundant: any = {};
    try {
      const tpl = JSON.parse(json);
      redundant = { paperSize: tpl.paperSize, orientation: tpl.orientation };
    } catch {
      // 保底走表单值
    }
    const payloadValues: Record<string, any> = {
      ...values,
      payload: json,
      paperSize: redundant.paperSize || values.paperSize,
      orientation: redundant.orientation || values.orientation,
      resourceKey: values.resourceKey || '',
    };

    if (mode.value === 'edit' && data.value?.row?.id) {
      await updateTemplate({ id: data.value.row.id, values: payloadValues });
      notification.success({ message: $t('ui.notification.update_success') });
    } else {
      await createTemplate({ data: { ...payloadValues, isBuiltIn: false } as any });
      notification.success({ message: $t('ui.notification.create_success') });
    }
    closeGuard.markClean();
    canvasDirty.value = false;
    emit('success');
    drawerApi.close();
  } catch (error) {
    console.error('save print template failed:', error);
    notification.error({
      message:
        mode.value === 'edit'
          ? $t('ui.notification.update_failed')
          : $t('ui.notification.create_failed'),
    });
  } finally {
    submitLoading.value = false;
  }
}
</script>

<template>
  <Drawer :title="getTitle">
    <div class="flex min-w-0 max-w-full flex-col gap-3 overflow-x-hidden">
      <BaseForm />
      <div class="mb-2 flex items-center gap-2">
        <a-button @click="handlePreview">{{ $t('page.printTemplate.preview') }}</a-button>
        <a-button type="primary" :loading="submitLoading" @click="handleSave">
          {{ $t('common.save') }}
        </a-button>
      </div>
      <div
        class="designer-host overflow-hidden rounded-lg border"
        @pointerdown.capture="markCanvasDirty"
        @keydown.capture="markCanvasDirty"
      >
        <PrintDesigner
          ref="canvasRef"
          is-edit
          :show-help="false"
          :initial-template="initialTemplate"
          :fields="fields"
          style="display: block; height: 100%"
        />
      </div>
    </div>

    <!-- 打印预览：放在 Drawer 内部保持单根（vben connectedComponent 对多根组件的绑定会错乱） -->
    <Modal v-model:open="previewVisible" :title="$t('page.printTemplate.previewTitle')" width="82%">
      <div class="preview-body">
        <div v-if="previewLoading" class="preview-loading">{{ $t('page.printTemplate.rendering') }}</div>
        <iframe ref="previewFrameRef" title="print-preview" class="preview-frame" />
      </div>
      <template #footer>
        <a-button @click="previewVisible = false">{{ $t('common.close') }}</a-button>
        <a-button type="primary" @click="handlePreviewPrint">
          {{ $t('page.printTemplate.print') }}
        </a-button>
      </template>
    </Modal>
  </Drawer>
</template>

<style scoped>
.designer-host {
  height: calc(100vh - 360px);
  min-height: 420px;
}

.preview-body {
  height: 68vh;
  background: #f5f5f5;
  overflow: auto;
}

.preview-loading {
  padding: 24px;
}

.preview-frame {
  width: 100%;
  height: 68vh;
  border: none;
}
</style>
