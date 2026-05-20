<template>
  <div
    class="rounded bg-[var(--el-bg-color)] border border-[var(--el-border-color)] px-3 py-2 h-full overflow-auto"
    style="min-width: 0;"
  >
    <!-- 表格工具 -->
    <div class="flex flex-col md:flex-row justify-between gap-y-2 mb-2">
      <!-- 左侧工具 -->
      <div class="toolbar-left flex gap-y-2.5 gap-x-2 md:gap-x-3 flex-wrap">
        <template v-for="(btn, index) in toolbarLeftBtn" :key="index">
          <el-button
            v-access:code="btn.perm ?? '*:**'"
            v-bind="btn.attrs"
            :disabled="btn.name === 'delete' && removeIds.length === 0"
            @click="handleToolbar(btn.name)"
          >
            {{ btn.text }}
          </el-button>
        </template>
      </div>
      <!-- 右侧工具 -->
      <div class="toolbar-right flex gap-y-2.5 gap-x-2 md:gap-x-3 flex-wrap">
        <!-- 右侧自定义按钮 -->
        <template v-for="(btn, index) in toolbarRightCustomBtn" :key="'custom-' + index">
          <el-button
            v-access:code="btn.perm ?? '*:**'"
            v-bind="btn.attrs"
            @click="handleToolbar(btn.name)"
          >
            {{ btn.text }}
          </el-button>
        </template>
        <!-- 默认工具栏按钮 -->
        <template v-for="(btn, index) in toolbarRightBtn" :key="index">
          <el-popover v-if="btn.name === 'filter'" placement="bottom" trigger="click">
            <template #reference>
              <el-button v-access:code="btn.perm ?? '*:**'" v-bind="btn.attrs"></el-button>
            </template>
            <el-scrollbar max-height="350px">
              <template v-for="col in columns" :key="col.prop">
                <el-checkbox v-if="col.prop" v-model="col.show" :label="col.label" />
              </template>
            </el-scrollbar>
          </el-popover>
          <el-button
            v-else
            v-access:code="btn.perm ?? '*:**'"
            v-bind="btn.attrs"
            @click="handleToolbar(btn.name)"
          ></el-button>
        </template>
      </div>
    </div>

    <!-- 列表 -->
    <vxe-table
      ref="tableRef"
      v-loading="loading"
      :border="contentConfig.table?.border"
      :stripe="contentConfig.table?.stripe"
      :size="contentConfig.table?.size as any"
      :show-header="contentConfig.table?.showHeader"
      :row-config="{ keyField: pk, isHover: true, isCurrent: true }"
      :tree-config="contentConfig.table?.treeConfig"
      :data="pageData"
      class="w-full vxe-table-fixed-header"
      @checkbox-change="handleSelectionChange"
      @checkbox-all="handleSelectionChange"
      @current-change="handleCurrentRowChange"
    >
      <template v-for="col in columns" :key="col.prop">
        <!-- 复选框列 -->
        <vxe-column v-if="col.type === 'selection'" type="checkbox" width="50" align="center" />
        <!-- 序号列 -->
        <vxe-column v-else-if="col.type === 'index'" type="seq" width="60" align="center" />
        <!-- 普通列 -->
        <vxe-column
          v-else-if="col.show"
          :field="col.prop"
          :title="col.label"
          :width="col.width"
          :min-width="col.minWidth"
          :fixed="col.fixed"
          :align="col.align || 'center'"
          :header-align="col.headerAlign || 'center'"
          :sortable="col.sortable"
          :formatter="col.formatter"
          :tree-node="col.treeNode"
        >
          <template #default="scope">
            <!-- 显示图片 -->
            <template v-if="col.template === 'image'">
              <template v-if="col.prop">
                <template v-if="Array.isArray(scope.row[col.prop])">
                  <template v-for="(item, index) in scope.row[col.prop]" :key="item">
                    <el-image
                      :src="item"
                      :preview-src-list="scope.row[col.prop]"
                      :initial-index="Number(index)"
                      :preview-teleported="true"
                      :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
                    />
                  </template>
                </template>
                <template v-else>
                  <el-image
                    :src="scope.row[col.prop]"
                    :preview-src-list="[scope.row[col.prop]]"
                    :preview-teleported="true"
                    :style="`width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`"
                  />
                </template>
              </template>
            </template>
            <!-- 根据行的selectList属性返回对应列表值 -->
            <template v-else-if="col.template === 'list'">
              <template v-if="col.prop">
                {{ (col.selectList ?? {})[scope.row[col.prop]] }}
              </template>
            </template>
            <!-- 格式化显示链接 -->
            <template v-else-if="col.template === 'url'">
              <template v-if="col.prop">
                <el-link type="primary" :href="scope.row[col.prop]" target="_blank">
                  {{ scope.row[col.prop] }}
                </el-link>
              </template>
            </template>
            <!-- 生成开关组 -->
            <template v-else-if="col.template === 'switch'">
              <template v-if="col.prop">
                <el-switch
                  v-model="scope.row[col.prop]"
                  :active-value="col.activeValue ?? 1"
                  :inactive-value="col.inactiveValue ?? 0"
                  :inline-prompt="true"
                  :active-text="col.activeText ?? ''"
                  :inactive-text="col.inactiveText ?? ''"
                  :validate-event="false"
                  :disabled="!hasButtonPerm(col.prop)"
                  @change="handleModify(col.prop, scope.row[col.prop], scope.row)"
                />
              </template>
            </template>
            <!-- 生成输入框组 -->
            <template v-else-if="col.template === 'input'">
              <template v-if="col.prop">
                <el-input
                  v-model="scope.row[col.prop]"
                  :type="col.inputType ?? 'text'"
                  :disabled="!hasButtonPerm(col.prop)"
                  @blur="handleModify(col.prop, scope.row[col.prop], scope.row)"
                />
              </template>
            </template>
            <!-- 格式化为价格 -->
            <template v-else-if="col.template === 'price'">
              <template v-if="col.prop">
                {{ `${col.priceFormat ?? ""}${scope.row[col.prop]}` }}
              </template>
            </template>
            <!-- 格式化为百分比 -->
            <template v-else-if="col.template === 'percent'">
              <template v-if="col.prop">{{ scope.row[col.prop] }}%</template>
            </template>
            <!-- 显示图标 -->
            <template v-else-if="col.template === 'icon'">
              <template v-if="col.prop">
                <template v-if="scope.row[col.prop].startsWith('el-icon-')">
                  <el-icon>
                    <component :is="scope.row[col.prop].replace('el-icon-', '')" />
                  </el-icon>
                </template>
                <template v-else>
                  <div class="i-svg:{{ scope.row[col.prop] }}" />
                </template>
              </template>
            </template>
            <!-- 格式化时间 -->
            <template v-else-if="col.template === 'date'">
              <template v-if="col.prop">
                {{
                  scope.row[col.prop]
                    ? useDateFormat(scope.row[col.prop], col.dateFormat ?? "YYYY-MM-DD HH:mm:ss")
                        .value
                    : ""
                }}
              </template>
            </template>
            <!-- 列操作栏 -->
            <template v-else-if="col.template === 'tool'">
              <template v-for="(btn, index) in tableToolbarBtn" :key="index">
                <el-button
                  v-if="btn.render === undefined || btn.render(scope.row)"
                  v-access:code="btn.perm ?? '*:**'"
                  v-bind="btn.attrs"
                  @click="
                    handleOperate({
                      name: btn.name,
                      row: scope.row,
                      column: scope.column,
                      $index: scope.rowIndex,
                    })
                  "
                >
                  {{ btn.text }}
                </el-button>
              </template>
            </template>
            <!-- 自定义 -->
            <template v-else-if="col.template === 'custom'">
              <slot :name="col.slotName ?? col.prop" :prop="col.prop" v-bind="scope" />
            </template>
            <!-- 有slotName的列 -->
            <template v-else-if="col.slotName">
              <slot :name="col.slotName" :prop="col.prop" v-bind="scope" />
            </template>
            <!-- 默认显示字段值 -->
            <template v-else>
              <template v-if="col.prop">
                {{ scope.row[col.prop] }}
              </template>
            </template>
          </template>
        </vxe-column>
      </template>
    </vxe-table>

    <!-- 分页 -->
    <div v-if="showPagination" class="pagination-container">
      <el-pagination
        v-model:current-page="pagination.currentPage"
        v-model:page-size="pagination.pageSize"
        :page-sizes="pagination.pageSizes"
        :total="pagination.total"
        :background="pagination.background"
        :layout="pagination.layout || 'total, sizes -> prev, pager, next'"
        :pager-count="5"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 导出弹窗 -->
    <el-dialog
      v-model="exportsModalVisible"
      :align-center="true"
      :title="t('pages.curd.export.title')"
      width="600px"
      style="padding-right: 0"
      @close="handleCloseExportsModal"
    >
      <!-- 滚动 -->
      <el-scrollbar max-height="60vh">
        <!-- 表单 -->
        <el-form
          ref="exportsFormRef"
          style="padding-right: var(--el-dialog-padding-primary)"
          :model="exportsFormData"
          :rules="exportsFormRules"
        >
          <el-form-item :label="t('pages.curd.export.filename')" prop="filename">
            <el-input v-model="exportsFormData.filename" clearable />
          </el-form-item>
          <el-form-item :label="t('pages.curd.export.sheetname')" prop="sheetname">
            <el-input v-model="exportsFormData.sheetname" clearable />
          </el-form-item>
          <el-form-item :label="t('pages.curd.export.origin')" prop="origin">
            <el-select v-model="exportsFormData.origin">
              <el-option
                :label="t('pages.curd.export.originOptions.current')"
                :value="ExportsOriginEnum.CURRENT"
              />
              <el-option
                :label="t('pages.curd.export.originOptions.selected')"
                :value="ExportsOriginEnum.SELECTED"
                :disabled="selectionData.length <= 0"
              />
              <el-option
                :label="t('pages.curd.export.originOptions.remote')"
                :value="ExportsOriginEnum.REMOTE"
                :disabled="contentConfig.exportsAction === undefined"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('pages.curd.export.fields')" prop="fields">
            <el-checkbox-group v-model="exportsFormData.fields">
              <template v-for="col in columns" :key="col.prop">
                <el-checkbox v-if="col.prop" :value="col.prop" :label="col.label" />
              </template>
            </el-checkbox-group>
          </el-form-item>
        </el-form>
      </el-scrollbar>
      <!-- 弹窗底部操作按钮 -->
      <template #footer>
        <div style="padding-right: var(--el-dialog-padding-primary)">
          <el-button type="primary" @click="handleExportsSubmit">
            {{ t("common.button.confirm") }}
          </el-button>
          <el-button @click="handleCloseExportsModal">{{ t("common.button.cancel") }}</el-button>
        </div>
      </template>
    </el-dialog>
    <!-- 导入弹窗 -->
    <el-dialog
      v-model="importModalVisible"
      :align-center="true"
      :title="t('pages.curd.import.title')"
      width="600px"
      style="padding-right: 0"
      @close="handleCloseImportModal"
    >
      <!-- 滚动 -->
      <el-scrollbar max-height="60vh">
        <!-- 表单 -->
        <el-form
          ref="importFormRef"
          style="padding-right: var(--el-dialog-padding-primary)"
          :model="importFormData"
          :rules="importFormRules"
        >
          <el-form-item :label="t('pages.curd.import.file')" prop="files">
            <el-upload
              ref="uploadRef"
              v-model:file-list="importFormData.files"
              class="w-full"
              accept="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet, application/vnd.ms-excel"
              :drag="true"
              :limit="1"
              :auto-upload="false"
              :on-exceed="handleFileExceed"
            >
              <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
              <div class="el-upload__text">
                <span>{{ t("pages.curd.import.dragText") }}</span>
                <em>{{ t("pages.curd.import.clickText") }}</em>
              </div>
              <template #tip>
                <div class="el-upload__tip">
                  {{ t("pages.curd.import.fileTypeTip") }}
                  <el-link
                    v-if="contentConfig.importTemplate"
                    type="primary"
                    icon="download"
                    underline="never"
                    @click="handleDownloadTemplate"
                  >
                    {{ t("pages.curd.import.downloadTemplate") }}
                  </el-link>
                </div>
              </template>
            </el-upload>
          </el-form-item>
        </el-form>
      </el-scrollbar>
      <!-- 弹窗底部操作按钮 -->
      <template #footer>
        <div style="padding-right: var(--el-dialog-padding-primary)">
          <el-button
            type="primary"
            :disabled="importFormData.files.length === 0"
            @click="handleImportSubmit"
          >
            {{ t("common.button.confirm") }}
          </el-button>
          <el-button @click="handleCloseImportModal">{{ t("common.button.cancel") }}</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useAccess } from "@/core/access";
