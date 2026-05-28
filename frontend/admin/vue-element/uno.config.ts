// https://unocss.nodejs.cn/guide/config-file
import {
  defineConfig,
  presetAttributify,
  presetIcons,
  presetTypography,
  presetUno,
  presetWebFonts,
  presetWind3,
  transformerDirectives,
  transformerVariantGroup,
} from "unocss";

import { FileSystemIconLoader } from "@iconify/utils/lib/loader/node-loaders";
import { createRequire } from "module";
import fs from "fs";
import path from "path";

const require = createRequire(import.meta.url);

// 本地SVG图标目录
const iconsDir = "./src/assets/icons";
// Lucide 图标路径
const lucideIcons = require("@iconify-json/lucide/icons.json");

// 扫描路由文件提取 lucide 图标
const extractLucideIcons = () => {
  const routeDir = "./src/router/routes/modules";
  const icons = new Set<string>();
  const regex = /icon:\s*["']lucide:([\w-]+)["']/g;

  const scanDir = (dir: string) => {
    if (!fs.existsSync(dir)) return;
    const files = fs.readdirSync(dir);
    for (const file of files) {
      const filePath = path.join(dir, file);
      const stat = fs.statSync(filePath);
      if (stat.isDirectory()) {
        scanDir(filePath);
      } else if (file.endsWith(".ts")) {
        const content = fs.readFileSync(filePath, "utf-8");
        let match;
        while ((match = regex.exec(content)) !== null) {
          icons.add(`i-lucide:${match[1]}`);
        }
      }
    }
  };

  try {
    scanDir(routeDir);
  } catch (error) {
    console.error("扫描路由文件提取图标失败:", error);
  }
  return Array.from(icons);
};

// 读取本地 SVG 目录，自动生成 safelist
const generateSafeList = () => {
  try {
    return fs
      .readdirSync(iconsDir)
      .filter((file) => file.endsWith(".svg"))
      .map((file) => `i-svg:${file.replace(".svg", "")}`);
  } catch (error) {
    console.error("无法读取图标目录:", error);
    return [];
  }
};

export default defineConfig({
  // 自定义快捷类
  shortcuts: {
    "wh-full": "w-full h-full",
    "flex-center": "flex justify-center items-center",
    "flex-x-center": "flex justify-center",
    "flex-y-center": "flex items-center",
    "flex-x-start": "flex items-center justify-start",
    "flex-x-between": "flex items-center justify-between",
    "flex-x-end": "flex items-center justify-end",
  },
  theme: {
    colors: {
      primary: "var(--el-color-primary)",
      primary_dark: "var(--el-color-primary-light-5)",

      success: "var(--success)",
      warning: "var(--warning)",
      danger: "var(--destructive)",
    },

    borderRadius: {
      DEFAULT: "var(--radius)",

      sm: "calc(var(--radius) * 0.5)",
      lg: "calc(var(--radius) * 1.5)",
    },

    breakpoints: Object.fromEntries(
      [640, 768, 1024, 1280, 1536, 1920, 2560].map((size, index) => [
        ["sm", "md", "lg", "xl", "2xl", "3xl", "4xl"][index],
        `${size}px`,
      ])
    ),
  },
  presets: [
    presetWind3(),
    presetAttributify(),
    presetIcons({
      // 额外属性
      extraProperties: {
        display: "inline-block",
        width: "1em",
        height: "1em",
      },
      // 图标集合配置
      collections: {
        // svg 是图标集合名称，使用 `i-svg:图标名` 调用
        svg: FileSystemIconLoader(iconsDir, (svg) => {
          // 不修改 SVG 内容，保持原始颜色
          return svg;
        }),
        // 显式加载 lucide 图标集
        lucide: (name) => {
          const icon = lucideIcons.icons[name];
          if (!icon || !icon.body) return undefined;
          // UnoCSS preset-icons 期望返回的是 SVG 的 body 内容（不包含 svg 标签）
          // 但有时候需要返回完整的 svg 字符串。根据报错 "not a valid SVG"，尝试返回完整结构
          return `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${icon.body}</svg>`;
        },
      },
      scale: 1.2, // 稍微放大一点图标
    }),
    presetTypography(),
    presetWebFonts({
      fonts: {
        // ...
      },
    }),
  ],
  safelist: [
    ...generateSafeList(),
    ...extractLucideIcons(), // 自动提取路由中使用的 lucide 图标
  ],
  transformers: [transformerDirectives(), transformerVariantGroup()],
});
