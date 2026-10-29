import "@go-wind/bi-react/style.css";
import { useEffect, useRef, useState } from 'react';
import { BiProvider, BiDashboard } from '@go-wind/bi-react';
import { layoutCharts } from '@go-wind/bi-core';
import type { BiDashboardSchema } from '@go-wind/bi-core';

/** 与 /bi/designer 共用：设计器落盘、仪表盘读取 */
const BI_SCHEMA_STORAGE_KEY = "bi:schema:default";

const SCHEMA: Partial<BiDashboardSchema> = {
  name: "default",
  linkage: { enabled: true },
  theme: { key: "tech-blue" },
  datasets: [
    {
      id: "ds_month",
      name: "月度汇总",
      type: "static",
      rows: Array.from({ length: 12 }, (_, i) => ({
        month: `${i + 1}月`,
        sales: Math.round(80 + Math.sin(i / 2) * 60 + i * 8),
        target: 160,
      })),
    },
    {
      id: "ds_orders",
      name: "订单明细",
      type: "static",
      rows: Array.from({ length: 48 }, (_, i) => ({
        city: ["北京", "上海", "广州", "深圳", "杭州", "成都"][i % 6],
        category: ["硬件", "软件", "服务"][i % 3],
        amount: Math.round(50 + Math.random() * 450),
      })),
    },
  ],
  charts: [
    {
      id: "c_bar", title: "城市订单 Top6", datasetId: "ds_orders", type: "bar",
      x: 0, y: 0, w: 6, h: 4,
      encoding: { dimension: "city", measure: "amount" },
      aggregation: "sum", sort: { by: "measure", order: "desc", topN: 6 },
    },
    {
      id: "c_wide", title: "销售 vs 目标", datasetId: "ds_month", type: "bar",
      x: 6, y: 0, w: 6, h: 4,
      encoding: { dimension: "month", measures: ["sales", "target"] },
      style: { showLegend: true },
    },
    {
      id: "c_pie", title: "品类占比", datasetId: "ds_orders", type: "donut",
      x: 0, y: 4, w: 6, h: 4,
      encoding: { dimension: "category", measure: "amount" }, aggregation: "sum",
    },
    {
      id: "c_area", title: "销售趋势", datasetId: "ds_month", type: "area",
      x: 6, y: 4, w: 6, h: 4,
      encoding: { dimension: "month", measure: "sales" },
    },
  ] as BiDashboardSchema["charts"],
};

/** 内容高度（设计单位）：直接复用渲染器的布局函数，避免公式重复走样 */
function contentHeightOf(value: Partial<BiDashboardSchema>): number {
  return layoutCharts(value.charts ?? [], value.grid, 1920).height;
}

export default function BiDashboardPage() {
  // 优先渲染设计器保存的 schema（与 /bi/designer 共用这个 localStorage 键），
  // 没有保存过才回落到内置演示。读取放在首帧后的 effect 里：SSR/首屏无需等待。
  const [schema, setSchema] = useState<Partial<BiDashboardSchema>>(SCHEMA);

  useEffect(() => {
    const saved = localStorage.getItem(BI_SCHEMA_STORAGE_KEY);
    if (!saved) return;
    try {
      setSchema(JSON.parse(saved));
    } catch (error) {
      // 脏数据不阻塞页面：记原始错误，继续用演示 schema
      console.error("[bi-dashboard] 本地 schema 解析失败，改用演示 schema", error);
    }
  }, []);

  // 视口填充：BiDashboard 的 fit=width 只按容器宽等比缩放，高度随内容（8 行 ≈ 644 设计px），
  // 容器偏矮时会缩成半屏、下方留白。按容器宽高比反推设计宽：
  // 缩放比 = 容器宽/设计宽，令 内容高×缩放比 = 容器高 → designWidth = 内容高 × W/H。
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const [designWidth, setDesignWidth] = useState(1920);

  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return;
    const sync = () => {
      const width = el.clientWidth;
      // 留 2px 余量，避免取整误差把内容顶出滚动条
      const height = el.clientHeight - 2;
      const contentHeight = contentHeightOf(schema);
      if (width <= 0 || height <= 0 || contentHeight <= 0) return;
      setDesignWidth(
        Math.min(3840, Math.max(960, Math.round((contentHeight * width) / height))),
      );
    };
    const observer = new ResizeObserver(sync);
    observer.observe(el);
    sync();
    return () => observer.disconnect();
  }, [schema]);

  return (
    <div style={{ height: '100%', padding: '12px 16px', boxSizing: 'border-box', display: 'flex', flexDirection: 'column' }}>
      {/* overflow:hidden 断掉"出滚动条→容器变窄→重算"的反馈环：设计宽已按可视区精确反推 */}
      <div ref={bodyRef} style={{ flex: 1, minHeight: 0, overflow: 'hidden' }}>
        <BiProvider env={{ emptyText: '暂无数据' }}>
          <BiDashboard schema={schema} view={{ designWidth, fit: 'width' }} />
        </BiProvider>
      </div>
    </div>
  );
}