import { useI18n } from "@/i18n";
import { useDateFormat, useThrottleFn } from "@vueuse/core";
import {
  genFileId,
  type FormInstance,
  type FormRules,
  type UploadInstance,
  type UploadRawFile,
  type UploadUserFile,
} from "element-plus";
import { UploadFilled } from "@element-plus/icons-vue";
import ExcelJS from "exceljs";
import { reactive, ref, computed } from "vue";
import type { VxeTableInstance } from "vxe-table";
import type { IContentConfig, IObject, IOperateData } from "./types";
import type { IToolsButton } from "./types";

// 定义接收的属性
const props = defineProps<{ contentConfig: IContentConfig }>();
// 定义自定义事件
const emit = defineEmits<{
  addClick: [];
  exportClick: [];
  searchClick: [];
  toolbarClick: [name: string];
  editClick: [row: IObject];
  filterChange: [data: IObject];
  operateClick: [data: IOperateData];
  rowClick: [row: IObject]; // 新增：行点击事件
}>();

// 国际化
const { t } = useI18n();

// 表格工具栏按钮配置
const config = computed(() => props.contentConfig);
const buttonConfig = reactive<Record<string, IObject>>({
  add: { textKey: "common.button.add", attrs: { icon: "plus", type: "success" }, perm: "add" },
  delete: {
    textKey: "common.button.delete",
    attrs: { icon: "delete", type: "danger" },
    perm: "delete",
  },
  import: { textKey: "common.button.import", attrs: { icon: "upload", type: "" }, perm: "import" },
  export: {
    textKey: "common.button.export",
    attrs: { icon: "download", type: "" },
    perm: "export",
  },
  refresh: {
    textKey: "common.button.refresh",
    attrs: { icon: "refresh", type: "" },
    perm: "*:*:*",
  },
  filter: { textKey: "pages.curd.toolbar.filter", attrs: { icon: "operation", type: "" }, perm: "*:*:*" },
  search: { textKey: "common.button.search", attrs: { icon: "search", type: "" }, perm: "search" },
  imports: {
    textKey: "common.button.batchImport",
    attrs: { icon: "upload", type: "" },
    perm: "imports",
  },
  exports: {
    textKey: "common.button.batchExport",
    attrs: { icon: "download", type: "" },
    perm: "exports",
  },
  view: { textKey: "common.button.view", attrs: { icon: "view", type: "primary" }, perm: "view" },
  edit: { textKey: "common.button.edit", attrs: { icon: "edit", type: "primary" }, perm: "edit" },
  // 右侧工具栏专用按钮
  rightAdd: { textKey: "common.button.add", attrs: { icon: "plus" }, perm: "add" },
});

