# 编辑器组件使用说明

## 概述

已成功整合四种编辑器：

1. **Tiptap Editor** - 现代化富文本编辑器（推荐使用）
2. **Markdown Editor** - Markdown 编辑器（使用 `md-editor-v3`）
3. **JSON Editor** - JSON 编辑器（使用 `json-editor-vue`）
4. **Code Editor** - 代码编辑器（使用 `monaco-editor`）
5. **PlainText Editor** - 纯文本编辑器（使用 `textarea`）

## 文件结构

```
/src/components/Editor/
├── Editor.vue          # 主编辑器组件（根据类型动态加载）
├── PlainTextEditor.vue # 纯文本编辑器
├── TiptapEditor.vue    # Tiptap 现代富文本编辑器（推荐）
├── MarkdownEditor.vue  # Markdown 编辑器
├── JsonEditor.vue      # JSON 编辑器
├── CodeEditor.vue      # 代码编辑器
├── types.ts            # TypeScript 类型定义
├── utils.ts            # 工具函数
└── index.ts            # 导出所有组件
```

## 使用方法

### 1. 基本使用

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Editor, EditorType } from '#/components/editor';

const content = ref('');
const editorType = ref(EditorType.RICH_TEXT); // 默认使用 Tiptap Editor
</script>

<template>
  <Editor
    v-model="content"
    :editor-type="editorType"
    :height="600"
    placeholder="Enter your content..."
  />
</template>
```

### 2. 使用图片上传功能

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Editor, EditorType } from '#/components/editor';
import { message } from 'ant-design-vue';

const content = ref('');

// 图片上传处理函数
const handleUploadImage = async (file: File): Promise<string> => {
  try {
    // 创建 FormData
    const formData = new FormData();
    formData.append('file', file);
    
    // 调用上传接口
    const response = await fetch('/api/upload/image', {
      method: 'POST',
      body: formData,
    });
    
    const data = await response.json();
    
    if (data.success) {
      message.success('Image uploaded successfully');
      return data.url; // 返回图片 URL
    } else {
      throw new Error(data.message);
    }
  } catch (error) {
    message.error('Image upload failed');
    throw error;
  }
};
</script>

<template>
  <Editor
    v-model="content"
    :editor-type="EditorType.RICH_TEXT"
    :upload-image="handleUploadImage"
    :height="600"
  />
</template>
```

### 3. 编辑器类型

```typescript
export enum EditorType {
  CODE = 'EDITOR_TYPE_CODE',
  JSON = 'EDITOR_TYPE_JSON_BLOCK',
  MARKDOWN = 'EDITOR_TYPE_MARKDOWN',
  PLAIN_TEXT = 'EDITOR_TYPE_PLAIN_TEXT',
  RICH_TEXT = 'EDITOR_TYPE_RICH_TEXT',
  VISUAL_BUILDER = 'EDITOR_TYPE_VISUAL_BUILDER',
}
```

### 4. 动态切换编辑器

```vue
<template>
  <!-- 编辑器类型选择 -->
  <a-select v-model:value="editorType">
    <a-select-option :value="EditorType.RICH_TEXT">
      Tiptap Editor (推荐)
    </a-select-option>
    <a-select-option :value="EditorType.MARKDOWN">
      Markdown Editor
    </a-select-option>
    <a-select-option :value="EditorType.CODE">
      Code Editor
    </a-select-option>
    <a-select-option :value="EditorType.JSON">
      JSON Editor
    </a-select-option>
    <a-select-option :value="EditorType.PLAIN_TEXT">
      Plain Text Editor
    </a-select-option>
  </a-select>

  <!-- 编辑器 -->
  <Editor
    v-model="content"
    :editor-type="editorType"
    :height="600"
  />
</template>
```

### 5. 单独使用特定编辑器

```vue
<script setup lang="ts">
import {
  TiptapEditor,
  MarkdownEditor,
  JsonEditor,
  CodeEditor,
  PlainTextEditor,
} from '#/components/editor';

// 图片上传处理（Tiptap 和 Markdown 支持）
const handleUploadImage = async (file: File): Promise<string> => {
  // ... 上传逻辑
  return uploadedUrl;
};
</script>

<template>
  <!-- Tiptap 现代富文本编辑器（推荐，支持图片上传） -->
  <TiptapEditor
    v-model="content"
    :height="500"
    :show-toolbar="true"
    :show-status-bar="true"
    :upload-image="handleUploadImage"
  />

  <!-- Markdown 编辑器（支持图片上传） -->
  <MarkdownEditor
    v-model="markdown"
    :height="500"
    :upload-image="handleUploadImage"
  />

  <!-- JSON 编辑器 -->
  <JsonEditor
    v-model="jsonData"
    :height="500"
  />

  <!-- 代码编辑器 -->
  <CodeEditor
    v-model="code"
    language="javascript"
    :height="500"
  />

  <!-- 纯文本编辑器 -->
  <PlainTextEditor
    v-model="text"
    :height="500"
  />
</template>
```

