/** @go-wind/bi-vue 的类型声明（库 d.ts 发布待修，先用 shim） */
declare module "@go-wind/bi-vue" {
  import type { DefineComponent } from "vue";
  import type { BiDashboardSchema } from "@go-wind/bi-core";

  export const BiDashboard: DefineComponent<{
    schema: Partial<BiDashboardSchema>;
    view?: { designWidth?: number; fit?: "width" | "none" };
  }, {}, any>;

  export const BiDesigner: DefineComponent<{
    initialSchema?: Partial<BiDashboardSchema>;
    demo?: boolean;
  }, { change: (schema: BiDashboardSchema) => void }, any>;

  export function provideBiEnv(env: Record<string, unknown>): void;
  export function setBiLocale(locale: string): void;
}
