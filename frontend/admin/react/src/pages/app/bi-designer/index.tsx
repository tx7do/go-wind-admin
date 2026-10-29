import "@go-wind/bi-react/style.css";
// React 设计器是「React 容器 + Vue 画布」的桥接壳，画布样式来自 bi-vue
import "@go-wind/bi-vue/style.css";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from 'react-router-dom';
import { Button, Tooltip } from 'antd';
import {
  FullscreenExitOutlined,
  FullscreenOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { BiDesignerReact } from "@go-wind/bi-react";
import { setBiLocale } from "@go-wind/bi-vue";
import type { BiDashboardSchema } from "@go-wind/bi-core";
import { fetchBiDashboard, useSaveBiDashboardPayload } from '@/api/hooks/bi-dashboard';
import { useI18n } from '@/core/i18n';
import { usePageFullscreen } from '@/hooks/usePageFullscreen';

/** 本地草稿键（无 id 模式），与仪表盘页共用 */
const DRAFT_KEY = "bi:schema:default";
/** 自动保存防抖 */
const SAVE_DEBOUNCE_MS = 1200;

type SaveStatus = "idle" | "saving" | "saved" | "conflict";

/**
 * BI 独立设计器页（React 端）：
 * - ?id=N：加载库中仪表盘，画布改动防抖自动保存回写（version 乐观锁，409 后自动刷新版本）；
 * - 无 id：本地草稿（localStorage），保存落草稿，不进服务端
 */
export default function BiDesignerPage() {
  const { t, i18n } = useI18n();
  const [searchParams] = useSearchParams();
  const idParam = searchParams.get('id');
  const dashboardId = idParam && Number(idParam) > 0 ? Number(idParam) : undefined;
  const { mutateAsync: savePayload } = useSaveBiDashboardPayload();
  const { rootRef, isFullscreen, toggle } = usePageFullscreen<HTMLDivElement>();

  // 桥接壳的 initialSchema 是「挂载时快照」，运行期改 props 不回灌，
  // 所以必须先读完数据源再挂载（后端行 / 本地草稿），否则保存过的仪表盘不会进画布。
  const [ready, setReady] = useState(false);
  const [initialSchema, setInitialSchema] = useState<Partial<BiDashboardSchema> | undefined>();
  const [targetName, setTargetName] = useState("");
  const [saveStatus, setSaveStatus] = useState<SaveStatus>("idle");
  /** 内容基线：首帧回显（normalizeSchema 会补默认值，与服务端原文永不字符串相等）
   *  定为画布首次 emit 的规范化串；此后内容与基线相同（如 select 类无内容变更）跳过保存 */
  const baselineRef = useRef<string | null>(null);
  /** 乐观锁本地持有版本：保存成功后 +1，409 后从服务端刷新 */
  const versionRef = useRef(1);
  const conflictToasted = useRef(false);

  useEffect(() => {
    setBiLocale((i18n.language ?? "zh-CN").startsWith("zh") ? "zh-CN" : "en-US");
  }, [i18n.language]);

  useEffect(() => {
    if (dashboardId) {
      fetchBiDashboard(dashboardId)
        .then((row) => {
          setTargetName(row.name || "");
          versionRef.current = row.version ?? 1;
          if (row.payload) {
            setInitialSchema(JSON.parse(row.payload));
          }
        })
        .catch((error: unknown) => {
          console.error("[bi-designer] 加载仪表盘失败，切换为本地草稿", error);
          loadDraft();
        })
        .finally(() => setReady(true));
      return;
    }
    loadDraft();
    setReady(true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardId]);

  function loadDraft() {
    const saved = localStorage.getItem(DRAFT_KEY);
    if (saved) {
      try {
        setInitialSchema(JSON.parse(saved));
      } catch (error) {
        // 脏数据不阻塞页面：记原始错误，回落到内置演示 schema
        console.error("[bi-designer] 本地草稿解析失败，改用演示 schema", error);
        setInitialSchema(undefined);
      }
    }
  }

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clearTimer = () => {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = null;
  };

  const saveToServer = useCallback(
    (schema: BiDashboardSchema, payloadStr: string) => {
      setSaveStatus("saving");
      savePayload({
        id: dashboardId!,
        version: versionRef.current,
        payload: payloadStr,
        chartCount: schema.charts?.length ?? 0,
        themeKey: schema.theme?.key ?? "",
      })
        .then(() => {
          versionRef.current += 1;
          baselineRef.current = payloadStr;
          setSaveStatus("saved");
        })
        .catch((error: Error) => {
          const msg = String(error?.message || "");
          if (
            msg.includes("modified by another session") ||
            msg.includes("CONFLICT")
          ) {
            // 他人已保存：刷新版本后继续自动保存（后续编辑会用新版本落盘）
            setSaveStatus("conflict");
            if (!conflictToasted.current) {
              conflictToasted.current = true;
              // message 弹层由 hook 的 options 里没有 App 上下文，这里用 alert-free 方案：状态条已示警
            }
            fetchBiDashboard(dashboardId!)
              .then((row) => {
                versionRef.current = row.version ?? versionRef.current;
                setSaveStatus("idle");
              })
              .catch((refreshError: unknown) =>
                console.error("[bi-designer] 冲突后刷新版本失败", refreshError),
              );
          } else {
            console.error("[bi-designer] 自动保存失败", error);
            setSaveStatus("idle");
          }
        });
    },
    [dashboardId, savePayload],
  );

  const onSchemaChange = useCallback(
    (schema: BiDashboardSchema) => {
      const payloadStr = JSON.stringify(schema);
      if (dashboardId) {
        // 首帧 emit = 加载内容的规范化形态，定为基线（原文与规范化串字符串永不相等，不能当基线）
        if (baselineRef.current === null) {
          baselineRef.current = payloadStr;
          return;
        }
        if (payloadStr === baselineRef.current) return;
        clearTimer();
        saveTimer.current = setTimeout(
          () => saveToServer(schema, payloadStr),
          SAVE_DEBOUNCE_MS,
        );
      } else {
        clearTimer();
        saveTimer.current = setTimeout(() => {
          localStorage.setItem(DRAFT_KEY, payloadStr);
        }, SAVE_DEBOUNCE_MS);
      }
    },
    [dashboardId, saveToServer],
  );

  useEffect(
    () => () => {
      clearTimer();
    },
    [],
  );

  const statusNode =
    dashboardId && saveStatus !== "idle" ? (
      <span
        style={{
          fontSize: 12,
          color:
            saveStatus === "saved"
              ? "#52c41a"
              : saveStatus === "conflict"
                ? "#ff4d4f"
                : "rgba(0,0,0,0.45)",
        }}
      >
        {saveStatus === "saving" && t("statusSaving")}
        {saveStatus === "saved" && t("statusSaved")}
        {saveStatus === "conflict" && t("statusConflict")}
      </span>
    ) : null;

  const targetLabel = dashboardId
    ? targetName || `#${dashboardId}`
    : t("localDraft");

  return (
    <div
      ref={rootRef}
      style={{
        height: "100%",
        padding: "12px 16px",
        boxSizing: "border-box",
        display: "flex",
        flexDirection: "column",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          flexShrink: 0,
          paddingBottom: 8,
        }}
      >
        <span style={{ fontSize: 15, fontWeight: 600 }}>{targetLabel}</span>
        {statusNode}
        <span style={{ flex: 1 }} />
        {dashboardId && (
          <Button
            size="small"
            icon={<ReloadOutlined />}
            onClick={() => {
              window.location.href = `/bi/dashboard?id=${dashboardId}`;
            }}
          >
            {t("preview")}
          </Button>
        )}
        <Tooltip title={isFullscreen ? t("exitFullscreen") : t("fullscreen")}>
          <Button
            size="small"
            icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            onClick={toggle}
          />
        </Tooltip>
      </div>

      <div style={{ flex: 1, minHeight: 0 }}>
        {ready && (
          <BiDesignerReact
            // key 带数据源：换目标强制重挂载（initialSchema 是挂载快照，不回灌）
            key={`${dashboardId ?? "draft"}`}
            demo={!initialSchema}
            initialSchema={initialSchema}
            onChange={onSchemaChange}
            style={{ height: "100%" }}
          />
        )}
      </div>
    </div>
  );
}