// 主键
const pk = props.contentConfig.pk ?? "id";
// 权限名称前缀
const authPrefix = computed(() => props.contentConfig.permPrefix);

// 使用权限 Hook
const { hasAccessByCodes } = useAccess();

// 获取按钮权限标识
function getButtonPerm(action: string): string | null {
  // 如果action已经包含完整路径(包含冒号)，则直接使用
  if (action.includes(":")) {
    return action;
  }
  // 否则使用权限前缀组合
  return authPrefix.value ? `${authPrefix.value}:${action}` : null;
}

// 检查是否有权限
function hasButtonPerm(action: string): boolean {
  const perm = getButtonPerm(action);
  // 如果没有设置权限标识，则默认具有权限
  if (!perm) return true;
  return hasAccessByCodes([perm]);
}

// 创建工具栏按钮
function createToolbar(toolbar: Array<string | IToolsButton>, attr = {}) {
  return toolbar.map((item) => {
    const isString = typeof item === "string";
    const textKey = isString ? buttonConfig[item].textKey : item?.textKey;
    return {
      name: isString ? item : item?.name || "",
      text: textKey ? t(textKey) : isString ? buttonConfig[item].text : item?.text,
      attrs: {
        ...attr,
        ...(isString ? buttonConfig[item].attrs : item?.attrs),
      },
      render: isString ? undefined : (item?.render ?? undefined),
      perm: isString
        ? getButtonPerm(buttonConfig[item].perm)
        : item?.perm
          ? getButtonPerm(item.perm as string)
          : "*:*:*",
    };
  });
}

