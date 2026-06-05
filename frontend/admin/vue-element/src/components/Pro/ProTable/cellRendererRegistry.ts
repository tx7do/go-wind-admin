import { defineComponent, h } from "vue";
import type { Component, FunctionalComponent } from "vue";
import { useDateFormat } from "@vueuse/core";
import { ElImage, ElTag, ElSwitch, ElLink, ElIcon, ElTooltip, ElButton } from "element-plus";
import SvgIcon from "@/components/SvgIcon/index.vue";
import { AccessControl } from "@/core/access";
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
    return () => {
      const buttons = props.col.buttons ?? [];
      return h("div", { class: "flex items-center justify-center gap-3" }, [
        buttons.map((btn: any) => {
          const codes = btn.auth ? (Array.isArray(btn.auth) ? btn.auth : [btn.auth]) : undefined;
          const visible = btn.visible?.(props.row) ?? true;
          if (!visible) return null;

          // 获取按钮颜色类型（优先使用 btn.type，其次使用 btn.attrs.type，默认为 primary）
          const btnType = btn.type ?? btn.attrs?.type ?? "primary";
          const iconBtnClass = btn.icon ? `table-icon-btn table-icon-btn--${btnType}` : "table-icon-btn";

          const el = h(ElTooltip, { content: btn.label ?? btn.name, placement: "top" }, () =>
            btn.icon
              ? h(
                  "div",
                  {
                    class: iconBtnClass,
                    onClick: () =>
                      emit("operate", { name: btn.name, row: props.row, $index: props.rowIndex }),
                  },
                  h(SvgIcon, { icon: btn.icon, size: 16 })
                )
              : h(ElButton, { size: "small", link: true, ...btn.attrs }, () => btn.label)
          );

          return h(AccessControl, { codes }, () => el);
        }),
      ]);
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
