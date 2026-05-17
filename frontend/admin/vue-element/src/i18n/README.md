# 国际化 (i18n) 规范文档

## 📋 目录结构

```
src/i18n/
├── langs/                    # 语言包目录
│   ├── zh-CN/               # 简体中文
│   │   ├── common.json      # 全局通用词汇（按钮、操作文字）
│   │   ├── core.json        # 框架核心文案（登录、404、个人中心、Layout）
│   │   ├── app.json         # 业务公共文案（标题、提示、状态）
│   │   ├── routes.json      # 路由/菜单名称（重要）
│   │   ├── validation.json  # 表单校验提示
│   │   └── pages/           # 页面级文案（按业务模块拆分）
│   │       ├── dashboard.json
│   │       ├── user.json
│   │       ├── order.json
│   │       └── ...
│   └── en-US/               # 英文（保持相同结构）
│       ├── common.json
│       ├── core.json
│       ├── app.json
│       ├── routes.json
│       ├── validation.json
│       └── pages/
│           ├── dashboard.json
│           ├── user.json
│           ├── order.json
│           └── ...
├── setup.ts                  # i18n 初始化配置
├── types.ts                  # 类型定义
└── utils.ts                  # 工具函数
```

---

## 🎯 设计原则

### 1. 分层管理
- **common.json**: 全局复用的高频词汇
- **core.json**: 框架级别的核心功能文案
- **app.json**: 应用级别的公共文案
- **routes.json**: 路由和菜单相关文案
- **validation.json**: 表单验证错误提示
- **pages/*.json**: 页面专属文案，按模块拆分

### 2. 命名规范
使用**点号分隔的层级命名**，遵循 `模块.子模块.具体项` 的格式：

```json
{
  "user": {
    "title": "用户管理",
    "form": {
      "username": "用户名",
      "email": "邮箱"
    },
    "message": {
      "createSuccess": "创建成功",
      "deleteConfirm": "确认删除？"
    }
  }
}
```

### 3. 键名规范
- ✅ 使用 **camelCase**（小驼峰）命名
- ✅ 使用**语义化**的英文键名
- ❌ 避免使用中文作为键名
- ❌ 避免过深的嵌套（建议不超过 4 层）

---

## 📝 文件职责说明

### 1. common.json - 全局通用词汇

**用途**: 存放全项目复用的高频词汇

**示例**:
```json
{
  "button": {
    "confirm": "确定",
    "cancel": "取消",
    "save": "保存",
    "edit": "编辑",
    "delete": "删除",
    "search": "搜索",
    "reset": "重置",
    "export": "导出",
    "import": "导入",
    "add": "新增",
    "update": "更新",
    "view": "查看",
    "close": "关闭",
    "back": "返回"
  },
  "status": {
    "enabled": "启用",
    "disabled": "禁用",
    "active": "活跃",
    "inactive": "非活跃",
    "pending": "待处理",
    "completed": "已完成",
    "failed": "失败"
  },
  "text": {
    "yes": "是",
    "no": "否",
    "all": "全部",
    "none": "无",
    "unknown": "未知",
    "total": "共",
    "items": "条"
  },
  "placeholder": {
    "input": "请输入",
    "select": "请选择",
    "date": "请选择日期",
    "search": "请输入搜索内容"
  },
  "message": {
    "success": "操作成功",
    "error": "操作失败",
    "warning": "警告",
    "info": "提示",
    "loading": "加载中...",
    "noData": "暂无数据",
    "networkError": "网络异常，请稍后重试"
  }
}
```

**使用方式**:
```typescript
const { t } = useI18n()

// 按钮文本
t('common.button.confirm')

// 状态文本
t('common.status.enabled')

// 占位符
t('common.placeholder.input')
```

---

### 2. core.json - 框架核心文案

**用途**: 框架级别的功能文案（登录、错误页、个人中心等）

**示例**:
```json
{
  "login": {
    "title": "用户登录",
    "subtitle": "欢迎回来，请登录您的账号",
    "username": "用户名",
    "password": "密码",
    "captcha": "验证码",
    "rememberMe": "记住我",
    "forgotPassword": "忘记密码？",
    "loginButton": "登 录",
    "register": "注册账号",
    "message": {
      "usernameRequired": "请输入用户名",
      "passwordRequired": "请输入密码",
      "captchaRequired": "请输入验证码",
      "loginSuccess": "登录成功",
      "loginFailed": "登录失败，请检查用户名和密码"
    }
  },
  "layout": {
    "dashboard": "首页",
    "profile": "个人中心",
    "settings": "系统设置",
    "logout": "退出登录",
    "searchMenu": "搜索菜单",
    "fullscreen": "全屏",
    "theme": "主题切换",
    "language": "语言切换",
    "notification": "消息通知"
  },
  "error": {
    "404": {
      "title": "页面不存在",
      "description": "抱歉，您访问的页面不存在",
      "backHome": "返回首页"
    },
    "403": {
      "title": "无权访问",
      "description": "抱歉，您没有权限访问该页面",
      "backHome": "返回首页"
    },
    "500": {
      "title": "服务器错误",
      "description": "抱歉，服务器发生错误",
      "backHome": "返回首页"
    }
  },
  "profile": {
    "title": "个人中心",
    "basicInfo": "基本信息",
    "security": "安全设置",
    "changePassword": "修改密码",
    "avatar": "头像",
    "nickname": "昵称",
    "email": "邮箱",
    "phone": "手机号"
  }
}
```

**使用方式**:
```typescript
const { t } = useI18n()

// 登录标题
t('core.login.title')

// 404 页面标题
t('core.error.404.title')

// 布局菜单
t('core.layout.dashboard')
```

---

### 3. app.json - 业务公共文案

**用途**: 应用级别的公共文案（标题、提示、状态等）

**示例**:
```json
{
  "title": "企业管理系统",
  "subtitle": "高效、安全、可靠的管理平台",
  "header": {
    "welcome": "欢迎",
    "tenant": "租户",
    "switchTenant": "切换租户"
  },
  "footer": {
    "copyright": "Copyright © 2024 企业名称. All rights reserved.",
    "version": "版本"
  },
  "tips": {
    "required": "必填项",
    "optional": "选填项",
    "dragUpload": "拖拽文件到此处上传",
    "clickUpload": "点击上传"
  },
  "enums": {
    "gender": {
      "male": "男",
      "female": "女",
      "secret": "保密"
    },
    "enable": {
      "true": "启用",
      "false": "禁用"
    },
    "yesNo": {
      "true": "是",
      "false": "否"
    }
  }
}
```

**使用方式**:
```typescript
const { t } = useI18n()

// 应用标题
t('app.title')

// 枚举值
t('app.enums.gender.male')
```

---

### 4. routes.json - 路由/菜单名称

**用途**: 路由和菜单的显示名称（**非常重要**，影响导航显示）

**示例**:
```json
{
  "dashboard": "首页",
  "system": {
    "title": "系统管理",
    "user": "用户管理",
    "role": "角色管理",
    "menu": "菜单管理",
    "dept": "部门管理",
    "dict": "字典管理"
  },
  "business": {
    "title": "业务管理",
    "order": "订单管理",
    "product": "商品管理",
    "customer": "客户管理"
  },
  "log": {
    "title": "日志管理",
    "login": "登录日志",
    "operation": "操作日志",
    "api": "API日志"
  },
  "profile": {
    "title": "个人中心",
    "internalMessage": "站内消息"
  }
}
```

**使用方式**:
```typescript
// 在路由配置中
{
  path: '/system',
  meta: {
    title: t('routes.system.title')
  }
}

// 在菜单组件中
t('routes.dashboard')
```

---

### 5. validation.json - 表单校验提示

**用途**: 统一的表单验证错误提示

**示例**:
```json
{
  "required": "该项为必填项",
  "email": {
    "invalid": "请输入有效的邮箱地址",
    "required": "请输入邮箱地址"
  },
  "phone": {
    "invalid": "请输入有效的手机号码",
    "required": "请输入手机号码"
  },
  "url": {
    "invalid": "请输入有效的URL地址"
  },
  "number": {
    "min": "数值不能小于 {min}",
    "max": "数值不能大于 {max}",
    "range": "数值范围应在 {min} 到 {max} 之间"
  },
  "string": {
    "min": "至少输入 {min} 个字符",
    "max": "最多输入 {max} 个字符",
    "range": "字符长度应在 {min} 到 {max} 之间"
  },
  "pattern": {
    "username": "用户名只能包含字母、数字和下划线",
    "password": "密码必须包含字母和数字，长度6-20位",
    "idCard": "请输入有效的身份证号码"
  },
  "confirm": {
    "password": "两次输入的密码不一致"
  }
}
```

**使用方式**:
```typescript
const rules = {
  email: [
    { required: true, message: t('validation.email.required') },
    { type: 'email', message: t('validation.email.invalid') }
  ],
  password: [
    { 
      pattern: /^(?=.*[A-Za-z])(?=.*\d)[A-Za-z\d]{6,20}$/,
      message: t('validation.pattern.password')
    }
  ]
}
```

---

### 6. pages/*.json - 页面级文案

**用途**: 按业务模块拆分的页面专属文案

#### 示例：pages/user.json

```json
{
  "title": "用户管理",
  "description": "管理系统用户信息",
  "table": {
    "columns": {
      "username": "用户名",
      "realname": "真实姓名",
      "email": "邮箱",
      "phone": "手机号",
      "status": "状态",
      "createdAt": "创建时间",
      "updatedAt": "更新时间"
    }
  },
  "form": {
    "title": {
      "create": "新增用户",
      "update": "编辑用户"
    },
    "fields": {
      "username": "用户名",
      "password": "密码",
      "realname": "真实姓名",
      "email": "邮箱",
      "phone": "手机号",
      "gender": "性别",
      "dept": "所属部门",
      "role": "角色",
      "status": "状态"
    },
    "placeholder": {
      "username": "请输入用户名",
      "password": "请输入密码（至少6位）",
      "realname": "请输入真实姓名",
      "email": "请输入邮箱地址",
      "phone": "请输入手机号码"
    }
  },
  "button": {
    "create": "新增用户",
    "batchDelete": "批量删除",
    "export": "导出用户",
    "import": "导入用户"
  },
  "message": {
    "createSuccess": "用户创建成功",
    "updateSuccess": "用户更新成功",
    "deleteSuccess": "用户删除成功",
    "deleteConfirm": "确认删除选中的用户吗？",
    "batchDeleteConfirm": "确认删除选中的 {count} 个用户吗？",
    "importSuccess": "成功导入 {count} 个用户",
    "exportSuccess": "导出成功"
  },
  "tips": {
    "username": "用户名一旦创建不可修改",
    "password": "首次登录后建议修改密码",
    "role": "可分配多个角色"
  }
}
```

#### 示例：pages/order.json

```json
{
  "title": "订单管理",
  "table": {
    "columns": {
      "orderNo": "订单号",
      "customer": "客户",
      "amount": "金额",
      "status": "状态",
      "createdAt": "下单时间"
    }
  },
  "status": {
    "pending": "待支付",
    "paid": "已支付",
    "shipped": "已发货",
    "completed": "已完成",
    "cancelled": "已取消",
    "refunded": "已退款"
  },
  "message": {
    "cancelConfirm": "确认取消该订单吗？",
    "refundConfirm": "确认退款该订单吗？"
  }
}
```

**使用方式**:
```vue
<template>
  <div>
    <h1>{{ t('pages.user.title') }}</h1>
    
    <el-table :data="userList">
      <el-table-column prop="username" :label="t('pages.user.table.columns.username')" />
      <el-table-column prop="email" :label="t('pages.user.table.columns.email')" />
    </el-table>
    
    <el-button @click="handleCreate">
      {{ t('pages.user.button.create') }}
    </el-button>
  </div>
</template>

<script setup lang="ts">
const { t } = useI18n()

const handleCreate = () => {
  ElMessage.success(t('pages.user.message.createSuccess'))
}
</script>
```

---

## 🔧 最佳实践

### 1. 组件中使用

```vue
<template>
  <div>
    <!-- 静态文本 -->
    <h1>{{ t('pages.user.title') }}</h1>
    
    <!-- 动态参数 -->
    <p>{{ t('common.message.total') }} {{ totalCount }} {{ t('common.text.items') }}</p>
    
    <!-- 带参数的翻译 -->
    <span>{{ t('pages.user.message.batchDeleteConfirm', { count: selectedCount }) }}</span>
    
    <!-- 属性绑定 -->
    <el-input :placeholder="t('pages.user.form.placeholder.username')" />
    
    <!-- 按钮文本 -->
    <el-button>{{ t('common.button.confirm') }}</el-button>
  </div>
</template>

<script setup lang="ts">
const { t } = useI18n()
const totalCount = ref(100)
const selectedCount = ref(5)
</script>
```

### 2. JavaScript/TypeScript 中使用

```typescript
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

// 简单文本
const title = t('pages.user.title')

// 带参数
const message = t('pages.user.message.batchDeleteConfirm', { count: 5 })

// 在 API 调用中
ElMessage.success(t('pages.user.message.createSuccess'))

// 在路由守卫中
document.title = t('routes.dashboard')
```

### 3. Store 中使用

```typescript
import { i18n } from '@/i18n/setup'

const t = i18n.global.t

export const useUserStore = defineStore('user', () => {
  const statusList = computed(() => [
    { value: 'active', label: t('common.status.active') },
    { value: 'inactive', label: t('common.status.inactive') }
  ])
  
  return { statusList }
})
```

### 4. 路由配置中使用

```typescript
import { i18n } from '@/i18n/setup'

const t = i18n.global.t

const routes: RouteRecordRaw[] = [
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: Dashboard,
    meta: {
      title: t('routes.dashboard'),
      icon: 'dashboard'
    }
  },
  {
    path: '/system',
    name: 'System',
    component: Layout,
    meta: {
      title: t('routes.system.title'),
      icon: 'setting'
    },
    children: [
      {
        path: 'user',
        name: 'UserManagement',
        component: UserList,
        meta: {
          title: t('routes.system.user')
        }
      }
    ]
  }
]
```

---

## ⚠️ 注意事项

### 1. 不要在模块顶层直接使用 useI18n()

❌ **错误做法**:
```typescript
// 在 .ts 文件顶层
const { t } = useI18n() // ❌ 会报错
```

✅ **正确做法**:
```typescript
// 方式1: 使用全局 i18n 实例
import { i18n } from '@/i18n/setup'
const t = i18n.global.t

// 方式2: 在函数内部使用
export function getStatusText(status: string) {
  const { t } = useI18n()
  return t(`common.status.${status}`)
}
```

### 2. 保持语言文件同步

当添加新的翻译键时，**必须**在所有语言文件中添加对应的翻译：

```json
// zh-CN/pages/user.json
{
  "newFeature": "新功能"
}

// en-US/pages/user.json
{
  "newFeature": "New Feature"  // ✅ 必须添加
}
```

### 3. 避免硬编码文本

❌ **错误做法**:
```vue
<el-button>确定</el-button>
<p>操作成功</p>
```

✅ **正确做法**:
```vue
<el-button>{{ t('common.button.confirm') }}</el-button>
<p>{{ t('common.message.success') }}</p>
```

### 4. 合理使用参数化翻译

对于包含动态内容的文本，使用参数化翻译：

```json
{
  "message": {
    "deleteConfirm": "确认删除 {name} 吗？",
    "totalCount": "共 {count} 条记录"
  }
}
```

```typescript
t('message.deleteConfirm', { name: userName })
t('message.totalCount', { count: total })
```

### 5. 键名要有意义

❌ **不推荐**:
```json
{
  "text1": "用户名",
  "text2": "密码"
}
```

✅ **推荐**:
```json
{
  "username": "用户名",
  "password": "密码"
}
```

### 6. 避免过度嵌套

❌ **不推荐**（超过4层）:
```json
{
  "user": {
    "form": {
      "fields": {
        "basic": {
          "info": {
            "username": "用户名"
          }
        }
      }
    }
  }
}
```

✅ **推荐**（扁平化）:
```json
{
  "user": {
    "form": {
      "username": "用户名",
      "password": "密码"
    }
  }
}
```

---

## 🚀 开发流程

### 1. 新增页面文案

1. 在 `langs/zh-CN/pages/` 下创建对应的 JSON 文件
2. 编写中文文案
3. 在 `langs/en-US/pages/` 下创建相同的文件结构
4. 编写英文翻译
5. 在组件中使用 `t('pages.xxx.xxx')` 引用

### 2. 新增通用文案

1. 判断文案属于哪个层级（common/core/app/validation）
2. 在对应的 JSON 文件中添加键值对
3. 在所有语言文件中同步添加
4. 在代码中使用

### 3. 修改现有文案

1. 找到对应的 JSON 文件
2. 修改所有语言文件中的对应键
3. 确保键名不变，只修改值

---

## 📊 文案分类决策树

```
需要添加新文案
    |
    ├─ 是否全局高频复用？
    │   └─ 是 → common.json
    │
    ├─ 是否是框架核心功能（登录、错误页等）？
    │   └─ 是 → core.json
    │
    ├─ 是否是应用级公共文案（标题、枚举等）？
    │   └─ 是 → app.json
    │
    ├─ 是否是路由/菜单名称？
    │   └─ 是 → routes.json
    │
    ├─ 是否是表单验证提示？
    │   └─ 是 → validation.json
    │
    └─ 是否是页面专属文案？
        └─ 是 → pages/{module}.json
```

---

## 🌍 支持的语言

当前项目支持以下语言：

- **zh-CN**: 简体中文（默认）
- **en-US**: 英文

添加新语言的步骤：

1. 在 `langs/` 下创建新的语言目录（如 `ja-JP/`）
2. 复制现有语言的所有 JSON 文件
3. 翻译成目标语言
4. 在 i18n 配置中添加新语言
5. 在语言切换组件中添加选项

---

## 📚 参考资源

- [Vue I18n 官方文档](https://vue-i18n.intlify.dev/)
- [国际化最佳实践](https://www.w3.org/International/)
- [ISO 639-1 语言代码](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes)

---

## 📝 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2024-01-01 | 初始版本 |

---

**最后更新**: 2024-01-01  
**维护者**: 开发团队
