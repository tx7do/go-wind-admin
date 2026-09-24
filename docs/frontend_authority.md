# 前端权限

前端的权限主要分为两个部分：

1. 路由的访问权限；
2. 按钮的访问权限。

## 路由的访问权限

路由的访问权限又分为两种方式：

1. 后端控制；
2. 前端控制。

### 后端访问控制

* **实现原理**: 是通过接口动态生成路由表，且遵循一定的数据结构返回。前端根据需要处理该数据为可识别的结构，再通过 `router.addRoute` 添加到路由实例，实现权限的动态生成。

* **缺点**: 后端需要提供符合规范的数据结构，前端需要处理数据结构，适合权限较为复杂的系统。

前端启用办法，修改`.env`配置文件`VITE_ROUTER_ACCESS_MODE`的值为`backend`：

```env
# 路由的访问模式：frontend，backend
VITE_ROUTER_ACCESS_MODE=backend
```

前端的核心代码在 `apps/admin/src/router/access.ts`（以下为现状，不是示意）：菜单来自
`fetchNavigation()`（`#/api` 的 admin-portal 封装，后端 RPC 是 `GetNavigation`，其响应消息名叫
`ListRouteResponse`——**别把消息名当成 RPC 名去找 `ListRoute`，那个端点不存在**）：

```typescript
// apps/admin/src/router/access.ts
async function getAllMenusApi(): Promise<RouteRecordStringComponent[]> {
  const data = (await fetchNavigation()) ?? [];
  const unwrapped = (data as any)?.data ?? data;   // 兼容 {items} 与 {data:{items}} 两种包装
  return unwrapped?.items ?? [];
}

// 后端模式：先预取路由；拿到空列表或请求失败，则把 effectiveMode 降级成 'frontend'
let effectiveMode = preferences.app.accessMode;
if (effectiveMode === 'backend') {
  try {
    cachedBackendRoutes = await getAllMenusApi();
    if (cachedBackendRoutes.length === 0) effectiveMode = 'frontend';
  } catch {
    effectiveMode = 'frontend';
  }
}
const result = await generateAccessible(effectiveMode, { ...options, fetchMenuListAsync: ... });
```

> **降级要知道**：`backend` 模式下菜单为空或接口报错时，vben 会静默退回前端模式，用本地固定路由表
> 生成菜单。也就是说"配了 backend 却看着像 frontend"是可能的，排障时先看网络面板有没有 `GetNavigation`
> 请求，再看是不是被降级了。react / vue-element 无此降级（后端模式失败即空菜单）。

### 前端访问控制

* **实现原理**: 在前端固定写死路由的权限，指定路由有哪些权限可以查看。只初始化通用的路由，需要权限才能访问的路由没有被加入路由表内。在登录后或者其他方式获取用户角色后，通过角色去遍历路由表，获取该角色可以访问的路由表，生成路由表，再通过 `router.addRoute` 添加到路由实例，实现权限的过滤。

* **缺点**: 权限相对不自由，如果后台改动角色，前台也需要跟着改动。适合角色较固定的系统。

前端启用办法，修改`.env`配置文件`VITE_ROUTER_ACCESS_MODE`的值为`frontend`：

> 本小节的 env 开关是 **vue-vben** 的机制；react / vue-element 的双模式生成器在
> `core/router/generators/generate-routes-{frontend,backend}.ts`，开关为运行时偏好
> `preferences.app.accessMode`——三端对照见下文「三端实现对照」。

```env
# 路由的访问模式：frontend，backend
VITE_ROUTER_ACCESS_MODE=frontend
```

然后，我们需要在本地的固定路由里面写入`authority`字段，里边填写的是后端配置的角色码：

```typescript
 {
    meta: {
      authority: ['super'],
    },
},
```

> **`authority: []`（空数组）三端语义不一致，别当成通用写法**：vue-vben 与 vue-element 的
> `hasAuthority` 判的是 `if (!authority) return true`——空数组在 JS 里是 truthy，于是继续走匹配、
> 必然落空，结果**谁都看不见**；react 判的是 `if (!meta?.authority?.length) return true`
> （`src/core/router/generators/generate-routes-frontend.ts:59-60`），空数组等于**无权限要求、所有人可见**，
> 语义正好相反。**结论**：要"所有人可见"就把 `authority` 整个省略，别写空数组；
> 要"锁死"就用一个没人持有的码。这是端间差异、不是配置技巧。

