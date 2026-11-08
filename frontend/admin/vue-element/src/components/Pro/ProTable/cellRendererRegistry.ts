import { defineComponent, h } from "vue";
import type { Component, FunctionalComponent } from "vue";
import { useDateFormat } from "@vueuse/core";
import {
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElImage,
  ElTag,
  ElSwitch,
  ElLink,
  ElIcon,
  ElButton,
  ElSpace,
} from "element-plus";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { useAccess } from "@/core/access";
import { $t } from "@/core/i18n";
import type { ProTableColumn } from "./types";

export interface CellRendererContext {
  col: ProTableColumn;
  row: any;
  field: string;
  rowIndex: number;
}

function getTagType(
  value: any,
  col: ProTableColumn
): "primary" | "success" | "warning" | "danger" | "info" {
  // 优先使用 tagTypeMap（根据值映射颜色）
  if (col.tagTypeMap && value != null) {
    return col.tagTypeMap[value] ?? "info";
  }

  // 其次使用固定 tagType
  if (col.tagType) return col.tagType as any;

  // 默认：有值为 success，无值为 danger
  return value ? "success" : "danger";
}

function getIconBtnVariant(btn: Record<string, any>): string {
  const type = btn.attrs?.type;
  if (type === "danger") return "danger";
  if (type === "success") return "success";
  if (type === "warning") return "warning";
  return "primary";
}

const ImageCell: FunctionalComponent<CellRendererContext> = ({ col, row, field }) => {
  if (!field) return null;
  const val = row[field];
  const style = `width: ${col.imageWidth ?? 40}px; height: ${col.imageHeight ?? 40}px`;

  if (Array.isArray(val)) {
    return h("template", [
      val.map((item: string, idx: number) =>
        h(ElImage, {
          src: item,
          previewSrcList: val,
          initialIndex: idx,
          previewTeleported: true,
          style,
        })
      ),
    ]);
  }

  return h(ElImage, {
    src: val,
    previewSrcList: [val],
    previewTeleported: true,
    style,
  });
};

const TagCell: FunctionalComponent<CellRendererContext> = ({ col, row, field }) => {
  const value = row[field];
  const tagType = getTagType(value, col);

  // 获取标签文本
  const label = col.labelMap?.[value] ?? value;

  // plain 效果：白底 + 深色文字 + 彩色边框，亮暗模式下都有高对比度
  return h(ElTag, { type: tagType, effect: col.tagEffect ?? "plain" }, () => label);
};

const SwitchCell = defineComponent({
  props: ["col", "row", "field", "rowIndex"],
  emits: ["modify"],
  setup(props, { emit }) {
    return () => {
      if (!props.field) return null;
      return h(ElSwitch, {
        modelValue: props.row[props.field],
        "onUpdate:modelValue": (val) => {
          emit("modify", { row: props.row, field: props.field, value: val });
        },
        activeValue: props.col.activeValue ?? 1,
        inactiveValue: props.col.inactiveValue ?? 0,
        inlinePrompt: false, // 不显示文本，只显示开关图标
      });
    };
  },
});

const DateCell: FunctionalComponent<CellRendererContext> = ({ col, row, field }) => {
  return row[field] ? useDateFormat(row[field], col.dateFormat ?? "YYYY-MM-DD HH:mm:ss").value : "";
};

const LinkCell: FunctionalComponent<CellRendererContext> = ({ row, field }) => {
  return h(ElLink, { type: "primary", href: row[field], target: "_blank" }, () => row[field]);
};

const PriceCell: FunctionalComponent<CellRendererContext> = ({ col, row, field }) => {
  return `${col.pricePrefix ?? ""}${row[field]}`;
};

const PercentCell: FunctionalComponent<CellRendererContext> = ({ row, field }) => {
  return `${row[field]}%`;
};

const IconCell: FunctionalComponent<CellRendererContext> = ({ row, field }) => {
  return h(ElIcon, () => h(row[field]));
};

