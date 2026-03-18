# Models 架构说明

## 📁 目录结构

```
src/models/
├── core/                  # 核心基础状态
│   ├── global.ts         # 全局配置（组合模式）
│   ├── theme.ts          # 主题管理
│   ├── language.ts       # 语言管理
│   └── loading.ts        # 加载状态
├── auth/                  # 认证相关状态
│   ├── user.ts           # 用户信息
│   ├── access.ts         # 权限管理
│   └── session.ts        # 会话管理
├── business/              # 业务数据（预留）
├── types.ts              # 类型定义
├── index.ts              # 统一导出
└── USAGE_EXAMPLES.ts     # 使用示例
```

## 🎯 设计理念

### 1. **分层设计**
- **Core 层**: 基础通用状态，所有页面都会用到
- **Auth 层**: 认证相关状态，登录授权场景
- **Business 层**: 业务数据状态，特定业务场景

### 2. **单一职责**
每个 model 只负责一个明确的状态领域，保持简单专注。

### 3. **持久化支持**
所有重要状态都通过 `StorageManager` 自动持久化到 localStorage。

### 4. **TypeScript 友好**
完整的类型定义，智能提示和类型检查。

## 📦 Model 说明

### Core Models

#### `core.global` - 全局配置（组合模式）
整合了 theme、language、loading，提供一站式全局状态管理。

**API:**
```typescript
{
  theme: ThemeMode,
  setTheme: (mode: ThemeMode) => void,
  toggleTheme: () => void,
  lang: string,
  setLang: (locale: string) => void,
  loading: boolean,
  setLoading: (isLoading: boolean) => void,
  startLoading: () => void,
  finishLoading: () => void,
  loadingError: () => void,
}
```

#### `core.theme` - 主题管理
**API:**
```typescript
{
  mode: ThemeMode,  // 'dark' | 'light' | 'system'
  setMode: (mode: ThemeMode) => void,
}
```

#### `core.language` - 语言管理
**API:**
```typescript
{
  locale: string,  // 'zh-CN' | 'en-US'
  setLocale: (locale: string) => void,
}
```

#### `core.loading` - 加载状态
**API:**
```typescript
{
  isLoading: boolean,
  error: boolean | null,
  startLoading: () => void,
  finishLoading: () => void,
  loadingError: () => void,
  clearError: () => void,
  reset: () => void,
}
```

### Auth Models

#### `auth.user` - 用户信息
**API:**
```typescript
{
  user: IUser | null,
  setUser: (user: IUser) => void,
  clearUser: () => void,
  updateUser: (updates: Partial<IUser>) => void,
}
```

#### `auth.access` - 权限管理
**API:**
```typescript
{
  permissions: string[],
  accessToken: TokenPayload | null,
  refreshToken: TokenPayload | null,
  
  setPermissions: (permissions: string[]) => void,
  addPermission: (permission: string) => void,
  removePermission: (permission: string) => void,
  setAccessToken: (token: TokenPayload | null) => void,
  setRefreshToken: (token: TokenPayload | null) => void,
  setTokens: (payload: {accessToken, refreshToken}) => void,
  clearTokens: () => void,
  resetAccess: () => void,
}
```

#### `auth.session` - 会话管理
**API:**
```typescript
{
  isAccessChecked: boolean,
  loginExpired: boolean,
  lastLoginTime?: number,
  
  setIsAccessChecked: (checked: boolean) => void,
  setLoginExpired: (expired: boolean) => void,
  setLastLoginTime: (time: number) => void,
  resetSession: () => void,
}
```

## 💡 使用方式

### 基础用法

```typescript
import {useModel} from '@umijs/max';

function MyComponent() {
  const theme = useModel('core.theme');
  const user = useModel('auth.user');
  
  return (
    <div className={theme.mode}>
      <h1>{user.user?.nickname}</h1>
    </div>
  );
}
```

### 组合用法

```typescript
import {useModel} from '@umijs/max';

function Dashboard() {
  // 使用 global model 获取所有全局状态
  const global = useModel('core.global');
  const user = useModel('auth.user');
  const access = useModel('auth.access');
  
  return (
    <div>
      <button onClick={global.toggleTheme}>切换主题</button>
      <button onClick={() => global.setLang('en-US')}>English</button>
      
      {access.permissions.includes('dashboard:view') && (
        <DashboardPanel />
      )}
    </div>
  );
}
```

### 登录流程示例

```typescript
import {useModel} from '@umijs/max';
import {history} from '@umijs/max';

function LoginPage() {
  const user = useModel('auth.user');
  const access = useModel('auth.access');
  const session = useModel('auth.session');
  const global = useModel('core.global');
  
  const handleLogin = async (credentials) => {
    try {
      global.startLoading();
      
      // 调用登录 API
      const response = await loginAPI(credentials);
      
      // 设置令牌
      access.setTokens({
        accessToken: response.accessToken,
        refreshToken: response.refreshToken,
      });
      
      // 设置用户信息
      user.setUser(response.userInfo);
      
      // 设置会话状态
      session.loginSuccess();
      
      global.finishLoading();
      history.push('/dashboard');
      
    } catch (error) {
      global.loadingError();
    }
  };
  
  return <LoginForm onFinish={handleLogin} />;
}
```

## 🔧 特性

### ✅ 自动持久化
所有重要状态都会自动保存到 localStorage，页面刷新后自动恢复。

### ✅ SSR 友好
服务端渲染环境下安全降级，只在客户端访问 localStorage。

### ✅ 类型安全
完整的 TypeScript 类型定义，开发体验更好。

### ✅ 易于测试
纯函数式的设计，易于单元测试。

## 🚀 最佳实践

1. **优先使用 global model**
   - 如果需要同时使用多个 core models，建议使用 `core.global`

2. **按需引入**
   - 只需要某个特定状态时，直接引入对应的 model

3. **避免过度使用**
   - 不是所有状态都需要放在 models 中
   - 组件内部状态优先使用 useState

4. **业务数据放 business 目录**
   - 将 API 服务封装放在 `business/` 目录下
   - 例如：`business/category.ts`, `business/post.ts`

## 📝 下一步

1. 根据业务需求，在 `business/` 目录下创建业务 models
2. 逐步替换现有的 Redux store 使用
3. 完善错误处理和边界情况
4. 添加性能优化（如防抖、节流）
