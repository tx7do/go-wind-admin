<script lang="ts" setup>
import { computed, ref } from 'vue';

import { useVbenDrawer } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { Modal, notification } from 'ant-design-vue';

import { useVbenForm } from '#/adapter/form';
import { Editor, EditorType } from '#/adapter/component/Editor';
import {
  fetchHookPoints,
  scriptLanguageList,
  useCreateScript,
  useUpdateScript,
} from '#/api';

const { mutateAsync: createScript } = useCreateScript();
const { mutateAsync: updateScript } = useUpdateScript();

const data = ref();
const source = ref('');
const language = ref('LUA');

// 内置骨架与 pristine 判定：源码为空或仍等于骨架时，切语言才自动换骨架
const DEFAULT_LUA_SOURCE = `local log = require "kratos_logger"

function execute()
    local ctx = __get_ctx()
    -- ctx.set("key", value)
    return true
end
`;

const DEFAULT_JS_SOURCE = `function execute() {
    var ctx = __get_ctx();
    // ctx.set("key", value);
    return true;
}
`;

const skeletonFor = (lang: string) => (lang === 'LUA' ? DEFAULT_LUA_SOURCE : DEFAULT_JS_SOURCE);

const isPristineSource = (src: string) =>
  !src.trim() || src === DEFAULT_LUA_SOURCE || src === DEFAULT_JS_SOURCE;

const hookPointOptions = ref<Array<{ value: string; label: string }>>([]);

async function loadHookPointOptions() {
  try {
    const res = await fetchHookPoints();
    hookPointOptions.value = (res.items ?? []).map((hp) => ({
      value: hp.name ?? '',
      label: hp.description ? `${hp.name}（${hp.description}）` : (hp.name ?? ''),
    }));
  } catch (error) {
    console.error('加载钩子点失败', error);
    hookPointOptions.value = [];
  }
}

const getTitle = computed(() =>
  data.value?.create
    ? $t('ui.modal.create', { moduleName: $t('page.script.moduleName') })
    : $t('ui.modal.update', { moduleName: $t('page.script.moduleName') }),
);

const [BaseForm, baseFormApi] = useVbenForm({
  showDefaultActions: false,
  commonConfig: {
    componentProps: {
      class: 'w-full',
    },
  },
  // 语言切换拦截：pristine 换骨架；已有真实代码弹确认，取消则回退
  handleValuesChange: (values: Record<string, any>) => {
    if (!('language' in values)) return;
    const next = values.language as string;
    if (next === language.value) return;

    if (isPristineSource(source.value)) {
      language.value = next;
      source.value = skeletonFor(next);
      return;
    }

    const prev = language.value;
    Modal.confirm({
      title: $t('page.script.switchLanguageTitle'),
      content: $t('page.script.switchLanguageConfirm'),
      okText: $t('ui.button.ok'),
      cancelText: $t('ui.button.cancel'),
      onOk: () => {
        language.value = next;
      },
      onCancel: () => {
        baseFormApi.setValues({ language: prev });
      },
    });
  },
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: $t('page.script.name'),
      componentProps: {
        placeholder: $t('page.script.namePlaceholder'),
        allowClear: true,
      },
      rules: 'required',
    },
    {
      component: 'RadioGroup',
      fieldName: 'language',
      label: $t('page.script.language'),
      defaultValue: 'LUA',
      rules: 'selectRequired',
      componentProps: {
        optionType: 'button',
        buttonStyle: 'solid',
        options: scriptLanguageList,
      },
      help: $t('page.script.languageLuaTip'),
    },
    {
      component: 'AutoComplete',
      fieldName: 'hookPoint',
      label: $t('page.script.hookPoint'),
      componentProps: {
        placeholder: $t('page.script.hookPointPlaceholder'),
        allowClear: true,
        options: hookPointOptions,
      },
    },
    {
      component: 'InputNumber',
      fieldName: 'priority',
      label: $t('page.script.priority'),
      defaultValue: 0,
    },
    {
      component: 'Textarea',
      fieldName: 'description',
      label: $t('page.script.description'),
      componentProps: {
        placeholder: $t('page.script.descriptionPlaceholder'),
        allowClear: true,
      },
    },
    {
      component: 'Switch',
      fieldName: 'critical',
      label: $t('page.script.critical'),
      defaultValue: false,
      help: $t('page.script.criticalTip'),
    },
  ],
});

const [Drawer, drawerApi] = useVbenDrawer({
  onCancel() {
    drawerApi.close();
  },

  async onConfirm() {
    const validate = await baseFormApi.validate();
    if (!validate.valid) {
      return;
    }

    if (!source.value.trim()) {
      notification.error({ message: $t('page.script.requiredSource') });
      return;
    }

    setLoading(true);

    const values = { ...(await baseFormApi.getValues()), source: source.value };

    try {
      await (data.value?.create
        ? createScript(values)
        : updateScript({ id: data.value.row.id, values }));

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
      data.value = drawerApi.getData<Record<string, any>>();
      loadHookPointOptions();

      const row = data.value?.row;
      baseFormApi.setValues(
        row
          ? {
              name: row.name ?? '',
              language: row.language ?? 'LUA',
              hookPoint: row.hookPoint ?? '',
              priority: row.priority ?? 0,
              description: row.description ?? '',
              critical: row.critical ?? false,
            }
          : { name: '', language: 'LUA', hookPoint: '', priority: 0, description: '', critical: false },
      );
      language.value = row?.language ?? 'LUA';
      source.value = row?.source ?? skeletonFor('LUA');

      setLoading(false);
    }
  },
});

function setLoading(loading: boolean) {
  drawerApi.setState({ loading });
}
</script>

<template>
  <Drawer :title="getTitle" class="w-[640px]">
    <BaseForm />
    <div class="mt-2">
      <div class="mb-2 text-sm font-medium">
        <span class="text-red-500">* </span>{{ $t('page.script.source') }}
      </div>
      <div class="overflow-hidden rounded-md border border-neutral-200 dark:border-neutral-700">
        <Editor
          :key="language"
          v-model="source"
          :editor-type="EditorType.CODE"
          :height="320"
          :code-options="{
            language: language === 'LUA' ? 'lua' : 'javascript',
            tabSize: 4,
            autoDetectLanguage: false,
          }"
        />
      </div>
    </div>
  </Drawer>
</template>
