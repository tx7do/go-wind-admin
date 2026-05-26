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
      <!-- 基本信息 -->
      <ElDivider content-position="left">{{ $t("common.section.basic") }}</ElDivider>

      <ElFormItem :label="$t('pages.menu.type')" prop="type">
        <ElRadioGroup v-model="formData.type">
          <ElRadioButton v-for="item in menuTypeList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem :label="$t('pages.menu.title')" prop="meta.title">
        <ElInput
          v-model="formData.meta.title"
          :placeholder="$t('common.placeholder.input')"
          clearable
        >
          <template v-if="titleSuffix" #append>
            {{ titleSuffix }}
          </template>
        </ElInput>
      </ElFormItem>

      <ElFormItem :label="$t('pages.menu.parentId')" prop="parentId">
        <ElTreeSelect
          v-model="formData.parentId"
          :data="menuTreeData"
          :props="{ label: 'meta.title', value: 'id', children: 'children' } as any"
          :placeholder="$t('common.placeholder.select')"
          filterable
          clearable
          check-strictly
          default-expand-all
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem :label="$t('pages.menu.order')" prop="meta.order">
        <ElInputNumber
          v-model="formData.meta.order"
          :min="0"
          :placeholder="$t('common.placeholder.input')"
          style="width: 100%"
        />
      </ElFormItem>

      <ElFormItem v-if="!isButton(formData.type)" :label="$t('pages.menu.icon')" prop="meta.icon">
        <IconSelect v-model="formData.meta.icon" prefix="lucide" />
      </ElFormItem>

      <ElFormItem v-if="!isButton(formData.type)" :label="$t('pages.menu.path')" prop="path">
        <ElInput v-model="formData.path" :placeholder="$t('common.placeholder.input')" clearable />
      </ElFormItem>

      <ElFormItem v-if="isMenu(formData.type)" :label="$t('pages.menu.component')" prop="component">
        <ElInput
          v-model="formData.component"
          :placeholder="$t('common.placeholder.input')"
          clearable
        />
      </ElFormItem>

      <ElFormItem
        v-if="!isCatalog(formData.type)"
        :label="$t('pages.menu.authority')"
        prop="meta.authority"
      >
        <ElSelect
          v-model="formData.meta.authority"
          :placeholder="$t('common.placeholder.input')"
          multiple
          filterable
          allow-create
          default-first-option
          style="width: 100%"
        >
          <ElOption
            v-for="auth in formData.meta.authority"
            :key="auth"
            :label="auth"
            :value="auth"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="$t('common.table.status')" prop="status">
        <ElRadioGroup v-model="formData.status">
          <ElRadioButton v-for="item in statusList" :key="item.value" :value="item.value">
            {{ item.label }}
          </ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <!-- 高级设置 -->
      <ElDivider v-if="!['BUTTON', 'LINK'].includes(formData.type)" content-position="left">
        {{ $t("pages.menu.advancedSettings") }}
      </ElDivider>

      <ElFormItem
        v-if="isMenu(formData.type)"
        :label="$t('pages.menu.keepAlive')"
        prop="meta.keepAlive"
      >
        <ElCheckbox v-model="formData.meta.keepAlive">
          {{ $t("pages.menu.keepAlive") }}
        </ElCheckbox>
      </ElFormItem>

      <ElFormItem
        v-if="['EMBEDDED', 'MENU'].includes(formData.type)"
        :label="$t('pages.menu.affixTab')"
        prop="meta.affixTab"
      >
        <ElCheckbox v-model="formData.meta.affixTab">
          {{ $t("pages.menu.affixTab") }}
        </ElCheckbox>
      </ElFormItem>

      <ElFormItem
        v-if="!['BUTTON'].includes(formData.type)"
        :label="$t('pages.menu.hideInMenu')"
        prop="meta.hideInMenu"
      >
        <ElCheckbox v-model="formData.meta.hideInMenu">
          {{ $t("pages.menu.hideInMenu") }}
        </ElCheckbox>
      </ElFormItem>

      <ElFormItem
        v-if="['CATALOG', 'MENU'].includes(formData.type)"
        :label="$t('pages.menu.hideChildrenInMenu')"
        prop="meta.hideChildrenInMenu"
      >
        <ElCheckbox v-model="formData.meta.hideChildrenInMenu">
          {{ $t("pages.menu.hideChildrenInMenu") }}
        </ElCheckbox>
      </ElFormItem>

      <ElFormItem
        v-if="!['BUTTON', 'LINK'].includes(formData.type)"
        :label="$t('pages.menu.hideInBreadcrumb')"
        prop="meta.hideInBreadcrumb"
      >
        <ElCheckbox v-model="formData.meta.hideInBreadcrumb">
          {{ $t("pages.menu.hideInBreadcrumb") }}
        </ElCheckbox>
      </ElFormItem>

      <ElFormItem
        v-if="!['BUTTON', 'LINK'].includes(formData.type)"
        :label="$t('pages.menu.hideInTab')"
        prop="meta.hideInTab"
      >
        <ElCheckbox v-model="formData.meta.hideInTab">
          {{ $t("pages.menu.hideInTab") }}
        </ElCheckbox>
      </ElFormItem>
    </ElForm>

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
import { computed, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus";

import IconSelect from "@/components/IconSelect/index.vue";
import {
  buildMenuTree,
  isButton,
  isCatalog,
  isMenu,
  menuTypeList,
  statusList,
  fetchListMenus,
  useCreateMenu,
  useUpdateMenu,
} from "@/api/composables";
import { PaginationQuery } from "@/core/transport/rest";
import { $t, $te } from "@/i18n";

const emit = defineEmits<{
  success: [];
}>();

const { mutateAsync: createMenu } = useCreateMenu();
const { mutateAsync: updateMenu } = useUpdateMenu();

const visible = ref(false);
const submitLoading = ref(false);
const isCreate = ref(true);
const currentId = ref<number>();
const formRef = ref();
const menuTreeData = ref<any[]>([]);
const titleSuffix = ref("");

// 表单数据
const formData = reactive({
  type: "MENU",
  parentId: 0,
  path: "",
  component: "BasicLayout",
  meta: {
    title: "",
    icon: "",
    order: 1,
    authority: [] as string[],
    keepAlive: false,
    affixTab: false,
    hideInMenu: false,
    hideChildrenInMenu: false,
    hideInBreadcrumb: false,
    hideInTab: false,
  },
  status: "ON",
});

// 表单验证规则
const formRules = {
  "meta.title": [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  path: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  component: [{ required: true, message: $t("common.validation.required"), trigger: "blur" }],
  status: [{ required: true, message: $t("common.validation.selectRequired"), trigger: "change" }],
};

// 标题
const title = computed(() =>
  isCreate.value ? $t("pages.menu.button.create") : $t("pages.menu.button.update")
);

// 监听标题变化，显示多语言翻译
watch(
  () => formData.meta.title,
  (newVal) => {
    if (newVal && $te(newVal)) {
      titleSuffix.value = $t(newVal);
    } else {
      titleSuffix.value = "";
    }
  }
);

// 打开抽屉
async function open(row?: any) {
  visible.value = true;

  // 加载菜单树数据
  await loadMenuTree();

  if (row) {
    // 编辑模式
    isCreate.value = false;
    currentId.value = row.id;
    Object.assign(formData, row);
    // 处理 meta 字段
    if (row.meta) {
      Object.assign(formData.meta, row.meta);
    }
    // 初始化 titleSuffix
    if (row.meta?.title && $te(row.meta.title)) {
      titleSuffix.value = $t(row.meta.title);
    }
  } else {
    // 创建模式
    isCreate.value = true;
    currentId.value = undefined;
    resetForm();
  }
}

// 关闭抽屉
function handleClose() {
  visible.value = false;
  resetForm();
}

// 重置表单
function resetForm() {
  formData.type = "MENU";
  formData.parentId = 0;
  formData.path = "";
  formData.component = "BasicLayout";
  formData.meta = {
    title: "",
    icon: "",
    order: 1,
    authority: [],
    keepAlive: false,
    affixTab: false,
    hideInMenu: false,
    hideChildrenInMenu: false,
    hideInBreadcrumb: false,
    hideInTab: false,
  };
  formData.status = "ON";
  titleSuffix.value = "";

  formRef.value?.clearValidate();
}

// 加载菜单树
async function loadMenuTree() {
  try {
    const result = await fetchListMenus(new PaginationQuery({ formValues: { status: "ON" } }));
    menuTreeData.value = buildMenuTree(result.items || []);
  } catch (error) {
    console.error("加载菜单树失败", error);
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return;

  try {
    await formRef.value.validate();
    submitLoading.value = true;

    const values = { ...formData };

    if (isCreate.value) {
      await createMenu(values);
      ElMessage.success($t("common.notification.createSuccess"));
    } else {
      await updateMenu({ id: currentId.value!, values });
      ElMessage.success($t("common.notification.updateSuccess"));
    }

    emit("success");
    handleClose();
  } catch (error) {
    if (error !== false) {
      // 不是验证错误
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    }
  } finally {
    submitLoading.value = false;
  }
}

// 暴露方法给父组件
defineExpose({
  open,
});
</script>

<style lang="scss" scoped>
.drawer-form {
  padding-right: 10px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