// 左侧工具栏按钮
const toolbarLeftBtn = computed(() => {
  if (!config.value.toolbar || config.value.toolbar.length === 0) return [];
  return createToolbar(config.value.toolbar, {});
});

// 右侧工具栏按钮
const toolbarRightBtn = computed(() => {
  if (!config.value.defaultToolbar || config.value.defaultToolbar.length === 0) return [];
  return createToolbar(config.value.defaultToolbar, { circle: true });
});

// 右侧自定义按钮（在defaultToolbar左侧）
const toolbarRightCustomBtn = computed(() => {
  if (!config.value.toolbarRight || config.value.toolbarRight.length === 0) return [];
  return createToolbar(config.value.toolbarRight, {});
});

// 表格操作工具栏
const tableToolbar = computed(() => {
  const columns = config.value.columns;
  if (!columns || columns.length === 0) return ["edit", "delete"];
  return columns[columns.length - 1].action ?? ["edit", "delete"];
});
const tableToolbarBtn = computed(() =>
  createToolbar(tableToolbar.value, { link: true, size: "small" })
);

// 表格相关
const columns = ref(
  props.contentConfig.columns.map((col) => {
    if (col.initFn) {
      col.initFn(col);
    }
    if (col.show === undefined) {
      col.show = true;
    }
    if (col.prop !== undefined && col.columnKey === undefined && col["column-key"] === undefined) {
      col.columnKey = col.prop;
    }
    if (
      col.type === "selection" &&
      col.reserveSelection === undefined &&
      col["reserve-selection"] === undefined
    ) {
      // 配合表格row-key实现跨页多选
      col.reserveSelection = true;
    }
    return col;
  })
);
// 加载状态
const loading = ref(false);
// 列表数据
const pageData = ref<IObject[]>([]);
// 显示分页
const showPagination = props.contentConfig.pagination !== false;
// 分页配置
const defaultPagination = {
  background: true,
  layout: "total, sizes -> prev, pager, next",
  pageSize: 20,
  pageSizes: [10, 20, 30, 50],
  total: 0,
  currentPage: 1,
};
const pagination = reactive(
  typeof props.contentConfig.pagination === "object"
    ? { ...defaultPagination, ...props.contentConfig.pagination }
    : defaultPagination
);
// 分页相关的请求参数
const request = props.contentConfig.request ?? {
  pageName: "page",
  limitName: "pageSize",
};

