import { computed, reactive, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import { $t } from "@/core/i18n";
import { DRAWER_WIDTH } from "@/constants";
import { cloneDeep } from "@/utils";

/**
 * Drawer 表单通用状态管理
 *
 * 收拢所有 Drawer 组件的重复逻辑：
 * - visible / isCreate / currentId
 * - pageLoading / submitLoading
 * - title（根据 isCreate 自动生成）
 * - open / close / resetForm
 * - handleSubmit（校验 → create/update → 成功提示 → emit success）
 *
 * @example
 * ```ts
 * const drawer = useDrawerForm<{ name: string; code: string }>({
 *   moduleKey: "pages.position.moduleName",
 *   defaults: { name: "", code: "" },
 *   createFn: createPosition,
 *   updateFn: (id, values) => updatePosition({ id, values }),
 *   asyncSetup: async () => {
 *     orgUnitTreeData.value = (await fetchListOrgUnits(...)).items || [];
 *   },
 * });
 *
 * function open(data?: { create: boolean; row?: any }) {
 *   drawer.open(data, (row) => {
 *     // 编辑时填充额外字段
 *     Object.assign(formData, row);
 *   });
 * }
 *
 * defineExpose({ open });
 * ```
 */
export function useDrawerForm<T extends Record<string, any>>(options: {
  /** i18n 模块名 key，如 "pages.position.moduleName" */
  moduleKey: string;
  /** 表单默认值 */
  defaults: T;
  /** 新增接口 */
  createFn: (values: T) => Promise<any>;
  /** 更新接口 */
  updateFn: (id: number, values: T) => Promise<any>;
  /** 打开时的异步准备函数（加载树、下拉等），自动管理 pageLoading */
  asyncSetup?: () => Promise<void>;
  /** Drawer 宽度，默认 DRAWER_WIDTH */
  width?: string;
}) {
  // === 状态 ===
  const visible = ref(false);
  const isCreate = ref(true);
  const currentId = ref<number | undefined>();
  const pageLoading = ref(false);
  const submitLoading = ref(false);

  // === 表单数据 ===
  const formData = reactive({ ...options.defaults }) as T;

  // === 标题 ===
  const title = computed(() =>
    isCreate.value
      ? $t("common.modal.create", { moduleName: $t(options.moduleKey as any) })
      : $t("common.modal.update", { moduleName: $t(options.moduleKey as any) })
  );

  // === Drawer 宽度 ===
  const drawerWidth = options.width ?? DRAWER_WIDTH;

  // === 重置表单 ===
  function resetForm() {
    // 深拷贝 defaults 后再赋值，避免：
    // 1) 浅拷贝下嵌套对象/数组与 options.defaults 共享引用，
    //    编辑后通过同一引用反向污染 defaults，导致后续打开带回脏数据；
    // 2) 上次编辑多出来的字段残留在 formData 里（先清空再赋值）。
    const fresh = cloneDeep(options.defaults);
    Object.keys(formData).forEach((k) => {
      delete (formData as any)[k];
    });
    Object.assign(formData, fresh);
  }

  // === 打开 ===
  async function open(data?: { create?: boolean; row?: any }, onEditFill?: (row: any) => void) {
    visible.value = true;
    isCreate.value = data?.create ?? true;
    currentId.value = data?.row?.id;
    resetForm();

    // 异步加载 + 编辑填充
    if (options.asyncSetup || onEditFill) {
      pageLoading.value = true;
      try {
        await options.asyncSetup?.();

        if (!isCreate.value && data?.row) {
          onEditFill?.(data.row);
        }
      } finally {
        pageLoading.value = false;
      }
    }
  }

  // 确保关闭时自动重置表单（兼容 ProModal handleClose 仅置 visible=false 的情况）
  watch(visible, (val) => {
    if (!val) resetForm();
  });

  // === 关闭 ===
  function close() {
    visible.value = false;
  }

  // === 提交 ===
  async function handleSubmit(
    formRef: any,
    onSuccess?: () => void,
    transformValues?: (values: T) => any
  ) {
    if (!formRef) return;

    // 校验失败时 validate 会 reject 字段错误对象（非 false），用二段式区分校验失败与接口失败
    const valid = await formRef.validate().then(
      () => true,
      () => false
    );
    if (!valid) return;

    try {
      submitLoading.value = true;

      const values = transformValues ? transformValues({ ...formData }) : { ...formData };

      if (isCreate.value) {
        await options.createFn(values as T);
        ElMessage.success($t("common.notification.createSuccess"));
      } else {
        await options.updateFn(currentId.value!, values as T);
        ElMessage.success($t("common.notification.updateSuccess"));
      }

      onSuccess?.();
      close();
    } catch {
      // 仅接口失败会进入此分支（校验失败已在上方提前 return）
      ElMessage.error(
        isCreate.value
          ? $t("common.notification.createFailed")
          : $t("common.notification.updateFailed")
      );
    } finally {
      submitLoading.value = false;
    }
  }

  return {
    // 状态
    visible,
    isCreate,
    currentId,
    pageLoading,
    submitLoading,
    formData,
    title,
    drawerWidth,
    // 方法
    open,
    close,
    resetForm,
    handleSubmit,
  };
}
