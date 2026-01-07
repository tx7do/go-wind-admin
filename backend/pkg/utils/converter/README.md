# 权限码转换器

## 概述

本工具将菜单信息转换为权限码，权限码用于权限控制与校验。权限码由模块（`Module`）、子模块（`SubModule`）与动作（`Action`）组成，使用分隔符 `:` 串联，均为单数、小写，多个单词使用 `-` 连接。

## 命名与格式规则

- 路径到模块规则：
    - 去除首尾多余的 `/` 和空白。
    - 将路径中的 `/` 替换为 `:`，得到模块与子模块的组合。
    - 对每个段使用单数化（例如 `users` -> `user`）。
    - 统一小写。
    - 例如：`/admin/settings/` -> `admin:setting`（再接动作）。

- 动作（Action）命名：
    - 动作为短小英文标识，统一小写。
    - 最终权限码形如：`{module}:{submodule}:{action}`（当仅有 module 时省略 submodule）。

## Menu_Type 到 Action 的映射

- `Menu_CATALOG` -> `dir`
- `Menu_MENU` -> `access`
- `Menu_BUTTON` -> 根据按钮标题分类（参见下表）
- `Menu_EMBEDDED` -> `view`
- `Menu_LINK` -> `jump`
- 未知类型 -> 空字符串（不生成动作）

## 按钮标题到 Action 的映射（常用关键词）

- **add**: `"add"`, `"create"`, `"new"`, `"新增"`, `"添加"`, `"创建"` 等 -> `add`
- **edit**: `"edit"`, `"update"`, `"modify"`, `"save"`, `"保存"`, `"修改"`, `"更新"`, `"编辑"` -> `edit`
- **delete**: `"delete"`, `"del"`, `"remove"`, `"删除"`, `"移除"` -> `delete`
- **import**: `"import"`, `"导入"`, `"导入为"`, `"importcsv"`, `"importexcel"` -> `import`
- **export**: `"export"`, `"导出"`, `"下载"`, `"exportcsv"`, `"exportexcel"` -> `export`
- 其它或空白标题 -> `act`

匹配逻辑：

- 先做 `strings.TrimSpace` 并转小写。
- 优先按关键词集进行匹配（可采用前缀/包含/完全匹配的策略，项目中实现为 `matchAnyKeyword`）。
- 支持中文与英文关键词混合匹配。

## 实现注意点

- 路径拆分与单数化依赖 `inflection.Singular`（若移除相关逻辑，请同步移除 `import`）。
- 若使用 `tokenize` 并检查 Unicode 字符类别，需要在 `import` 中包含 `unicode`。
- `buttonAction` 建议支持 `import` 关键字（如 `import`、`导入` 等），并返回动作 `import`。
- 在调用 `buttonAction` 前应对标题做 `strings.TrimSpace` 以避免空白影响结果。

## 示例

- 输入：路径 `/users`，类型 `Menu_BUTTON`，标题 `新增`  
  输出权限码示例：`user:add`

- 输入：路径 `/admin/settings`，类型 `Menu_MENU`，标题 空  
  输出权限码示例：`admin:setting:access`

- 输入：路径 `/reports`，类型 `Menu_BUTTON`，标题 `一键导出为Excel`  
  输出权限码示例：`report:export`

## 单元测试

已包含或建议的测试文件：

- `pkg/utils/converter/convert_code_test.go`（覆盖路径处理、单数化与类型映射）
- `pkg/utils/converter/type_to_action_test.go`（覆盖各 `Menu_Type` 分支与 `buttonAction` 的多种标题映射）

运行测试（在项目根目录）：

```bash
go test ./pkg/utils/converter -v
```