## Props 说明

### Editor 组件

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| modelValue | string | - | 编辑器内容（v-model） |
| editorType | EditorType \| string | RICH_TEXT | 编辑器类型 |
| height | string \| number | 500 | 编辑器高度 |
| disabled | boolean | false | 是否禁用 |
| placeholder | string | 'Please enter content...' | 占位符文本 |
| **uploadImage** | **(file: File) => Promise\<string\>** | **undefined** | **图片上传回调函数，返回上传后的图片 URL（Tiptap 和 Markdown 支持）** |
| markdownOptions | object | {} | Markdown 编辑器配置 |
| jsonOptions | object | {} | JSON 编辑器配置 |
| tiptapOptions | object | {} | Tiptap 编辑器配置 |
| codeOptions | object | {} | 代码编辑器配置 |
| plainTextOptions | object | {} | 纯文本编辑器配置 |

### Tiptap Editor 配置

```typescript
{
  height: number | string,           // 编辑器高度
  disabled: boolean,                  // 是否禁用
  placeholder: string,                // 占位符
  showToolbar: boolean,               // 显示工具栏（默认 true）
  showStatusBar: boolean,             // 显示状态栏（默认 true）
  uploadImage: (file: File) => Promise<string>,  // 图片上传回调
  fullHeight: boolean,                // 是否撑满容器（默认 true）
}
```

**Tiptap Editor 功能特性：**
- ✅ 标题（H1/H2/H3）、段落格式
- ✅ 文本样式：加粗、斜体、删除线、下划线、行内代码
- ✅ 上标、下标
- ✅ 文字颜色、高亮背景
- ✅ 列表：无序列表、有序列表、任务列表
- ✅ 引用块、**代码块（支持语法高亮，200+ 种编程语言）**
- ✅ 表格：插入、删除、行列操作、合并拆分、表头切换
- ✅ 水平分割线
- ✅ 文本对齐：左、中、右、两端对齐
- ✅ 插入链接、图片上传、**视频插入**
- ✅ **导入 Markdown 文档**（.md 文件）
- ✅ **插入代码块**（支持 JavaScript、TypeScript、Python、Java、C++、Go 等 200+ 种语言，右上角内联语言选择器）
- ✅ **插入视频**（支持 MP4、WebM 等格式，自定义宽度）
- ✅ **插入 Iframe**（支持 YouTube、Figma、CodePen 等嵌入式内容）
- ✅ 撤销/重做
- ✅ 清除格式、清空内容
- ✅ 暗黑模式适配
- ✅ 实时字数统计、光标位置显示


### Markdown 编辑器配置

```typescript
{
  height: number | string,            // 编辑器高度（支持 '100%' 撑满容器）
  disabled: boolean,                  // 是否禁用
  placeholder: string,                // 占位符
  enableExport: boolean,              // 启用导出功能（PDF/HTML）
  uploadImage: (file: File) => Promise<string>,  // 图片上传回调
  options: Partial<EditorProps>,      // md-editor-v3 原生配置
}
```

**Markdown Editor 功能特性：**
- ✅ Markdown 语法支持（标题、列表、引用、代码块等）
- ✅ 实时预览（支持切换编辑/预览/双栏模式）
- ✅ 工具栏：格式化、插入表格、链接、图片、Emoji
- ✅ 数学公式支持（KaTeX）
- ✅ 流程图/时序图（Mermaid）
- ✅ 代码高亮
- ✅ **保存为 .md 文件**（另存为本地文件）
- ✅ 图片上传支持
- ✅ 暗黑模式适配
- ✅ 响应式高度调整

### JSON 编辑器配置

```typescript
{
  deep: number,                       // 展开深度（默认 3）
  showLength: boolean,                // 显示数组/对象长度（默认 true）
  showLine: boolean,                  // 显示行号（默认 true）
  showDoubleQuotes: boolean,          // 显示双引号（默认 true）
  collapsedOnClickBrackets: boolean,  // 点击括号折叠（默认 true）
  editable: boolean,                  // 是否可编辑（默认 false）
}
```

