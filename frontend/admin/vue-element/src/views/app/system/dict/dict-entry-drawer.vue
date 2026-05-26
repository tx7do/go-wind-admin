<template>
  <ElDrawer
    v-model="visible"
    :title="title"
    size="800px"
    :close-on-click-modal="false"
    :append-to-body="true"
    :destroy-on-close="true"
    @close="handleClose"
  >
    <ElForm
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      class="drawer-form"
    >
      <ElFormItem :label="$t('pages.dict.typeId')" prop="typeId">
        <ElSelect
          v-model="formData.typeId"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          style="width: 100%"
        >
          <ElOption
            v-for="item in dictTypeList"
            :key="item.id"
            :label="item.typeName"
            :value="item.id"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('pages.dict.entryValue')" prop="entryValue">
        <ElInput
          v-model="formData.entryValue"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.dict.numericValue')" prop="numericValue">
        <ElInputNumber
          v-model="formData.numericValue"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.sortOrder')" prop="sortOrder">
        <ElInputNumber
          v-model="formData.sortOrder"
          :min="1"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="isEnabled">
        <ElRadioGroup v-model="formData.isEnabled">
          <ElRadioButton
            v-for="item in enableBoolList"
            :key="String(item.value)"
            :value="item.value"
          >
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>
    </ElForm>

    <!-- 多语言配置表格 -->
    <div class="i18n-section">
      <ElDivider content-position="left">{{ $t("pages.dict.i18nConfig") }}</ElDivider>
      <ElTable :data="i18nTableData" border stripe>
        <ElTableColumn prop="languageCode" :label="$t('pages.dict.languageCode')" width="120" />
        <ElTableColumn prop="languageName" :label="$t('pages.dict.languageName')" width="120" />
        <ElTableColumn :label="$t('pages.dict.entryLabel')">
          <template #default="{ row }">
            <ElInput
              v-if="row.editing"
              v-model="row.entryLabel"
              size="small"
              @blur="handleSaveRow(row)"
              @keyup.enter="handleSaveRow(row)"
            />
            <span v-else>{{ row.entryLabel }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('common.table.description')">
          <template #default="{ row }">
            <ElInput
              v-if="row.editing"
              v-model="row.description"
              size="small"
              @blur="handleSaveRow(row)"
              @keyup.enter="handleSaveRow(row)"
            />
            <span v-else>{{ row.description }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('common.table.action')" width="140" fixed="right">
          <template #default="{ row }">
            <ElButton
              v-if="!row.editing"
              type="primary"
              link
              size="small"
              @click="handleEditRow(row)"
            >
              {{ $t("common.button.edit") }}
            </ElButton>
            <template v-else>
              <ElButton type="primary" link size="small" @click="handleSaveRow(row)">
                {{ $t("common.button.save") }}
              </ElButton>
              <ElButton link size="small" @click="handleCancelRow(row)">
                {{ $t("common.button.cancel") }}
              </ElButton>
            </template>
          </template>
        </ElTableColumn>
      </ElTable>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <ElButton @click="handleClose">{{ $t("common.button.cancel") }}</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ $t("common.button.confirm") }}
        </ElButton>
      </div>
    </template>
  </ElDrawer>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { ElMessage } from "element-plus";

import type { dictservicev1_DictType as DictType } from "@/api/generated/admin/service/v1";

import {
  enableBoolList,
  fetchListDictTypes,
  useCreateDictEntry,
  useUpdateDictEntry,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t } from "@/i18n";
import { useDictViewStore } from "@/views/app/system/dict/dict-view.state";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createDictEntry } = useCreateDictEntry();
const { mutateAsync: updateDictEntry } = useUpdateDictEntry();
const dictViewStore = useDictViewStore();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();

// 表单数据
const formData = reactive({
  typeId: undefined as number | undefined,
  entryValue: "",
  numericValue: undefined as number | undefined,
  sortOrder: 1,
  isEnabled: true,
});

// 多语言表格数据
const i18nTableData = ref<any[]>([]);
const i18nDataMap = ref<Record<string, { entryLabel?: string; description?: string }>>({});

// 字典类型列表
const dictTypeList = ref<(DictType & { id: number })[]>([]);

// 表单验证规则
const formRules = {
  typeId: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
  entryValue: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  sortOrder: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  isEnabled: [
    { required: true, message: $t("common.validation.selectRequired"), trigger: "change" },
  ],
};