const tableRef = ref<VxeTableInstance>();

// 行选中
const selectionData = ref<IObject[]>([]);
// 删除ID集合 用于批量删除
const removeIds = ref<(number | string)[]>([]);
function handleSelectionChange({ records }: { records: IObject[] }) {
  selectionData.value = records;
  removeIds.value = records.map((item) => item[pk]);
}

// 行点击事件处理
function handleCurrentRowChange({ row }: { row: IObject }) {
  emit("rowClick", row);
}

// 获取行选中
function getSelectionData() {
  return selectionData.value;
}

// 刷新
function handleRefresh(isRestart = false) {
  fetchPageData(lastFormData, isRestart);
}

// 删除
function handleDelete(id?: number | string) {
  const ids = [id || removeIds.value].join(",");
  if (!ids) {
    ElMessage.warning(t("pages.curd.message.selectDeleteItems"));
    return;
  }

  ElMessageBox.confirm(t("pages.curd.message.confirmDelete"), t("common.title.confirm"), {
    confirmButtonText: t("common.button.confirm"),
    cancelButtonText: t("common.button.cancel"),
    type: "warning",
  }).then(
    function () {
      if (props.contentConfig.deleteAction) {
        props.contentConfig.deleteAction(ids).then(
          () => {
            ElMessage.success(t("pages.curd.message.deleteSuccess"));
            removeIds.value = [];
            // 清空选中项
            tableRef.value?.clearCheckboxRow();
            handleRefresh(true);
          },
          () => {
            // 交由全局错误处理
          }
        );
      } else {
        ElMessage.error(t("pages.curd.message.noDeleteAction"));
      }
    },
    () => {
      // 用户取消
    }
  );
}

// 导出表单
const fields: string[] = [];
columns.value.forEach((item) => {
  if (item.prop !== undefined) {
    fields.push(item.prop);
  }
});
// 数据来源枚举（使用 const 对象而非 const enum，以便在模板中访问）
const ExportsOriginEnum = {
  CURRENT: "current",
  SELECTED: "selected",
  REMOTE: "remote",
} as const;

// 导出表单数据类型
type ExportsOriginType = (typeof ExportsOriginEnum)[keyof typeof ExportsOriginEnum];

