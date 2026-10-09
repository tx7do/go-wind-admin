<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';

import { h, ref } from 'vue';

import { Page, useVbenDrawer, type VbenFormProps } from '@vben/common-ui';
import { LucideFilePenLine, LucideCirclePlay, LucideTrash2 } from '@vben/icons';

import { notification } from 'ant-design-vue';

import { useVbenVxeGrid } from '#/adapter/vxe-table';
import {
  fetchListScripts,
  PaginationQuery,
  scriptLanguageList,
  scriptLanguageToColor,
  scriptLanguageToName,
  useDeleteScript,
  useTestRunScript,
  useUpdateScript,
} from '#/api';
import { type scriptservicev1_Script as Script } from '#/api';
import { $t } from '#/locales';
import TableExportButton from '#/components/TableExportButton.vue';

import ScriptDrawer from './script-drawer.vue';
import ScriptLogDrawer from './script-log-drawer.vue';

const { mutateAsync: deleteScript } = useDeleteScript();
const { mutateAsync: updateScript } = useUpdateScript();
const { mutateAsync: testRunMutate } = useTestRunScript();

const testRunOpen = ref(false);
const testRunScript = ref<Script>();
const testRunInputRows = ref<Array<{ key: string; value: string }>>([]);
const testRunResult = ref<any>();
const testRunLoading = ref(false);

const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.script.name'),
      componentProps: {
        placeholder: $t('page.script.namePlaceholder'),
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'language',
      label: $t('page.script.language'),
      componentProps: {
        options: scriptLanguageList,
        placeholder: $t('ui.placeholder.select'),
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'hookPoint',
      label: $t('page.script.hookPoint'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
    },
  ],
};

const gridOptions: VxeGridProps<Script> = {
  toolbarConfig: {
    custom: true,
    export: true,
    refresh: true,
    zoom: true,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  height: 'auto',
  stripe: true,
  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        return await fetchListScripts(
          new PaginationQuery({
            paging: { page: page.currentPage, pageSize: page.pageSize },
            formValues,
          }),
        );
      },
    },
  },
  columns: [
    { type: 'seq', width: 60 },
    { title: $t('page.script.name'), field: 'name', minWidth: 180 },
    {
      title: $t('page.script.language'),
      field: 'language',
      slots: { default: 'language' },
      width: 110,
    },
    {
      title: $t('page.script.hookPoint'),
      field: 'hookPoint',
      slots: { default: 'hookPoint' },
      minWidth: 160,
    },
    { title: $t('page.script.priority'), field: 'priority', width: 80 },
    {
      title: $t('page.script.isEnabled'),
      field: 'isEnabled',
      slots: { default: 'isEnabled' },
      width: 90,
    },
    { title: $t('page.script.version'), field: 'version', width: 80 },
    { title: $t('page.script.description'), field: 'description', minWidth: 150 },
    {
      title: $t('ui.table.updatedAt'),
      field: 'updatedAt',
      formatter: 'formatDateTime',
      width: 140,
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 130,
    },
  ],
};

const exportFetcher = (page: number, pageSize: number) =>
  fetchListScripts(new PaginationQuery({ paging: { page, pageSize } }));

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

const [LogDrawer, logDrawerApi] = useVbenDrawer({
  connectedComponent: ScriptLogDrawer,
});

const [Drawer, drawerApi] = useVbenDrawer({
  connectedComponent: ScriptDrawer,
  onOpenChange(isOpen: boolean) {
    if (!isOpen) {
      gridApi.reload();
    }
  },
});

function handleOpenLogs() {
  logDrawerApi.open();
}

function handleCreate() {
  drawerApi.setData({ create: true });
  drawerApi.open();
}

function handleEdit(row: any) {
  drawerApi.setData({ create: false, row });
  drawerApi.open();
}

async function handleDelete(row: any) {
  try {
    await deleteScript({ ids: [row.id] });
    notification.success({ message: $t('ui.notification.delete_success') });
    await gridApi.reload();
  } catch {
    notification.error({ message: $t('ui.notification.delete_failed') });
  }
}

async function handleEnabledChanged(row: any, checked: boolean) {
  try {
    await updateScript({ id: row.id, values: { isEnabled: checked } });
    notification.success({ message: $t('ui.notification.update_success') });
  } catch {
    notification.error({ message: $t('ui.notification.update_failed') });
  } finally {
    await gridApi.reload();
  }
}

/* -------- 试运行 -------- */
function openTestRun(row: Script) {
  testRunScript.value = row;
  testRunInputRows.value = [];
  testRunResult.value = undefined;
  testRunOpen.value = true;
}

function addTestRunInputRow() {
  testRunInputRows.value.push({ key: '', value: '' });
}

function removeTestRunInputRow(index: number) {
  testRunInputRows.value.splice(index, 1);
}

const contextText = ref('');

