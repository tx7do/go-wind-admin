<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import type {
  dictservicev1_DictEntryI18n as DictEntryI18n,
  dictservicev1_DictType as DictType,
} from '#/generated/api/admin/service/v1';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { notification } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { enableBoolList, useDictStore } from '#/stores';
import {
  getTypeName,
  useDictViewStore,
} from '#/views/app/system/dict/dict-view.state';

const dictStore = useDictStore();
const dictViewStore = useDictViewStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.dict.dictEntry') })
    : $t('ui.modal.update', { moduleName: $t('page.dict.dictEntry') }),
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
      component: 'ApiSelect',
      fieldName: 'typeId',
      label: $t('page.dict.typeId'),
      rules: 'required',
      defaultValue: dictViewStore.currentTypeId,
      componentProps: {
        placeholder: $t('ui.placeholder.select'),
        showSearch: true,
        allowClear: false,
        filterOption: (input: string, option: any) =>
          option.label.toLowerCase().includes(input.toLowerCase()),
        afterFetch: (data: DictType[]) => {
          return data.map((item: DictType) => ({
            label: getTypeName(item),
            value: item.id,
          }));
        },
        api: async () => {
          const result = await dictStore.listDictType(undefined, {
            is_enabled: 'true',
          });
          return result.items;
        },
      },
    },
    {
      component: 'Input',
      fieldName: 'entryValue',
      label: $t('page.dict.entryValue'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: z.string().min(1, { message: $t('ui.formRules.required') }),
    },
    {
      component: 'InputNumber',
      fieldName: 'numericValue',
      label: $t('page.dict.numericValue'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
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
    },
    {
      component: 'RadioGroup',
      fieldName: 'isEnabled',
      label: $t('ui.table.status'),
      defaultValue: true,
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        class: 'flex flex-wrap', // 如果选项过多，可以添加class来自动折叠
        options: enableBoolList,
      },
    },
  ],
});

const gridOptions: VxeGridProps<DictEntryI18n> = {
  height: 'auto',
  stripe: true,
  toolbarConfig: {
    custom: false,
    export: false,
    import: false,
    refresh: false,
    zoom: false,
  },
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
    isCurrent: true,
  },
  editConfig: {
    mode: 'row',
    trigger: 'click',
  },

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        const gridData = [];
        const language = await dictViewStore.fetchLanguageList(
          page.currentPage,
          page.pageSize,
          formValues,
        );

        const i18nMap = (data.value?.row?.i18n ?? {}) as Record<
          string,
          { description?: string; entryLabel?: string }
        >;

        for (const lang of language.items || []) {
          const languageCode = lang.languageCode || '';
          if (languageCode === '') {
            continue;
          }

          gridData.push({
            languageCode,
            languageName: lang.nativeName,
            entryLabel: i18nMap[languageCode]?.entryLabel ?? '',
            description: i18nMap[languageCode]?.description ?? '',
          });
        }

        return { items: gridData, total: gridData.length };
      },
    },
  },

  columns: [
    {
      title: $t('page.dict.languageCode'),
      field: 'languageCode',
      fixed: 'left',
      width: 85,
    },
    {
      title: $t('page.dict.languageName'),
      field: 'languageName',
      width: 85,
    },
    {
      title: $t('page.dict.entryLabel'),
      field: 'entryLabel',
      editRender: { name: 'input' },
    },
    {
      title: $t('ui.table.description'),
      field: 'description',
      editRender: { name: 'input' },
    },
    {
      title: $t('ui.table.action'),
      field: 'action',
      fixed: 'right',
      slots: { default: 'action' },
      width: 140,
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions });

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
        ? dictStore.createDictEntry(values)
        : dictStore.updateDictEntry(data.value.row.id, values));

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
      if (data.value.row === undefined) {
        baseFormApi.setValues({ typeId: dictViewStore.currentTypeId });
      } else {
        data.value.row.typeId = dictViewStore.currentTypeId;
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

function hasEditStatus(row: DictEntryI18n) {
  return gridApi.grid?.isEditByRow(row);
}

function editRowEvent(row: DictEntryI18n) {
  gridApi.grid?.setEditRow(row);
}

async function saveRowEvent(row: DictEntryI18n) {
  await gridApi.grid?.clearEdit();

  if (row.languageCode !== undefined) {
    data.value.row.i18n[row.languageCode] = {
      languageCode: row.languageCode,
      entryLabel: row.entryLabel,
      description: row.description,
    };
  }

  gridApi.setLoading(true);

  try {
    const values = await baseFormApi.getValues();
    await dictStore.updateDictEntry(data.value.row.id, {
      ...values,
      i18n: data.value.row.i18n,
    });

    notification.success({
      message: $t('ui.notification.save_success'),
    });
  } catch {
    notification.error({
      message: $t('ui.notification.update_failed'),
    });
    return;
  } finally {
    gridApi.setLoading(false);
    await gridApi.reload();
  }
}

const cancelRowEvent = (_row: DictEntryI18n) => {
  gridApi.grid?.clearEdit();
};
</script>

<template>
  <Drawer :title="getTitle" class="w-full max-w-[800px]">
    <BaseForm class="mx-0" />
    <Grid>
      <template #action="{ row }">
        <template v-if="hasEditStatus(row)">
          <a-button type="link" @click="saveRowEvent(row)">
            {{ $t('ui.button.save') }}
          </a-button>
          <a-button type="link" @click="cancelRowEvent(row)">
            {{ $t('ui.button.cancel') }}
          </a-button>
        </template>
        <template v-else>
          <a-button type="link" @click="editRowEvent(row)">
            {{ $t('ui.button.edit') }}
          </a-button>
        </template>
      </template>
    </Grid>
  </Drawer>
</template>