const exportsModalVisible = ref(false);
const exportsFormRef = ref<FormInstance>();
const exportsFormData = reactive<{
  filename: string;
  sheetname: string;
  fields: string[];
  origin: ExportsOriginType;
}>({
  filename: "",
  sheetname: "",
  fields,
  origin: ExportsOriginEnum.CURRENT,
});
const exportsFormRules: FormRules = {
  fields: [{ required: true, message: t("pages.curd.message.selectFields") }],
  origin: [{ required: true, message: t("pages.curd.message.selectOrigin") }],
};
// 打开导出弹窗
function handleOpenExportsModal() {
  exportsModalVisible.value = true;
}
// 导出确认
const handleExportsSubmit = useThrottleFn(() => {
  exportsFormRef.value?.validate((valid: boolean) => {
    if (valid) {
      handleExports();
      handleCloseExportsModal();
    }
  });
}, 3000);
// 关闭导出弹窗
function handleCloseExportsModal() {
  exportsModalVisible.value = false;
  exportsFormRef.value?.resetFields();
  nextTick(() => {
    exportsFormRef.value?.clearValidate();
  });
}
// 导出
function handleExports() {
  const filename = exportsFormData.filename
    ? exportsFormData.filename
    : props.contentConfig.permPrefix || "export";
  const sheetname = exportsFormData.sheetname ? exportsFormData.sheetname : "sheet";
  const workbook = new ExcelJS.Workbook();
  const worksheet = workbook.addWorksheet(sheetname);
  const excelColumns: Partial<ExcelJS.Column>[] = [];
  columns.value.forEach((col) => {
    if (col.label && col.prop && exportsFormData.fields.includes(col.prop)) {
      excelColumns.push({ header: col.label, key: col.prop });
    }
  });
  worksheet.columns = excelColumns;
  if (exportsFormData.origin === ExportsOriginEnum.REMOTE) {
    if (props.contentConfig.exportsAction) {
      props.contentConfig.exportsAction(lastFormData).then((data) => {
        worksheet.addRows(data);
        workbook.xlsx.writeBuffer().then(
          (buffer) => {
            saveXlsx(buffer, filename as string);
          },
          (error) => console.log(error)
        );
      });
    } else {
      ElMessage.error(t("pages.curd.message.noExportsAction"));
    }
  } else {
    worksheet.addRows(
      exportsFormData.origin === ExportsOriginEnum.SELECTED ? selectionData.value : pageData.value
    );
    workbook.xlsx.writeBuffer().then(
      (buffer) => {
        saveXlsx(buffer, filename as string);
      },
      (error) => console.log(error)
    );
  }
}