**JSON Editor 功能特性：**
- ✅ 树形视图展示
- ✅ 语法高亮
- ✅ 折叠/展开节点
- ✅ 数据类型标识
- ✅ 复制节点路径
- ✅ 在线编辑（可选）
- ✅ 暗黑模式适配

### Code Editor 配置

```typescript
{
  language: string,                   // 语言模式（默认 'javascript'）
  theme: 'light' | 'dark',            // 主题（默认自动跟随系统）
  readonly: boolean,                  // 是否只读（默认 false）
  lineNumbers: boolean,               // 显示行号（默认 true）
  lineWrapping: boolean,              // 自动换行（默认 false）
  tabSize: number,                    // Tab 缩进大小（默认 2）
}
```

**Code Editor 功能特性：**
- ✅ 多语言支持（JavaScript、TypeScript、JSON、HTML、CSS、Python 等）
- ✅ 语法高亮
- ✅ 代码补全
- ✅ 括号匹配
- ✅ 代码折叠
- ✅ 行号显示
- ✅ 暗黑模式适配
- ✅ 多光标编辑
- ✅ 搜索替换

### PlainText 编辑器配置

```typescript
{
  height: number | string,            // 编辑器高度
  disabled: boolean,                  // 是否禁用
  placeholder: string,                // 占位符
  rows: number,                       // textarea 行数（默认 10）
  maxlength: number,                  // 最大字符数
  showCount: boolean,                 // 显示字符计数（默认 false）
  autoSize: boolean | object,         // 自适应高度
}
```

**PlainText Editor 功能特性：**
- ✅ 简洁轻量
- ✅ 纯文本输入
- ✅ 字符计数
- ✅ 自适应高度
- ✅ 暗黑模式适配

## Events

| Event | 参数 | 说明 |
|-------|------|------|
| update:modelValue | value: string | 内容更新 |
| change | value: string | 内容变化 |
| ready | editor?: any | 编辑器加载完成（部分编辑器返回实例） |
| imageUpload | file: File | 图片上传事件（Tiptap/Markdown） |
| error | error: Error | 错误事件（JSON 编辑器） |

## 在 Post 编辑页面中的使用

已经在 `/content/posts/create` 和 `/content/posts/edit/:id` 页面中集成了编辑器：

```vue
<!-- 编辑器类型选择 -->
<a-form-item label="Editor Type" name="editorType">
  <a-select v-model:value="formData.editorType">
    <a-select-option :value="EditorType.RICH_TEXT">
      Tiptap Editor (推荐)
    </a-select-option>
    <a-select-option :value="EditorType.MARKDOWN">
      Markdown Editor
    </a-select-option>
    <a-select-option :value="EditorType.CODE">
      Code Editor
    </a-select-option>
    <a-select-option :value="EditorType.JSON">
      JSON Editor
    </a-select-option>
    <a-select-option :value="EditorType.PLAIN_TEXT">
      Plain Text Editor
    </a-select-option>
  </a-select>
</a-form-item>

<!-- 内容编辑器 -->
<a-form-item label="Content" name="content" required>
  <Editor
    v-model="formData.content"
    :editor-type="formData.editorType"
    :height="600"
    placeholder="Enter your content here..."
  />
</a-form-item>
```

## 扩展开发

### 添加新的编辑器

1. 在 `components/editor/` 下创建新编辑器组件
2. 在 `types.ts` 中添加新的编辑器类型
3. 在 `editor.vue` 的 switch 语句中添加新的 case
4. 在 `index.ts` 中导出新组件

### 示例：添加代码编辑器

```vue
<!-- CodeEditor.vue -->
<script setup lang="ts">
import { ref } from 'vue';
// import Monaco Editor or CodeMirror
</script>

<template>
  <div class="code-editor">
    <!-- 编辑器实现 -->
  </div>
</template>
```

然后在 `editor.vue` 中添加：

```typescript
case EditorType.CODE: {
  return CodeEditor;
}
```

## 图片上传完整示例

### 使用 axios 上传

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Editor, EditorType } from '#/components/editor';
import { message } from 'ant-design-vue';
import axios from 'axios';

const content = ref('');

