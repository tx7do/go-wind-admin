import "@go-wind/bi-react/style.css";
import { useMemo } from 'react';
import { BiProvider, BiDashboard } from '@go-wind/bi-react';
import type { BiDashboardSchema } from '@go-wind/bi-core';

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

export default function BiDashboardPage() {
  const schema = useMemo(() => SCHEMA, []);
  return (
    <div style={{ height: "100%", padding: "16px 20px", boxSizing: "border-box" }}>
      <BiProvider env={{ emptyText: "暂无数据" }}>
        <BiDashboard schema={schema} view={{ designWidth: 1920, fit: "width" }} />
      </BiProvider>
    </div>
  );
}