// 导入表单
let isFileImport = false;
const uploadRef = ref<UploadInstance>();
const importModalVisible = ref(false);
const importFormRef = ref<FormInstance>();
const importFormData = reactive<{
  files: UploadUserFile[];
}>({
  files: [],
});
const importFormRules: FormRules = {
  files: [{ required: true, message: t("pages.curd.message.selectFile") }],
};
// 打开导入弹窗
function handleOpenImportModal(isFile: boolean = false) {
  importModalVisible.value = true;
  isFileImport = isFile;
}
// 覆盖前一个文件
function handleFileExceed(files: File[]) {
  uploadRef.value!.clearFiles();
  const file = files[0] as UploadRawFile;
  file.uid = genFileId();
  uploadRef.value!.handleStart(file);
}
// 下载导入模板
function handleDownloadTemplate() {
  const importTemplate = props.contentConfig.importTemplate;
  if (typeof importTemplate === "string") {
    window.open(importTemplate);
  } else if (typeof importTemplate === "function") {
    importTemplate().then((response) => {
      const fileData = response.data;
      const fileName = decodeURI(
        response.headers["content-disposition"].split(";")[1].split("=")[1]
      );
      saveXlsx(fileData, fileName);
    });
  } else {
    ElMessage.error(t("pages.curd.message.noImportTemplate"));
  }
}
// 导入确认
const handleImportSubmit = useThrottleFn(() => {
  importFormRef.value?.validate((valid: boolean) => {
    if (valid) {
      if (isFileImport) {
        handleImport();
      } else {
        handleImports();
      }
    }
  });
}, 3000);
// 关闭导入弹窗
function handleCloseImportModal() {
  importModalVisible.value = false;
  importFormRef.value?.resetFields();
  nextTick(() => {
    importFormRef.value?.clearValidate();
  });
}
// 文件导入
function handleImport() {
  const importAction = props.contentConfig.importAction;
  if (importAction === undefined) {
    ElMessage.error(t("pages.curd.message.noImportAction"));
    return;
  }
  importAction(importFormData.files[0].raw as File).then(() => {
    ElMessage.success(t("pages.curd.message.importSuccess"));
    handleCloseImportModal();
    handleRefresh(true);
  });
}
// 导入
function handleImports() {
  const importsAction = props.contentConfig.importsAction;
  if (importsAction === undefined) {
    ElMessage.error(t("pages.curd.message.noImportsAction"));
    return;
  }
  // 获取选择的文件
  const file = importFormData.files[0].raw as File;
  // 创建Workbook实例
  const workbook = new ExcelJS.Workbook();
  // 使用FileReader对象来读取文件内容
  const fileReader = new FileReader();
  // 二进制字符串的形式加载文件
  fileReader.readAsArrayBuffer(file);
  fileReader.onload = (ev) => {
    if (ev.target !== null && ev.target.result !== null) {
      const result = ev.target.result as ArrayBuffer;
      // 从 buffer 中加载并解析数据
      workbook.xlsx.load(result).then(
        (workbook) => {
          // 解析后的数据
          const data = [];
          // 获取第一个worksheet内容
          const worksheet = workbook.getWorksheet(1);
          if (worksheet) {
            // 获取第一行的标题
            const fields: any[] = [];
            worksheet.getRow(1).eachCell((cell) => {
              fields.push(cell.value);
            });
            // 遍历工作表的每一行（从第二行开始，因为第一行通常是标题行）
            for (let rowNumber = 2; rowNumber <= worksheet.rowCount; rowNumber++) {
              const rowData: IObject = {};
              const row = worksheet.getRow(rowNumber);
              // 遍历当前行的每个单元格
              row.eachCell((cell, colNumber) => {
                // 获取标题对应的键，并将当前单元格的值存储到相应的属性名中
                rowData[fields[colNumber - 1]] = cell.value;
              });
              // 将当前行的数据对象添加到数组中
              data.push(rowData);
            }
          }
          if (data.length === 0) {
            ElMessage.error(t("pages.curd.message.noDataParsed"));
            return;
          }
          importsAction(data).then(() => {
            ElMessage.success(t("pages.curd.message.importSuccess"));
            handleCloseImportModal();
            handleRefresh(true);
          });
        },
        (error) => console.log(error)
      );
    } else {
      ElMessage.error(t("pages.curd.message.readFileFailed"));
    }
  };
}
// 操作人"
function handleToolbar(name: string) {
  switch (name) {
    case "refresh":
      handleRefresh();
      break;
    case "exports":
      handleOpenExportsModal();
      break;
    case "imports":
      handleOpenImportModal();
      break;
    case "search":
      emit("searchClick");
      break;
    case "add":
    case "rightAdd":
      emit("addClick");
      break;
    case "delete":
      handleDelete();
      break;
    case "import":
      handleOpenImportModal(true);
      break;
    case "export":
      emit("exportClick");
      break;
    default:
      emit("toolbarClick", name);
      break;
  }
}

// 操作人"
function handleOperate(data: IOperateData) {
  switch (data.name) {
    case "delete":
      if (props.contentConfig?.deleteAction) {
        handleDelete(data.row[pk]);
      } else {
        emit("operateClick", data);
      }
      break;
    default:
      emit("operateClick", data);
      break;
  }
}

// 属性修改
function handleModify(field: string, value: boolean | string | number, row: Record<string, any>) {
  if (props.contentConfig.modifyAction) {
    props.contentConfig.modifyAction({
      [pk]: row[pk],
      field,
      value,
    });
  } else {
    ElMessage.error(t("pages.curd.message.noModifyAction"));
  }
}

// 分页切换
function handleSizeChange(value: number) {
  pagination.pageSize = value;
  handleRefresh();
}
function handleCurrentChange(value: number) {
  pagination.currentPage = value;
  handleRefresh();
}

// 远程数据筛选
let filterParams: IObject = {};
function handleFilterChange(newFilters: any) {
  const filters: IObject = {};
  for (const key in newFilters) {
    const col = columns.value.find((col) => {
      return col.columnKey === key || col["column-key"] === key;
    });
    if (col && col.filterJoin !== undefined) {
      filters[key] = newFilters[key].join(col.filterJoin);
    } else {
      filters[key] = newFilters[key];
    }
  }
  filterParams = { ...filterParams, ...filters };
  emit("filterChange", filterParams);
}

// 获取筛选条件
function getFilterParams() {
  return filterParams;
}