// 标题
const title = computed(() =>
  isCreate.value
    ? $t("common.modal.create", { moduleName: $t("pages.dict.dictEntry") })
    : $t("common.modal.update", { moduleName: $t("pages.dict.dictEntry") })
);

// 加载字典类型列表
async function loadDictTypeList() {
  try {
    const result = await fetchListDictTypes(
      new PaginationQuery({ formValues: { is_enabled: "true" } })
    );
    dictTypeList.value = (result.items || []).filter(
      (item): item is DictType & { id: number } => item.id != null
    );
  } catch (error) {
    console.error("Failed to load dict type list:", error);
  }
}

// 加载语言列表
async function loadLanguageList() {
  try {
    const language = await dictViewStore.fetchLanguageList(1, 100, {});
    const items = language.items || [];

    i18nTableData.value = items
      .filter((lang: any) => lang.languageCode)
      .map((lang: any) => ({
        languageCode: lang.languageCode,
        languageName: lang.nativeName,
        entryLabel: i18nDataMap.value[lang.languageCode]?.entryLabel || "",
        description: i18nDataMap.value[lang.languageCode]?.description || "",
        editing: false,
      }));
  } catch (error) {
    console.error("Failed to load language list:", error);
  }
}

// 编辑行
function handleEditRow(row: any) {
  row._backup = { entryLabel: row.entryLabel, description: row.description };
  row.editing = true;
}

// 保存行
async function handleSaveRow(row: any) {
  row.editing = false;

  if (row.languageCode) {
    i18nDataMap.value[row.languageCode] = {
      entryLabel: row.entryLabel,
      description: row.description,
    };
  }

  // 如果不是新建，立即保存
  if (!isCreate.value && currentId.value) {
    try {
      submitLoading.value = true;
      await updateDictEntry({
        id: currentId.value!,
        values: {
          ...formData,
          i18n: i18nDataMap.value,
        },
      });
      ElMessage.success($t("common.notification.save_success"));
    } catch {
      ElMessage.error($t("common.notification.update_failed"));
      // 恢复原值
      if (row._backup) {
        row.entryLabel = row._backup.entryLabel;
        row.description = row._backup.description;
      }
    } finally {
      submitLoading.value = false;
    }
  }
}

// 取消编辑
function handleCancelRow(row: any) {
  row.editing = false;
  if (row._backup) {
    row.entryLabel = row._backup.entryLabel;
    row.description = row._backup.description;
    delete row._backup;
  }
}

// 重置表单
function resetForm() {
  Object.assign(formData, {
    typeId: dictViewStore.currentTypeId,
    entryValue: "",
    numericValue: undefined,
    sortOrder: 1,
    isEnabled: true,
  });
  i18nTableData.value = [];
  i18nDataMap.value = {};
  formRef.value?.clearValidate();
}

// 打开抽屉
async function open(data?: { create: boolean; row?: any }) {
  visible.value = true;
  isCreate.value = data?.create ?? true;
  currentId.value = data?.row?.id;

  // 重置表单
  resetForm();

  // 加载数据
  await loadDictTypeList();

  // 编辑时填充数据
  if (data?.row && !isCreate.value) {
    i18nDataMap.value = (data.row.i18n as Record<string, any>) || {};
    Object.assign(formData, {
      typeId: data.row.typeId || dictViewStore.currentTypeId,
      entryValue: data.row.entryValue || "",
      numericValue: data.row.numericValue,
      sortOrder: data.row.sortOrder || 1,
      isEnabled: data.row.isEnabled ?? true,
    });
  } else {
    formData.typeId = dictViewStore.currentTypeId ?? undefined;
  }

  // 加载语言列表
  await loadLanguageList();
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const submitData = {
      ...formData,
      i18n: i18nDataMap.value,
    };

    if (isCreate.value) {
      await createDictEntry(submitData);
      ElMessage.success($t("common.notification.create_success"));
    } else {
      await updateDictEntry({ id: currentId.value!, values: submitData });
      ElMessage.success($t("common.notification.update_success"));
    }

    handleClose();
    emit("success");
  } catch (error) {
    if (error !== false) {
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.create_failed")
          : $t("common.notification.update_failed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 12px;
}

.i18n-section {
  margin-top: 16px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