根据`src/store/auth`里面的代码显示：

```typescript
// 设置登录用户信息，需要确保 userInfo.roles 是一个数组，且包含路由表中的权限
// 例如：userInfo.roles=['super', 'admin']
authStore.setUserInfo(userInfo);
```

所以，我们需要在用户的登陆数据里面加入一个`roles`的字段：

```protobuf
message User {
    repeated string roles;
}
```

并且读取角色表里面的角色值，传递给前端。

#### 菜单可见，但禁止访问

有时候，我们需要菜单可见，但是禁止访问，可以通过下面的方式实现，设置 `menuVisibleWithForbidden` 为 `true`，此时菜单可见，但是禁止访问，会跳转403页面。

```typescript
{
    meta: {
      menuVisibleWithForbidden: true,
    },
},
```

## 按钮的访问权限

在某些情况下，我们需要对按钮进行细粒度的控制，我们可以借助接口或者角色来控制按钮的显示。

### 权限码

权限码为接口返回的权限码，通过权限码来判断按钮是否显示。**三端统一**经
`apiClient.adminPortalService.GetMyPermissionCode`（admin-portal 域唯一端点）拉取，
响应同时携带 `codes`（权限码）与 `hiddenFields`（字段级权限黑名单，见下文字段级权限一节）；
落库到各端的 access store（vue-vben：`accessStore`；react：`userStore.accessCodes`；
vue-element：`accessStore.accessCodes`）。

```typescript
// 三端登录流的公共形态（composables/hooks 层封装）：
const me = await getMe();                       // 用户信息（含 roles）
const codes = await getMyPermissionCode();      // { codes, hiddenFields }
userStore.setUser(me);
accessStore.setAccessCodes(codes.codes);        // 按钮级权限码
accessStore.setHiddenFields(codes.hiddenFields); // 字段级权限黑名单
```

权限码返回的数据结构为字符串数组，例如：`['AC_100100', 'AC_100110', 'AC_100120', 'AC_100010']`

有了权限码，就可以使用 `@vben/access` 提供的`AccessControl`组件及API来进行按钮的显示与隐藏。

#### 组件方式

```typescript
<script lang="ts" setup>
import { AccessControl, useAccess } from '@vben/access';

const { accessMode, hasAccessByCodes } = useAccess();
</script>

<template>
  <!-- 需要指明 type="code" -->
  <AccessControl :codes="['AC_100100']" type="code">
    <Button> Super 账号可见 ["AC_1000001"] </Button>
  </AccessControl>
  <AccessControl :codes="['AC_100030']" type="code">
    <Button> Admin 账号可见 ["AC_100010"] </Button>
  </AccessControl>
  <AccessControl :codes="['AC_1000001']" type="code">
    <Button> User 账号可见 ["AC_1000001"] </Button>
  </AccessControl>
  <AccessControl :codes="['AC_100100', 'AC_100010']" type="code">
    <Button> Super & Admin 账号可见 ["AC_100100","AC_1000001"] </Button>
  </AccessControl>
</template>
```

#### API方式

```typescript
<script lang="ts" setup>
import { AccessControl, useAccess } from '@vben/access';

const { hasAccessByCodes } = useAccess();
</script>

<template>
  <Button v-if="hasAccessByCodes(['AC_100100'])">
    Super 账号可见 ["AC_1000001"]
  </Button>
  <Button v-if="hasAccessByCodes(['AC_100030'])">
    Admin 账号可见 ["AC_100010"]
  </Button>
  <Button v-if="hasAccessByCodes(['AC_1000001'])">
    User 账号可见 ["AC_1000001"]
  </Button>
  <Button v-if="hasAccessByCodes(['AC_100100', 'AC_1000001'])">
    Super & Admin 账号可见 ["AC_100100","AC_1000001"]
  </Button>
</template>
```