// 获取分页数据
let lastFormData = {};
function fetchPageData(formData: IObject = {}, isRestart = false) {
  loading.value = true;
  // 上一次搜索条件
  lastFormData = formData;
  // 重置页码
  if (isRestart) {
    pagination.currentPage = 1;
  }
  props.contentConfig
    .indexAction(
      showPagination
        ? {
            [request.pageName]: pagination.currentPage,
            [request.limitName]: pagination.pageSize,
            ...getFilterParams(),
            ...formData,
          }
        : {
            ...getFilterParams(),
            ...formData,
          }
    )
    .then((data) => {
      if (showPagination) {
        pagination.total = Number((data as any)?.total ?? 0);
        pageData.value = (data as any)?.items ?? [];
      } else {
        pageData.value = Array.isArray(data) ? data : (data?.items ?? (data as any)?.items ?? []);
      }
    })
    .finally(() => {
      loading.value = false;
    });
}
fetchPageData();

// 导出Excel
function exportPageData(formData: IObject = {}) {
  if (props.contentConfig.exportAction) {
    props.contentConfig.exportAction(formData).then((response) => {
      const fileData = response.data;
      const fileName = decodeURI(
        response.headers["content-disposition"].split(";")[1].split("=")[1]
      );
      saveXlsx(fileData, fileName);
    });
  } else {
    ElMessage.error(t("pages.curd.message.noExportAction"));
  }
}

// 浏览器保存文件
function saveXlsx(fileData: any, fileName: string) {
  const fileType =
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;charset=utf-8";

  const blob = new Blob([fileData], { type: fileType });
  const downloadUrl = window.URL.createObjectURL(blob);

  const downloadLink = document.createElement("a");
  downloadLink.href = downloadUrl;
  downloadLink.download = fileName;

  document.body.appendChild(downloadLink);
  downloadLink.click();

  document.body.removeChild(downloadLink);
  window.URL.revokeObjectURL(downloadUrl);
}

// 暴露的属性和方法
defineExpose({ fetchPageData, exportPageData, getFilterParams, getSelectionData, handleRefresh, tableRef });
</script>

<style lang="scss" scoped>
.toolbar-left,
.toolbar-right {
  .el-button {
    margin-right: 0 !important;
    margin-left: 0 !important;
  }
}

// vxe-table 样式优化
:deep(.vxe-table) {
  border-radius: 4px;
  overflow: hidden;

  // 表头样式
  .vxe-table--header-wrapper {
    background-color: var(--el-fill-color-light);

    .vxe-header--column {
      height: 45px !important;
      line-height: 45px !important;

      .vxe-cell {
        height: 45px !important;
        line-height: 45px !important;
      }
    }

    .vxe-cell {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  // 表格主体样式
  .vxe-table--body-wrapper {
    background-color: var(--el-bg-color);
  }

  // 去掉列之间的分割线
  .vxe-body--column,
  .vxe-header--column,
  .vxe-footer--column {
    border-right: none !important;
  }

  // 表格行样式
  .vxe-body--row {
    background-color: var(--el-bg-color);

    .vxe-body--column {
      height: 40px !important;
      line-height: 40px !important;

      .vxe-cell {
        height: 40px !important;
        line-height: 40px !important;
      }
    }

    &.row--hover {
      background-color: var(--el-fill-color-light);
    }

    &.row--current {
      background-color: var(--el-fill-color);
    }
  }

  // 暗黑模式下的悬停效果增强（最高优先级）
  html.dark & .vxe-body--row {
    background-color: #1a1a1a !important;
    transition: background-color 0.2s ease !important;

    &:hover,
    &.row--hover {
      background-color: #2a2a2a !important;

      > td {
        background-color: transparent !important;
      }
    }

    &.row--current {
      background-color: #333333 !important;

      > td {
        background-color: transparent !important;
      }
    }

    &.row--hover.row--current {
      background-color: #333333 !important;
    }
  }

  // 表格单元格
  .vxe-cell {
    color: var(--el-text-color-regular);
  }

  // 表格边框
  &.vxe-table--border-line--inner {
    border-color: var(--el-border-color);
  }
}

// 分页容器样式
.pagination-container {
  padding: 4px 0 0 0;
  background-color: var(--el-bg-color);

  :deep(.el-pagination) {
    display: flex;
    align-items: center;

    // 左侧元素
    .el-pagination__total,
    .el-pagination__sizes {
      margin-right: auto;
    }

    // 右侧元素
    .el-pagination__jump {
      margin-left: auto;
    }
  }
}
</style>
