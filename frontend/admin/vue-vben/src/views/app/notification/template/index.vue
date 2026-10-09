<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import type {
  notificationservicev1_NotificationTemplate as NotificationTemplate,
} from '#/api';

import { reactive, ref } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { message, notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListNotificationTemplates,
  NOTIFICATION_TEMPLATE_UPDATE_MASK,
  PaginationQuery,
  useCreateNotificationTemplate,
  useDeleteNotificationTemplate,
  useRenderNotificationTemplate,
  useUpdateNotificationTemplate,
} from '#/api';

/**
 * 通知模板管理（可复用的标题/正文占位模板，{{var}} 占位符）。
 *
 * 发送方以 template_code 引用（SendDirect 的 templateCode/templateVars），
 * 渲染发生在落台账之前。「预览」用给的变量集走服务端 Render，
 * 看到的字节即发送字节。建后 code 不可改（引用锚，改码等于让引用悬空）。
 */

const { mutateAsync: createTemplate } = useCreateNotificationTemplate();
const { mutateAsync: updateTemplate } = useUpdateNotificationTemplate();
const { mutateAsync: deleteTemplate } = useDeleteNotificationTemplate();
const { mutateAsync: renderTemplate } = useRenderNotificationTemplate();

const formOptions = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.notificationTemplate.name'),
      componentProps: { placeholder: $t('ui.placeholder.input'), allowClear: true },
    },
    {
      component: 'Input',
      fieldName: 'code',
      label: $t('page.notificationTemplate.code'),
      componentProps: { placeholder: $t('ui.placeholder.input'), allowClear: true },
    },
  ],
};

