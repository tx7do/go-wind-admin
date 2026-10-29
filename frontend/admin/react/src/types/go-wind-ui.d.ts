/** @go-wind/bi-react 类型声明（库 d.ts 发布待修，先用 shim） */
declare module "@go-wind/bi-react" {
  import type { BiDashboardSchema } from "@go-wind/bi-core";

  export interface BiEnv {
    themeMode?: () => "light" | "dark" | "auto";
    resolvers?: Record<string, (dataset: any, params?: any) => Promise<any[]>>;
    emptyText?: string;
  }

  export function BiProvider(props: { env: BiEnv; children: React.ReactNode }): React.JSX.Element;
  export function BiDashboard(props: {
    schema: Partial<BiDashboardSchema>;
    view?: { designWidth?: number; fit?: "width" | "none" };
    onChartClick?: (event: unknown) => void;
    onChartSelect?: (chartId: string) => void;
  }): React.JSX.Element;
  export function BiDesignerReact(props: {
    initialSchema?: Partial<BiDashboardSchema>;
    demo?: boolean;
    onChange?: (schema: BiDashboardSchema) => void;
    env?: BiEnv;
  }): React.JSX.Element;
}