// 方式1：使用 axios
const handleUploadImage = async (file: File): Promise<string> => {
  try {
    const formData = new FormData();
    formData.append('file', file);
    
    const { data } = await axios.post('/api/upload/image', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    
    if (data.success) {
      message.success('图片上传成功');
      return data.url;
    }
    
    throw new Error(data.message || '上传失败');
  } catch (error) {
    message.error(`图片上传失败: ${error.message}`);
    throw error;
  }
};
</script>

<template>
  <Editor
    v-model="content"
    :editor-type="EditorType.RICH_TEXT"
    :upload-image="handleUploadImage"
    :height="600"
  />
</template>
```

### 使用 OSS 直传

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { TiptapEditor } from '#/components/editor';
import { message } from 'ant-design-vue';
import OSS from 'ali-oss';

const content = ref('');

// 方式2：使用 OSS 直传
const handleUploadImage = async (file: File): Promise<string> => {
  try {
    // 1. 获取 OSS 临时凭证
    const { data: credentials } = await axios.get('/api/oss/credentials');
    
    // 2. 初始化 OSS 客户端
    const client = new OSS({
      region: credentials.region,
      accessKeyId: credentials.accessKeyId,
      accessKeySecret: credentials.accessKeySecret,
      stsToken: credentials.securityToken,
      bucket: credentials.bucket,
    });
    
    // 3. 生成文件名
    const timestamp = Date.now();
    const ext = file.name.split('.').pop();
    const fileName = `images/${timestamp}.${ext}`;
    
    // 4. 上传到 OSS
    const result = await client.put(fileName, file);
    
    message.success('图片上传成功');
    return result.url;
  } catch (error) {
    message.error(`图片上传失败: ${error.message}`);
    throw error;
  }
};
</script>

<template>
  <TiptapEditor
    v-model="content"
    :upload-image="handleUploadImage"
    :height="600"
  />
</template>
```

### 插入代码块功能

Tiptap Editor 支持插入带语法高亮的代码块，使用 `lowlight` 的 `all` 预设支持 **200+ 种编程语言**：

**常用语言列表（部分）：**
- JavaScript, TypeScript, Python, Java, C++, C, C#
- Go, Rust, PHP, Ruby, Swift, Kotlin
- HTML, CSS, SCSS, Less, JSON, YAML, XML
- SQL, Bash, Shell, PowerShell, Markdown, Plain Text
- 以及其他 180+ 种语言

**技术实现：**
```typescript
import { all, createLowlight } from 'lowlight';
// all 预设包含所有 200+ 种语言，无需额外注册
const lowlight = createLowlight(all);
```

> **注意**：使用 `all` 预设会包含所有语言，打包体积较大（~500KB）。如果需要优化体积，可以改用 `common` 预设 + 按需注册：
> ```typescript
> import { common, createLowlight } from 'lowlight';
> import python from 'highlight.js/lib/languages/python';
> const lowlight = createLowlight(common);
> lowlight.register('python', python); // 仅注册需要的语言
> ```

**使用方式：**

#### 方式 1：通过工具栏插入
1. 点击工具栏中的"代码块"按钮（`</>`图标）
2. 在弹出的对话框中选择编程语言
3. 输入代码内容
4. 点击确认插入

#### 方式 2：直接在代码块内切换语言（推荐✨）
1. 点击代码块进入编辑状态
2. 在代码块右上角的下拉框中选择语言
3. 语法高亮会自动更新
4. 无需重新插入代码块

**代码示例：**

```vue
<template>
  <TiptapEditor
    v-model="content"
    :height="600"
  />
</template>
```

**功能特性：**
- ✅ 语法高亮显示（200+ 种语言）
- ✅ **右上角内联语言选择器**（实时切换语言）
- ✅ 使用等宽字体
- ✅ 支持代码块内编辑
- ✅ 暗黑模式自动适配
- ✅ 悬停效果和交互反馈

### 插入视频功能

Tiptap Editor 支持插入视频文件，支持常见的视频格式：

**支持的视频格式：**
- MP4 (H.264)
- WebM
- Ogg
- 其他浏览器原生支持的格式

**使用方式：**

1. 点击工具栏中的"视频"按钮（📹 图标）
2. 在弹出的对话框中输入视频URL
3. 选择视频宽度（100%, 75%, 50%, 640px, 800px）
4. 点击确认插入

**功能特性：**
- ✅ 支持多种视频格式
- ✅ 自定义视频宽度
- ✅ 响应式适配（自动调整高度）
- ✅ 原生 HTML5 video 控件
- ✅ 悬停阴影效果
- ✅ 选中时高亮边框
- ✅ 暗黑模式适配
- ✅ 可拖拽移动位置

**代码示例：**

```vue
<template>
  <TiptapEditor
    v-model="content"
    :height="600"
  />
</template>
```

**视频URL示例：**
- 相对路径：`/videos/sample.mp4`
- 绝对路径：`https://example.com/video.mp4`
- 本地上传：需配合视频上传功能使用

**注意事项：**
1. 视频文件需要可通过URL访问
2. 建议使用 MP4 格式以获得最佳兼容性
3. 大视频文件建议使用 CDN 或视频托管服务
4. 视频会自动显示原生浏览器控件（播放、暂停、音量等）

### 插入 Iframe 功能

Tiptap Editor 支持插入 iframe，用于嵌入外部内容如 YouTube 视频、Figma 设计、CodePen 演示等：

**支持的内容类型：**
- YouTube 视频
- Figma 设计
- CodePen 演示
- Google 文档
- 其他支持 iframe 嵌入的服务

**使用方式：**

1. 点击工具栏中的 iframe 按钮（框架图标）
2. 在弹出的对话框中输入 iframe URL
3. 选择 iframe 的宽度和高度
4. （可选）输入 iframe 标题
5. 设置是否允许全屏
6. 点击确认插入

**功能特性：**
- ✅ 支持多种嵌入式内容源
- ✅ 自定义宽度（100%, 75%, 50%, 640px, 800px）
- ✅ 自定义高度（300px, 500px, 100%）
- ✅ 可选的 iframe 标题
- ✅ 全屏控制开关
- ✅ 响应式设计
- ✅ 暗黑模式适配
- ✅ 可拖拽移动位置

**代码示例：**

```vue
<template>
  <TiptapEditor v-model="content" :height="600" />
</template>
```

**常见 iframe URL 示例：**

- YouTube：`https://www.youtube.com/embed/VIDEO_ID`
- Figma：`https://www.figma.com/embed?url=YOUR_FIGMA_URL`
- CodePen：`https://codepen.io/USERNAME/embed/SLUG`
- Google 文档：`https://docs.google.com/document/d/DOC_ID/preview`

**注意事项：**
1. 确保使用正确的 embed URL（不是普通链接）
2. 某些服务可能需要登录或权限才能访问
3. 嵌入内容的加载速度取决于源服务器
4. 不是所有网站都支持 iframe 嵌入（受 X-Frame-Options 限制）
5. 全屏功能取决于 iframe 源是否允许

### 在 Post 编辑页面中使用

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Editor, EditorType } from '#/components/editor';
import { useFileStore } from '#/stores/file.state';

