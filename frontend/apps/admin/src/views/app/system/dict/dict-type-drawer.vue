<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import type { dictservicev1_DictTypeI18n as DictTypeI18n } from '#/generated/api/admin/service/v1';

import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';
import { useVbenVxeGrid } from '@vben/plugins/vxe-table';

import { notification } from 'ant-design-vue';

import { useVbenForm, z } from '#/adapter/form';
import { enableBoolList, useDictStore } from '#/stores';
import { useDictViewStore } from '#/views/app/system/dict/dict-view.state';

const dictStore = useDictStore();
const dictViewStore = useDictViewStore();

const data = ref();

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.dict.dictType') })
    : $t('ui.modal.update', { moduleName: $t('page.dict.dictType') }),
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
      fieldName: 'typeCode',
      label: $t('page.dict.typeCode'),
      componentProps: {
        placeholder: $t('ui.placeholder.input'),
        allowClear: true,
      },
      rules: z.string().min(1, { message: $t('ui.formRules.required') }),
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

const gridOptions: VxeGridProps<DictTypeI18n> = {
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
          { description?: string; typeName?: string }
        >;

        for (const lang of language.items || []) {
          const languageCode = lang.languageCode || '';
          if (languageCode === '') {
            continue;
          }

          gridData.push({
            languageCode,
            languageName: lang.nativeName,
            typeName: i18nMap[languageCode]?.typeName ?? '',
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
      width: 95,
    },
    {
      title: $t('page.dict.typeName'),
      field: 'typeName',
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
        ? dictStore.createDictType(values)
        : dictStore.updateDictType(data.value.row.id, values));

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

function hasEditStatus(row: DictTypeI18n) {
  return gridApi.grid?.isEditByRow(row);
}

function editRowEvent(row: DictTypeI18n) {
  gridApi.grid?.setEditRow(row);
}

async function saveRowEvent(row: DictTypeI18n) {
  await gridApi.grid?.clearEdit();

  console.log('onSaveRowEvent', row);

  if (row.languageCode !== undefined) {
    data.value.row.i18n[row.languageCode] = {
      languageCode: row.languageCode,
      typeName: row.typeName,
      description: row.description,
    };
  }

  console.log('data.value.row.i18n', data.value.row.i18n);

  gridApi.setLoading(true);

  try {
    const values = await baseFormApi.getValues();
    console.log(getTitle.value, data.value.row.id, Object.keys(values));
    await dictStore.updateDictType(data.value.row.id, {
      ...values,
      i18n: data.value.row.i18n,
    });

    notification.success({
      message: $t('ui.notification.save_success'),
    });
  } catch (error) {
    console.error(error);
    notification.error({
      message: $t('ui.notification.update_failed'),
    });
    return;
  } finally {
    gridApi.setLoading(false);
    await gridApi.reload();
  }
}

const cancelRowEvent = (_row: DictTypeI18n) => {
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