const gridOptions: VxeGridProps<NotificationTemplate> = {
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
        return fetchListNotificationTemplates(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues: {
              code: values.code,
              name: values.name,
            },
          }),
        );
      },
    },
  },
  columns: [
    { title: $t('page.notificationTemplate.name'), field: 'name', minWidth: 150 },
    {
      title: $t('page.notificationTemplate.code'),
      field: 'code',
      slots: { default: 'code' },
      width: 180,
    },
    {
      title: $t('page.notificationTemplate.titleTemplate'),
      field: 'titleTemplate',
      minWidth: 220,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.notificationTemplate.contentTemplate'),
      field: 'contentTemplate',
      minWidth: 280,
      showOverflow: 'tooltip',
    },
    {
      title: $t('page.notificationTemplate.isEnabled'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      width: 90,
    },
    {
      title: $t('page.notificationTemplate.updatedAt'),
      field: 'updatedAt',
      formatter: 'formatDateTime',
      width: 170,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 200,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// ============ 创建/编辑 ============
const editOpen = ref(false);
const editMode = ref<'create' | 'edit'>('create');
const editSaving = ref(false);
const editingId = ref<number>();

const form = reactive({
  code: '',
  contentTemplate: '',
  isEnabled: true,
  name: '',
  remark: '',
  titleTemplate: '',
});

function resetForm() {
  form.name = '';
  form.code = '';
  form.titleTemplate = '';
  form.contentTemplate = '';
  form.isEnabled = true;
  form.remark = '';
}

function openCreate() {
  editMode.value = 'create';
  editingId.value = undefined;
  resetForm();
  editOpen.value = true;
}

function openEdit(row: NotificationTemplate) {
  editMode.value = 'edit';
  editingId.value = row.id;
  form.name = row.name || '';
  form.code = row.code || '';
  form.titleTemplate = row.titleTemplate || '';
  form.contentTemplate = row.contentTemplate || '';
  form.isEnabled = !!row.isEnabled;
  form.remark = row.remark || '';
  editOpen.value = true;
}

async function handleSave() {
  if (!form.name) {
    message.error($t('page.notificationTemplate.requiredName'));
    return;
  }
  if (!form.code) {
    message.error($t('page.notificationTemplate.requiredCode'));
    return;
  }
  if (!form.titleTemplate) {
    message.error($t('page.notificationTemplate.requiredTitleTemplate'));
    return;
  }
  if (!form.contentTemplate) {
    message.error($t('page.notificationTemplate.requiredContentTemplate'));
    return;
  }

  editSaving.value = true;
  try {
    // CRUD 的请求体必须包 data；掩码固定列（NOTIFICATION_TEMPLATE_UPDATE_MASK），code 不在其中
    const data = {
      code: form.code,
      contentTemplate: form.contentTemplate,
      isEnabled: form.isEnabled,
      name: form.name,
      remark: form.remark,
      titleTemplate: form.titleTemplate,
    };
    if (editMode.value === 'create') {
      await createTemplate({ data });
      message.success($t('page.notificationTemplate.createSuccess'));
    } else if (editingId.value) {
      await updateTemplate({
        data,
        id: editingId.value,
        updateMask: NOTIFICATION_TEMPLATE_UPDATE_MASK,
      });
      message.success($t('page.notificationTemplate.updateSuccess'));
    }
    editOpen.value = false;
    await gridApi.reload();
  } catch (error: any) {
    console.error('[notification-template] save failed', error);
    message.error(error?.message || $t('page.notificationTemplate.saveFailed'));
  } finally {
    editSaving.value = false;
  }
}

// ============ 删除 ============
async function handleDelete(row: NotificationTemplate) {
  try {
    await deleteTemplate({ id: row.id });
    notification.success({ message: $t('page.notificationTemplate.deleteSuccess') });
    await gridApi.reload();
  } catch (error: any) {
    console.error('[notification-template] delete failed', error);
    notification.error({
      message: error?.message || $t('page.notificationTemplate.deleteFailed'),
    });
  }
}

// ============ 试渲染 ============
const renderOpen = ref(false);
const renderSaving = ref(false);
const renderRow = ref<NotificationTemplate>();
const varsJson = ref('');
const renderResult = ref<{ content?: string; title?: string }>();

function openRender(row: NotificationTemplate) {
  renderRow.value = row;
  varsJson.value = '';
  renderResult.value = undefined;
  renderOpen.value = true;
}

async function handleRender() {
  const id = renderRow.value?.id;
  if (!id) return;

  let variables: Record<string, string> = {};
  const raw = (varsJson.value || '').trim();
  if (raw) {
    try {
      variables = JSON.parse(raw);
    } catch (error: any) {
      console.error('[notification-template] parse vars json failed', error);
      message.error($t('page.notificationTemplate.varsJsonInvalid'));
      return;
    }
  }

  renderSaving.value = true;
  try {
    // body:"*" 自定义路由，请求体扁平：包一层 { data } 会被 protojson 当未知字段丢掉
    const resp = await renderTemplate({ id, variables });
    // 结果就地展示，弹窗保持打开，用户看完手动关闭
    renderResult.value = { content: resp.content ?? '', title: resp.title ?? '' };
  } catch (error: any) {
    console.error('[notification-template] render failed', error);
    message.error(error?.message || $t('page.notificationTemplate.renderFailed'));
  } finally {
    renderSaving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.notification.templates')">
      <template #toolbar-tools>
        <a-button type="primary" class="mr-2" @click="openCreate">
          {{ $t('page.notificationTemplate.create') }}
        </a-button>
      </template>
      <template #code="{ row }">
        <a-tag v-if="row.code" color="geekblue">{{ row.code }}</a-tag>
        <template v-else>-</template>
      </template>
      <template #isEnabled="{ row }">
        <a-tag :color="row.isEnabled ? 'green' : 'default'">
          {{
            row.isEnabled
              ? $t('page.notificationTemplate.enabledOn')
              : $t('page.notificationTemplate.enabledOff')
          }}
        </a-tag>
      </template>
      <template #action="{ row }">
        <a-button type="link" size="small" @click.stop="openEdit(row)">
          {{ $t('page.notificationTemplate.edit') }}
        </a-button>
        <a-button type="link" size="small" @click.stop="openRender(row)">
          {{ $t('page.notificationTemplate.render') }}
        </a-button>
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="$t('page.notificationTemplate.deleteConfirm')"
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" size="small">
            {{ $t('page.notificationTemplate.delete') }}
          </a-button>
        </a-popconfirm>
      </template>
    </Grid>

    <!-- 创建/编辑 -->
    <a-modal
      v-model:open="editOpen"
      :title="
        editMode === 'create'
          ? $t('page.notificationTemplate.create')
          : $t('page.notificationTemplate.edit')
      "
      :confirm-loading="editSaving"
      :mask-closable="false"
      destroy-on-close
      @ok="handleSave"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item :label="$t('page.notificationTemplate.name')" required>
          <a-input
            v-model:value="form.name"
            :maxlength="100"
            :placeholder="$t('ui.placeholder.input')"
          />
        </a-form-item>
        <a-form-item
          :help="$t('page.notificationTemplate.codeHint')"
          :label="$t('page.notificationTemplate.code')"
          required
        >
          <!-- 编辑态禁改：它是发送方的引用锚，改码等于让所有引用悬空 -->
          <a-input
            v-model:value="form.code"
            :disabled="editMode === 'edit'"
            :maxlength="64"
            :placeholder="$t('page.notificationTemplate.codePattern')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.notificationTemplate.titleTemplate')" required>
          <a-input
            v-model:value="form.titleTemplate"
            :maxlength="500"
            :placeholder="$t('page.notificationTemplate.titleTemplatePlaceholder')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.notificationTemplate.contentTemplate')" required>
          <a-textarea
            v-model:value="form.contentTemplate"
            :maxlength="20000"
            :rows="6"
            show-count
            :placeholder="$t('page.notificationTemplate.contentTemplatePlaceholder')"
          />
        </a-form-item>
        <a-form-item :label="$t('page.notificationTemplate.isEnabled')">
          <a-switch v-model:checked="form.isEnabled" />
        </a-form-item>
        <a-form-item :label="$t('page.notificationTemplate.remark')">
          <a-textarea v-model:value="form.remark" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 试渲染 -->
    <a-modal
      v-model:open="renderOpen"
      :title="
        $t('page.notificationTemplate.renderTitle', { name: renderRow?.name ?? '' })
      "
      :confirm-loading="renderSaving"
      destroy-on-close
      @ok="handleRender"
    >
      <a-form layout="vertical" class="pt-2">
        <a-form-item
          :help="$t('page.notificationTemplate.varsJsonHint')"
          :label="$t('page.notificationTemplate.varsJson')"
        >
          <a-textarea
            v-model:value="varsJson"
            :rows="4"
            :placeholder="$t('page.notificationTemplate.varsJsonPlaceholder')"
          />
        </a-form-item>
      </a-form>
      <a-descriptions v-if="renderResult" bordered :column="1" class="mt-2" size="small">
        <a-descriptions-item :label="$t('page.notificationTemplate.renderedTitle')">
          {{ renderResult.title }}
        </a-descriptions-item>
        <a-descriptions-item :label="$t('page.notificationTemplate.renderedContent')">
          <pre class="whitespace-pre-wrap">{{ renderResult.content }}</pre>
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </Page>
</template>