#### 指令方式

指令支持绑定单个或多个权限码。单个时可以直接传入字符串或数组中包含一个权限码，多个权限码则传入数组。

```typescript
<template>
  <Button class="mr-4" v-access:code="'AC_100100'">
    Super 账号可见 'AC_100100'
  </Button>
  <Button class="mr-4" v-access:code="['AC_100030']">
    Admin 账号可见 ["AC_100010"]
  </Button>
  <Button class="mr-4" v-access:code="['AC_1000001']">
    User 账号可见 ["AC_1000001"]
  </Button>
  <Button class="mr-4" v-access:code="['AC_100100', 'AC_1000001']">
    Super & Admin 账号可见 ["AC_100100","AC_1000001"]
  </Button>
</template>
```

### 角色

角色判断方式不需要接口返回的权限码，直接通过角色来判断按钮是否显示。

#### 组件方式

```typescript
<script lang="ts" setup>
import { AccessControl } from '@vben/access';
</script>

<template>
  <AccessControl :codes="['super']">
    <Button> Super 角色可见 </Button>
  </AccessControl>
  <AccessControl :codes="['admin']">
    <Button> Admin 角色可见 </Button>
  </AccessControl>
  <AccessControl :codes="['user']">
    <Button> User 角色可见 </Button>
  </AccessControl>
  <AccessControl :codes="['super', 'admin']">
    <Button> Super & Admin 角色可见 </Button>
  </AccessControl>
</template>
```

#### API方式

```typescript
<script lang="ts" setup>
import { useAccess } from '@vben/access';

const { hasAccessByRoles } = useAccess();
</script>

<template>
  <Button v-if="hasAccessByRoles(['super'])"> Super 账号可见 </Button>
  <Button v-if="hasAccessByRoles(['admin'])"> Admin 账号可见 </Button>
  <Button v-if="hasAccessByRoles(['user'])"> User 账号可见 </Button>
  <Button v-if="hasAccessByRoles(['super', 'admin'])">
    Super & Admin 账号可见
  </Button>
</template>
```

#### 指令方式

指令支持绑定单个或多个角色。单个时可以直接传入字符串或数组中包含一个角色，多个角色均可访问则传入数组。

```typescript
<template>
  <Button class="mr-4" v-access:role="'super'"> Super 角色可见 </Button>
  <Button class="mr-4" v-access:role="['super']"> Super 角色可见 </Button>
  <Button class="mr-4" v-access:role="['admin']"> Admin 角色可见 </Button>
  <Button class="mr-4" v-access:role="['user']"> User 角色可见 </Button>
  <Button class="mr-4" v-access:role="['super', 'admin']">
    Super & Admin 角色可见
  </Button>
</template>
```

### 三端实现对照

三端的访问控制子系统 API 同构、实现位置与个别语义有差异：

