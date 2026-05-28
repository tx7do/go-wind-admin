# core/i18n 国际化模块

> 面向 admin 业务逻辑开发者的使用指南

## 模块定位

基于 `vue-i18n` 的国际化基础设施，提供 **语言包加载、语言切换、翻译工具函数** 等能力。

翻译资源与框架代码分离——框架层只负责加载和切换，翻译文本全部存放在 `src/locales/` 目录。

---

## 目录结构

```
core/i18n/
├── index.ts       # 统一导出
├── setup.ts       # 初始化、语言加载、翻译工具
├── types.ts       # 类型定义
└── utils.ts       # 语言包解析工具

src/locales/       # 翻译资源（与模块分离）
├── zh-CN/
│   ├── common.json       # 通用文本
│   ├── core.json         # 框架核心文本
│   ├── routes.json       # 路由标题
│   ├── preferences.json  # 偏好设置面板
│   ├── enum.json         # 枚举值
│   └── pages/            # 页面级翻译
│       ├── user.json
│       ├── role.json
│       └── ...（每个页面一个文件）
└── en-US/
    └── （同结构）
```

---

## 支持的语言

当前支持 `zh-CN`（中文）和 `en-US`（英文），定义在偏好类型中：

```ts
type SupportedLanguagesType = "zh-CN" | "en-US";
```

默认语言为 `zh-CN`。

---

## 快速使用

### 在 Vue 组件中

```vue
<script setup lang="ts">
import { useI18n } from "@/core/i18n";

const { t } = useI18n();
</script>

<template>
  <span>{{ t("common.button.confirm") }}</span>
  <span>{{ t("pages.user.form.username") }}</span>
</template>
```

### 在非组件上下文中（composable / 工具函数）

```ts
import { $t, $te } from "@/core/i18n";

// 直接翻译
const label = $t("common.button.save"); // "保存"

// 检查 key 是否存在
if ($te("pages.user.form.mobile")) {
  // key 存在
}
```

> `$t` 和 `$te` 是 `i18n.global.t` / `i18n.global.te` 的简写，可在任何上下文使用，不需要 `useI18n()`。

---

## 翻译文件约定

### 命名空间映射

翻译文件按目录结构自动映射为嵌套命名空间：

```
locales/zh-CN/common.json           → common.*
locales/zh-CN/routes.json           → routes.*
locales/zh-CN/pages/user.json       → pages.user.*
locales/zh-CN/pages/org_unit.json   → pages.org_unit.*
```

### 引用路径规则

```ts
// 通用文本 → common.json
t("common.button.confirm")
t("common.message.success")
t("common.table.action")

// 路由标题 → routes.json
t("routes.dashboard.title")
t("routes.opm.moduleName")

// 页面文本 → pages/xxx.json
t("pages.user.form.username")
t("pages.user.table.realname")
t("pages.role.table.name")

// 偏好设置 → preferences.json
t("preferences.layout.sidebar")

// 枚举翻译 → enum.json
t("enum.status.ON")

// 框架核心 → core.json
t("core.authentication.loginSuccess")
```

### 带插值的翻译

JSON 文件中使用 `{变量名}` 占位：

```json
{
  "message": {
    "confirmDelete": "确定要删除该{moduleName}吗？"
  }
}
```

```ts
t("common.message.confirmDelete", { moduleName: "用户" });
// → "确定要删除该用户吗？"
```

---

## 核心 API

### `useI18n()`

Vue 组件内使用的 composable，返回 `{ t, te, locale, ... }`。

```ts
const { t, locale } = useI18n();
```

> 从 `@/core/i18n` 导入，不要从 `vue-i18n` 直接导入（已重新导出）。

### `$t(key, params?)`

全局翻译函数，可在任何上下文使用。

```ts
import { $t } from "@/core/i18n";

$t("common.button.save");                          // "保存"
$t("common.pagination.total", { total: 100 });     // "共 100 条记录"
```

### `$te(key)`

检查翻译 key 是否存在。

```ts
import { $te } from "@/core/i18n";

$te("common.button.save");     // true
$te("nonexistent.key");        // false
```

### `translateRouteTitle(title)`

翻译路由标题（`meta.title`），用于面包屑、侧边栏、标签页。