const fileStore = useFileStore();
const formData = ref({
  title: '',
  content: '',
  editorType: EditorType.RICH_TEXT,
});

// 复用已有的文件上传服务
const handleUploadImage = async (file: File): Promise<string> => {
  try {
    const uploadedFile = await fileStore.uploadFile(file);
    return uploadedFile.url;
  } catch (error) {
    console.error('Upload failed:', error);
    throw error;
  }
};
</script>

<template>
  <a-form>
    <a-form-item label="Content" name="content">
      <Editor
        v-model="formData.content"
        :editor-type="formData.editorType"
        :upload-image="handleUploadImage"
        :height="600"
      />
    </a-form-item>
  </a-form>
</template>
```

## 注意事项

### 图片上传相关

1. **返回值要求**: `uploadImage` 函数必须返回 `Promise<string>`，字符串为图片的访问 URL
2. **支持编辑器**: 目前只有 **Tiptap Editor** 和 **Markdown Editor** 支持图片上传功能
3. **错误处理**: 建议在 `uploadImage` 函数中添加错误提示，提升用户体验
4. **文件验证**: 可以在 `uploadImage` 中添加文件大小、类型验证
5. **Loading 状态**: 编辑器会自动显示上传中的状态，无需手动处理

### 文件验证示例

```typescript
const handleUploadImage = async (file: File): Promise<string> => {
  // 验证文件类型
  const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
  if (!allowedTypes.includes(file.type)) {
    message.error('只支持 JPG、PNG、GIF、WebP 格式的图片');
    throw new Error('Invalid file type');
  }
  
  // 验证文件大小（5MB）
  const maxSize = 5 * 1024 * 1024;
  if (file.size > maxSize) {
    message.error('图片大小不能超过 5MB');
    throw new Error('File too large');
  }
  
  // 执行上传
  // ...
};
```
