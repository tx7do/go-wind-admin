import { BUILT_IN_THEME_PRESETS } from "./config/constants";
import type { Preferences } from "./types";
import { generateColorVariables, generatorColorVariables } from "@/utils/theme";
import { toHex } from "@/utils/color";

/**
 * 更新 CSS 变量的函数
 * @param variables 要更新的 CSS 变量与其新值的映射
 * @param id
 */
function executeUpdateCSSVariables(
  variables: { [key: string]: string },
  id = "__gowind-styles__"
): void {
  // 获取或创建内联样式表元素
  const styleElement = document.querySelector(`#${id}`) || document.createElement("style");

  styleElement.id = id;

  // 构建要更新的 CSS 变量的样式文本
  let cssText = ":root {";
  for (const key in variables) {
    if (Object.prototype.hasOwnProperty.call(variables, key)) {
      cssText += `${key}: ${variables[key]};`;
    }
  }
  cssText += "}";

  // 将样式文本赋值给内联样式表
  styleElement.textContent = cssText;

  // 将内联样式表添加到文档头部
  if (!document.querySelector(`#${id}`)) {
    setTimeout(() => {
      document.head.append(styleElement);
    });
  }
}

/**
 * 更新主题的 CSS 变量以及其他 CSS 变量
 * @param preferences - 当前偏好设置对象，它的主题值将被用来设置文档的主题。
 */
function updateCSSVariables(preferences: Preferences) {
  // 当修改到颜色变量时，更新 css 变量
  const root = document.documentElement;
  if (!root) {
    return;
  }

  const theme = preferences?.theme ?? {};

  const { builtinType, mode, radius, semiDarkSidebar, semiDarkHeader } = theme;

  // html 设置 dark 类
  if (Reflect.has(theme, "mode")) {
    const dark = isDarkTheme(mode);
    root.classList.toggle("dark", dark);
  }

  // html 设置 semi-dark-sidebar / semi-dark-header
  // 仅在浅色模式下生效：给侧边栏/顶栏启用深色背景
  const isDark = isDarkTheme(mode);
  if (Reflect.has(theme, "semiDarkSidebar")) {
    root.classList.toggle("semi-dark-sidebar", !isDark && !!semiDarkSidebar);
  }
  if (Reflect.has(theme, "semiDarkHeader")) {
    root.classList.toggle("semi-dark-header", !isDark && !!semiDarkHeader);
  }

  // html 设置 data-theme=[builtinType]
  if (Reflect.has(theme, "builtinType")) {
    const rootTheme = root.dataset.theme;
    if (rootTheme !== builtinType) {
      root.dataset.theme = builtinType;
    }
  }

  // 获取当前的内置主题
  const currentBuiltType = [...BUILT_IN_THEME_PRESETS].find((item) => item.type === builtinType);

  let builtinTypeColorPrimary: string | undefined = "";

  if (currentBuiltType) {
    const isDark = isDarkTheme(preferences.theme.mode);
    // 设置不同主题的主要颜色
    const color = isDark
      ? currentBuiltType.darkPrimaryColor || currentBuiltType.primaryColor
      : currentBuiltType.primaryColor;
    builtinTypeColorPrimary = color || currentBuiltType.color;
  }

  // 如果内置主题颜色和自定义颜色都不存在，则不更新主题颜色
  if (
    builtinTypeColorPrimary ||
    Reflect.has(theme, "colorPrimary") ||
    Reflect.has(theme, "colorDestructive") ||
    Reflect.has(theme, "colorSuccess") ||
    Reflect.has(theme, "colorWarning")
  ) {
    // preferences.theme.colorPrimary = builtinTypeColorPrimary || colorPrimary;
    updateMainColorVariables(preferences);
  }

  // 更新圆角
  if (Reflect.has(theme, "radius")) {
    document.documentElement.style.setProperty("--radius", `${radius}rem`);
  }
}

/**
 * 更新主要的 CSS 变量
 * @param  preference - 当前偏好设置对象，它的颜色值将被转换成 HSL 格式并设置为 CSS 变量。
 */
function updateMainColorVariables(preference: Preferences) {
  if (!preference.theme) {
    return;
  }
  const { colorDestructive, colorPrimary, colorSuccess, colorWarning } = preference.theme;

  // 将颜色统一转换为 hex 格式（支持 HSL/RGB/Hex 输入）
  const hexPrimary = toHex(colorPrimary || "#1677ff");
  const hexSuccess = toHex(colorSuccess || "#52c41a");
  const hexWarning = toHex(colorWarning || "#faad14");
  const hexDestructive = toHex(colorDestructive || "#ff4d4f");

  // 1. 生成自定义颜色变量 (--primary-500, --success-500, ...)
  const customVariables = generatorColorVariables([
    { color: hexPrimary, name: "primary" },
    { alias: "warning", color: hexWarning, name: "yellow" },
    { alias: "success", color: hexSuccess, name: "green" },
    { alias: "destructive", color: hexDestructive, name: "red" },
  ]);

  // 2. 生成 Element Plus 颜色变量 (--el-color-primary, --el-color-primary-light-1, ...)
  const mode = isDarkTheme(preference.theme.mode) ? "dark" : "light";
  const epVariables = generateColorVariables({
    colorPrimary: hexPrimary,
    colorSuccess: hexSuccess,
    colorWarning: hexWarning,
    colorDestructive: hexDestructive,
    mode,
  });

  // 3. 使用内联样式设置 Element Plus 变量（最高优先级，不会被覆盖）
  const root = document.documentElement;
  Object.entries(epVariables).forEach(([key, value]) => {
    root.style.setProperty(key, value);
  });

  // 4. 设置映射变量 (--primary-500 → --primary 等)
  const colorMappings: Record<string, string> = {
    "--green-500": "--success",
    "--primary-500": "--primary",
    "--red-500": "--destructive",
    "--yellow-500": "--warning",
  };
  Object.entries(colorMappings).forEach(([sourceVar, targetVar]) => {
    const colorValue = customVariables[sourceVar];
    if (colorValue) {
      root.style.setProperty(targetVar, colorValue);
    }
  });

  // 5. 自定义变量通过 style 标签设置
  executeUpdateCSSVariables(customVariables);
}

function isDarkTheme(theme: string) {
  let dark = theme === "dark";
  if (theme === "auto") {
    dark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  }
  return dark;
}

export { isDarkTheme, updateCSSVariables };