async function handleTestRun() {
  if (!testRunScript.value?.id) return;

  const input: Record<string, string> = {};
  for (const row of testRunInputRows.value) {
    const key = row.key.trim();
    if (!key) continue;
    input[key] = row.value;
  }

  testRunLoading.value = true;
  try {
    testRunResult.value = await testRunMutate({ id: testRunScript.value.id, input });
    const context = testRunResult.value?.context || {};
    const entries = Object.entries(context);
    contextText.value =
      entries.length === 0
        ? $t('page.script.testRunNoOutput')
        : JSON.stringify(
            Object.fromEntries(
              entries.map(([k, v]) => {
                try {
                  return [k, JSON.parse(v as string)];
                } catch {
                  return [k, v];
                }
              }),
            ),
            null,
            2,
          );
  } catch (error) {
    console.error('脚本试运行失败', error);
    notification.error({ message: $t('page.script.fetchFailed') });
  } finally {
    testRunLoading.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid :table-title="$t('menu.system.scripts')">
      <template #toolbar-tools>
        <a-button class="mr-2" @click="handleOpenLogs">
          {{ $t('page.script.logTitle') }}
        </a-button>
        <a-button class="mr-2" type="primary" @click="handleCreate">
          {{ $t('page.script.button.create') }}
        </a-button>
        <TableExportButton :fetcher="exportFetcher" :columns="gridOptions.columns" filename="scripts" />
      </template>
      <template #language="{ row }">
        <a-tag :color="scriptLanguageToColor(row.language)">
          {{ scriptLanguageToName(row.language) }}
        </a-tag>
      </template>
      <template #hookPoint="{ row }">
        <a-tag v-if="row.hookPoint" color="geekblue">{{ row.hookPoint }}</a-tag>
        <span v-else class="text-neutral-400">{{ $t('page.script.unmounted') }}</span>
      </template>
      <template #isEnabled="{ row }">
        <a-switch
          :checked="!!row.isEnabled"
          :checked-children="$t('page.script.enabledTag')"
          :un-checked-children="$t('page.script.disabledTag')"
          @change="(checked: any) => handleEnabledChanged(row, !!checked)"
        />
      </template>
      <template #action="{ row }">
        <a-tooltip :title="$t('page.script.testRun')">
          <a-button type="link" :icon="h(LucideCirclePlay)" @click.stop="openTestRun(row)" />
        </a-tooltip>
        <a-button type="link" :icon="h(LucideFilePenLine)" @click.stop="handleEdit(row)" />
        <a-popconfirm
          :cancel-text="$t('ui.button.cancel')"
          :ok-text="$t('ui.button.ok')"
          :title="
            $t('ui.text.do_you_want_delete', { moduleName: $t('page.script.moduleName') })
          "
          @confirm="handleDelete(row)"
        >
          <a-button danger type="link" :icon="h(LucideTrash2)" />
        </a-popconfirm>
      </template>
    </Grid>
    <Drawer />
    <LogDrawer />

    <!-- 试运行对话框 -->
    <a-modal
      v-model:open="testRunOpen"
      :title="$t('page.script.testRunTitle', { name: testRunScript?.name ?? '-' })"
      :width="640"
      :mask-closable="false"
    >
      <a-descriptions :column="2" size="small" class="mb-3">
        <a-descriptions-item :label="$t('page.script.language')">
          {{ scriptLanguageToName(testRunScript?.language) }}
        </a-descriptions-item>
        <a-descriptions-item :label="$t('page.script.hookPoint')">
          {{ testRunScript?.hookPoint || $t('page.script.unmounted') }}
        </a-descriptions-item>
      </a-descriptions>

      <div class="mb-2 text-sm text-neutral-500">
        {{ $t('page.script.testRunInput') }}
      </div>
      <div
        v-for="(row, index) in testRunInputRows"
        :key="index"
        class="mb-2 flex items-center gap-2"
      >
        <a-input
          v-model:value="row.key"
          :placeholder="$t('page.script.testRunInputKey')"
          style="width: 180px"
        />
        <a-input v-model:value="row.value" :placeholder="$t('page.script.testRunInputValue')" />
        <a-button danger type="link" :icon="h(LucideTrash2)" @click="removeTestRunInputRow(index)" />
      </div>
      <a-button class="mb-4 w-full" dashed @click="addTestRunInputRow">
        {{ $t('page.script.testRunAddInput') }}
      </a-button>

      <template v-if="testRunResult">
        <a-alert
          :type="testRunResult.success ? 'success' : 'error'"
          :message="
            testRunResult.success ? $t('page.script.testRunSuccess') : $t('page.script.testRunFailed')
          "
          show-icon
          class="mb-3"
        >
          <template v-if="!testRunResult.success && testRunResult.error" #description>
            <pre class="m-0 whitespace-pre-wrap">{{ testRunResult.error }}</pre>
          </template>
        </a-alert>
        <div class="mb-1 text-sm">
          {{ $t('page.script.testRunOutput') }} ·
          {{ $t('page.script.testRunDuration') }}: {{ testRunResult.durationMs ?? 0 }}ms
        </div>
        <pre class="max-h-56 overflow-auto rounded-md bg-neutral-100 p-3 text-xs dark:bg-neutral-800">{{ contextText }}</pre>
      </template>

      <template #footer>
        <a-button @click="testRunOpen = false">{{ $t('ui.button.cancel') }}</a-button>
        <a-button
          type="primary"
          :disabled="!testRunScript?.id"
          :loading="testRunLoading"
          @click="handleTestRun"
        >
          {{ $t('page.script.testRunRun') }}
        </a-button>
      </template>
    </a-modal>
  </Page>
</template>
