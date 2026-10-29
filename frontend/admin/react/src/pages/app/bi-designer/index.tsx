import "@go-wind/bi-react/style.css";
// React 设计器是「React 容器 + Vue 画布」的桥接壳，画布样式来自 bi-vue
import "@go-wind/bi-vue/style.css";
import { useCallback, useEffect, useRef, useState } from "react";
import { BiDesignerReact } from "@go-wind/bi-react";
import { setBiLocale } from "@go-wind/bi-vue";
import type { BiDashboardSchema } from "@go-wind/bi-core";
import { useI18n } from "@/core/i18n";

/** 与 vue-element 端同一键：设计器保存的仪表盘，仪表盘页读取它渲染 */
const STORAGE_KEY = "bi:schema:default";
/** 落盘防抖：画布拖动会高频触发 change */
const SAVE_DEBOUNCE_MS = 1000;

/**
 * BI 设计器页面（React 端）：BiDesignerReact 桥接壳内挂 Vue 画布，
 * schema 与布局数学同源（bi-core）。变更落 localStorage（MVP 存储；后续接后端 CRUD）。
 */
export default function BiDesignerPage() {
  const { i18n } = useI18n();

  // 桥接壳的 initialSchema 是「挂载时快照」，运行期改 props 不回灌，
  // 所以必须先读完 localStorage 再挂载，否则保存过的仪表盘不会进画布。
  const [ready, setReady] = useState(false);
  const [initialSchema, setInitialSchema] = useState<Partial<BiDashboardSchema> | undefined>();

  useEffect(() => {
    setBiLocale((i18n.language ?? "zh-CN").startsWith("zh") ? "zh-CN" : "en-US");
  }, [i18n.language]);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        setInitialSchema(JSON.parse(saved));
      } catch (error) {
        // 脏数据不阻塞页面：记原始错误，回落到内置演示 schema
        console.error("[bi-designer] 本地 schema 解析失败，改用演示 schema", error);
        setInitialSchema(undefined);
      }
    }
    setReady(true);
  }, []);

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onSchemaChange = useCallback((schema: BiDashboardSchema) => {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(schema));
    }, SAVE_DEBOUNCE_MS);
  }, []);

  useEffect(
    () => () => {
      if (saveTimer.current) clearTimeout(saveTimer.current);
    },
    [],
  );

  return (
    <div style={{ height: "100%", padding: "12px 16px", boxSizing: "border-box" }}>
      {ready && (
        <BiDesignerReact
          demo
          initialSchema={initialSchema}
          onChange={onSchemaChange}
          style={{ height: "100%" }}
        />
      )}
    </div>
  );
}