| | vue-vben | react | vue-element |
|---|---|---|---|
| 子系统位置 | `@vben/access`（框架包） | `src/core/access/`（`access-control.tsx` + `use-access.ts`） | `src/core/access/`（`access-control.vue` + `use-access.ts` + `directive.ts`） |
| 组件式 | `AccessControl`（`type="code"\|"role"`） | `AccessControl`（`type="code"\|"role"\|"authority"`，另支持 `fallback`） | `AccessControl`（对齐 vben） |
| Hook 式 | `hasAccessByRoles` / `hasAccessByCodes` | `hasAccessByRoles` / `hasAccessByCodes` / `hasAccessByAuthority`（roles ∪ codes 混合判定，对应路由 `meta.authority`） | `hasAccessByRoles` / `hasAccessByCodes` / `hasAccess`（roles ∪ codes 并集） |
| 指令式 | `v-access:code` / `v-access:role` | 无（React 无指令机制） | `v-access`（混合判定，无权限 `el.remove()`） |
| 权限码匹配语义 | **精确匹配**；注意 vben 的 accessCodes 实为 roles∪codes 并集（authentication.store 的 getUserPermissionCodes 把角色码并入），按钮级判定同样吃角色码 | **精确匹配** | **精确 + 前缀授权**：用户持 `sys:a` 即判过 `sys:a:b`（`requiredCode.startsWith(userCode + ":")`）——**比另两端宽** |
| 路由双模式 | `@vben/access` 的 `generateAccessible` + `VITE_ROUTER_ACCESS_MODE`（env）；后端模式菜单为空/失败**静默降级**为前端模式 | `src/core/router/generators/generate-routes-{frontend,backend}.ts` + `preferences.app.accessMode`（运行时偏好；react 无 UI 开关也无 toggleAccessMode，改偏好即生效）；后端模式失败只 `console.error` 并返回空路由，**不降级** | 同 react 结构，另有 `useAccess().toggleAccessMode()` 运行时切换；后端模式失败同样返回 `[]` 不降级 |
| `meta.authority: []` 语义 | 空数组 = 所有人不可见（`if (!authority) return true`，空数组是 truthy） | **空数组 = 所有人可见**（`if (!meta?.authority?.length) return true`）——与另两端相反 | 空数组 = 所有人不可见（同 vben） |
| 路由生成防线 | `effects/access/accessible.ts` 生成前 `cloneDeep(options.routes)`（2026-09 实证必要：filterTree 会就地改写 node.children，未登录空权限预构建会把带 authority 路由从共享单例永久剔除、登录后找不回） | `generate-routes-frontend.ts` 纯过滤：递归构造新节点、element 按引用共享、零回写共享单例（react 路由带活的 React 元素，不宜整树 cloneDeep） | `core/router/accessible.ts` 生成前 `cloneDeep(options.routes)`（同 vben） |
| 权限码/菜单端点 | 权限码三端统一 `adminPortalService.GetMyPermissionCode`（codes+hiddenFields）；后端模式的菜单下发经同域 `GetNavigation`（返回 `ListRouteResponse` 菜单路由） | 同左 | 同左 |

**移植警示**：三端按钮/路由权限码的匹配语义不一致（上表"权限码匹配语义"行）——
同一权限码集合在 vue-element 下可判过另两端拒绝的前缀子码。跨端移植受控按钮前按目标端语义核对；
长期应对齐（收敛或显式声明各端语义差异为产品决策）。

## 字段级权限

字段级权限是路由与按钮之外的第三个权限轴：角色可配置一组「黑名单字段」，受限用户在界面上看不到这些字段，服务端响应中这些字段同样被裁剪。当前试点资源为用户表。

* **服务端强制**：角色配置的黑名单字段集在登录 / 刷新令牌时聚合进令牌的 `hfs` claim；服务端在读路径按黑名单清值（字段自响应中消失），对写路径清值并从 `field_mask` 中剔除对应字段（越权字段被静默剥离，不会因显式置零覆盖原值）。落点是 service 层切面 `internal/service/user_field_permission.go`（`fieldperm.ApplyReadMask*` / `StripWriteFields`，工具在 `pkg/fieldperm/`），不在 ent 隐私层——所以它只覆盖接了这个切面的资源。`/me` 端点刻意不裁剪（自编辑需要全字段）。
* **前端消费**：`ListPermissionCode`（perm-codes）响应在权限码之外同时下发 `hiddenFields`——
  **一条扁平的 `repeated string`，每项是 `"资源.字段"` 串**（如 `User.phone`；proto 见
  `admin/service/v1/i_admin_portal.proto:42`），不是按资源分组的对象。分组是各端自己做的
  （react `src/core/access/field-permission.ts` 的 `parseResourceHiddenFields` / `isFieldHidden`，
  按前缀切回资源维度）。三端 access store 持久化该清单，列表列与搜索项按其过滤、详情页条件渲染、
  编辑抽屉隐藏受控字段并从提交载荷中剔除对应键。
* **注意**：前端裁剪只是体验层，服务端裁剪才是权威——绕过前端直接调接口，受控字段同样不可读、不可写。

## 项目代码

* [go-wind-admin Gitee](https://gitee.com/tx7do/go-wind-admin)
* [go-wind-admin Github](https://github.com/tx7do/go-wind-admin)

## 参考资料

* [Vben Admin 权限](https://doc.vben.pro/guide/in-depth/access.html)