```ts
import { translateRouteTitle } from "@/core/i18n";

translateRouteTitle("routes.dashboard.title"); // "概览"
translateRouteTitle("原始标题");                // key 不存在时返回原文
```

> 如果 key 在当前语言包中不存在，直接返回原始字符串，不会显示 key 名。

### `loadLocaleMessages(lang)`

加载指定语言的翻译包并切换语言。按需加载——只加载当前需要的语言。

```ts
import { loadLocaleMessages } from "@/core/i18n";

await loadLocaleMessages("en-US");  // 切换到英文
await loadLocaleMessages("zh-CN");  // 切换到中文
```

**语言切换流程**：

1. 通过 `import.meta.glob` 懒加载对应语言的 JSON 文件
2. 调用 `i18n.global.setLocaleMessage(lang, messages)` 设置语言包（完全替换，非合并）
3. 执行 `loadMessages(lang)` 回调（用于合并第三方库的语言包）
4. 更新 `i18n.global.locale` 和 `<html lang="...">`

---

## 如何添加新的翻译

### 1. 在翻译文件中添加 key

**通用文本** → `src/locales/zh-CN/common.json` 和 `src/locales/en-US/common.json`：

```json
{
  "section": {
    "mySection": "我的分区"
  }
}
```

**页面级文本** → `src/locales/zh-CN/pages/my-module.json`：

```json
{
  "title": "我的模块",
  "table": {
    "name": "名称",
    "status": "状态"
  },
  "form": {
    "name": "名称",
    "status": "状态"
    "description": "描述"
  },
  "button": {
    "customAction": "自定义操作"
  }
}
```

同时在 `src/locales/en-US/pages/my-module.json` 添加英文翻译。

### 2. 在代码中使用

```vue
<template>
  <span>{{ t("pages.myModule.title") }}</span>
</template>
```

### 3. 页面翻译文件的推荐结构

```json
{
  "title": "页面标题",
  "table": { },
  "form": { },
  "button": { },
  "message": { }
}
```

---

## 语言切换触发方式

语言切换通过偏好设置统一管理：

```
用户切换语言
  → preferencesManager.updatePreferences({ app: { locale: "en-US" } })
    → preferences.ts handleUpdates() 检测到 locale 变化
      → loadLocaleMessages("en-US")
        → 加载 JSON → setLocaleMessage → 更新 locale
```

业务层**不要**直接调用 `loadLocaleMessages`，应通过偏好设置切换：

```ts
import { updatePreferences } from "@/core/preferences";

// 正确方式
updatePreferences({ app: { locale: "en-US" } });
```

---

## 路由标题国际化

路由 `meta.title` 存储的是 i18n key，不是最终显示文本：

```ts
// 路由定义
{
  path: "users",
  meta: {
    title: "routes.opm.user",  // i18n key
  }
}
```

渲染时通过 `translateRouteTitle()` 翻译：

- 侧边栏菜单：`LayoutSidebarItem` → `translateRouteTitle(item.meta.title)`
- 面包屑：`LayoutBreadcrumb` → `translateRouteTitle()`
- 标签页：`LayoutTagsView` → `translateRouteTitle()`
- 浏览器标题：`common.guard.ts` 守卫 → `translateRouteTitle()`

---

## 导入路径

```ts
// 组件内 composable
import { useI18n } from "@/core/i18n";

// 非组件上下文（全局翻译函数）
import { $t, $te } from "@/core/i18n";

// 路由标题翻译
import { translateRouteTitle } from "@/core/i18n";

// 语言包加载（通常由偏好设置自动调用）
import { loadLocaleMessages } from "@/core/i18n";

// 类型
import type { Locale } from "@/core/i18n";
```

---

## 常见场景速查

| 场景 | 用法 |
|---|---|
| 组件内翻译 | `const { t } = useI18n(); t("common.button.save")` |
| 非组件翻译 | `$t("common.message.success")` |
| 带参数翻译 | `t("common.pagination.total", { total: 100 })` |
| 检查 key 存在 | `$te("pages.user.form.mobile")` |
| 路由标题 | `translateRouteTitle("routes.opm.user")` |
| 切换语言 | `updatePreferences({ app: { locale: "en-US" } })` |
| 新增页面翻译 | 在 `locales/zh-CN/pages/` 和 `locales/en-US/pages/` 各加一个 JSON 文件 |