const ToolCell = defineComponent({
  props: ["col", "row", "field", "rowIndex"],
  emits: ["operate"],
  setup(props, { emit }) {
    const { hasAccess } = useAccess();
    return () => {
      const declared = props.col.buttons ?? [];
      // 与渲染口径一致地判活：visible() + 权限码（AccessControl 的同一套 hasAccess），
      // 只对判活后的按钮数做溢出判定，权限隐藏后不足 3 个就不出「更多」。
      const buttons = declared.filter((btn: any) => {
        const visible = btn.visible?.(props.row) ?? true;
        if (!visible) return false;
        const codes = btn.auth ? (Array.isArray(btn.auth) ? btn.auth : [btn.auth]) : undefined;
        return !codes || hasAccess(codes);
      });

      // 操作列定式：>3 个操作不堆行内，行内只留前 3 个，「更多」下拉收其余（行内图标+文本；
      // 触发器只留图标，文案由原生 title 兜底——EP 的 ElTooltip 与 ElDropdown 默认插槽的
      // ElOnlyChild 单子节点约束嵌套不稳，不冒险）。
      const inline = buttons.slice(0, 3);
      const overflow = buttons.slice(3);

      const renderButton = (btn: any) => {
        // 统一为 link 文本按钮：type=primary（编辑）/ danger（删除），link 弱化视觉。
        // 图标 + 文字形式，比纯图标更直观，避免暗色下高亮图标凌乱。
        const btnType = btn.type ?? btn.attrs?.type ?? "primary";
        const label = btn.label ?? btn.name;
        const iconNode = btn.icon ? h(SvgIcon, { icon: btn.icon, size: 14 }) : null;
        return h(
          ElButton,
          {
            link: true,
            type: btnType,
            size: "small",
            onClick: () =>
              emit("operate", { name: btn.name, row: props.row, $index: props.rowIndex }),
          },
          () => [iconNode, label]
        );
      };

      const items = inline.map(renderButton);

      if (overflow.length > 0) {
        items.push(
          h(
            ElDropdown,
            {
              trigger: "click",
              onCommand: (name: string) =>
                emit("operate", { name, row: props.row, $index: props.rowIndex }),
            },
            {
              // 触发器独占默认插槽（ElOnlyChild 要求恰好一个子节点），菜单走具名 #dropdown 插槽
              default: () =>
                h(
                  ElButton,
                  {
                    link: true,
                    type: "primary",
                    size: "small",
                    title: $t("common.button.more"),
                  },
                  () => [h(SvgIcon, { icon: "lucide:chevron-down", size: 14 })]
                ),
              dropdown: () =>
                h(ElDropdownMenu, null, () =>
                  overflow.map((btn: any) => {
                    const label = btn.label ?? btn.name;
                    const iconNode = btn.icon ? h(SvgIcon, { icon: btn.icon, size: 14 }) : null;
                    const danger = (btn.type ?? btn.attrs?.type) === "danger";
                    return h(
                      ElDropdownItem,
                      {
                        command: btn.name,
                        class: danger ? "pro-table__more-item--danger" : undefined,
                      },
                      () => [iconNode, label]
                    );
                  })
                ),
            }
          )
        );
      }

      return h(ElSpace, { class: "pro-table__tool-cell" }, () => items);
    };
  },
});

const TextCell: FunctionalComponent<CellRendererContext> = ({ row, field }) =>
  field ? row[field] : "";

type CellRenderer = Component;
const registry = new Map<string, CellRenderer>();

export function registerCellRenderer(type: string, renderer: CellRenderer) {
  registry.set(type, renderer);
}

export function getCellRenderer(type: string) {
  return registry.get(type);
}

registerCellRenderer("image", ImageCell);
registerCellRenderer("tag", TagCell);
registerCellRenderer("switch", SwitchCell);
registerCellRenderer("date", DateCell);
registerCellRenderer("link", LinkCell);
registerCellRenderer("price", PriceCell);
registerCellRenderer("percent", PercentCell);
registerCellRenderer("icon", IconCell);
registerCellRenderer("tool", ToolCell);
registerCellRenderer("text", TextCell);
