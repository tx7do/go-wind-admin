# 编辑器组件使用说明

## 概述

已成功整合五种编辑器：

1. **Tiptap Editor** - 现代化富文本编辑器（推荐使用）
2. **Markdown Editor** - Markdown 编辑器（使用 md-editor-rt）
3. **JSON Editor** - JSON 编辑器
4. **Code Editor** - 代码编辑器（使用 Monaco Editor）
5. **PlainText Editor** - 纯文本编辑器（使用 textarea）

## 文件结构

```
src/components/common/Editor/
├── index.ts              # 导出所有组件
├── types.ts              # TypeScript 类型定义
└── src/
    ├── Editor.tsx         # 主编辑器组件（根据类型动态加载）
    ├── TiptapEditor.tsx   # Tiptap 现代富文本编辑器（推荐）
    ├── MarkdownEditor.tsx # Markdown 编辑器
    ├── JsonEditor.tsx     # JSON 编辑器
    ├── CodeEditor.tsx     # 代码编辑器
    ├── PlainTextEditor.tsx# 纯文本编辑器
    ├── utils.ts           # 工具函数
    └── tiptap-editor.css  # Tiptap 编辑器样式
```

## 使用方法

### 1. 基本使用

```tsx
import { useState } from 'react';
import { Editor, EditorType } from '@/components/common/Editor';

const MyComponent = () => {
  const [content, setContent] = useState('');

  return (
    <Editor
      value={content}
      onChange={setContent}
      editorType={EditorType.RICH_TEXT}
      height={600}
      placeholder="请输入内容..."
    />
  );
};
```

### 2. 配合 antd Form.Item 使用

```tsx
import { Form } from 'antd';
import { Editor, EditorType } from '@/components/common/Editor';

const MyForm = () => {
  return (
    <Form.Item
      name="content"
      label="内容"
      rules={[{ required: true, message: '请输入内容' }]}
    >
      <Editor
        editorType={EditorType.RICH_TEXT}
        height={400}
        placeholder="请输入内容..."
      />
    </Form.Item>
  );
};
```

> `Form.Item` 会自动注入 `value` 和 `onChange`，无需手动传递。

### 3. 使用图片上传功能

```tsx
import { useState } from 'react';
import { App } from 'antd';
import { Editor, EditorType } from '@/components/common/Editor';

const MyComponent = () => {
  const [content, setContent] = useState('');
  const { message } = App.useApp();

  const handleUploadImage = async (file: File): Promise<string> => {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch('/api/upload/image', {
      method: 'POST',
      body: formData,
    });

    const data = await response.json();

    if (data.success) {
      message.success('图片上传成功');
      return data.url;
    } else {
      throw new Error(data.message);
    }
  };

  return (
    <Editor
      value={content}
      onChange={setContent}
      editorType={EditorType.RICH_TEXT}
      uploadImage={handleUploadImage}
      height={600}
    />
  );
};
```

### 4. 编辑器类型

```typescript
export const EditorType = {
  CODE: 'EDITOR_TYPE_CODE',
  JSON: 'EDITOR_TYPE_JSON_BLOCK',
  MARKDOWN: 'EDITOR_TYPE_MARKDOWN',
  PLAIN_TEXT: 'EDITOR_TYPE_PLAIN_TEXT',
  RICH_TEXT: 'EDITOR_TYPE_RICH_TEXT',
  VISUAL_BUILDER: 'EDITOR_TYPE_VISUAL_BUILDER',
} as const;

export type EditorType = (typeof EditorType)[keyof typeof EditorType];
```

### 5. 动态切换编辑器

```tsx
import { useState } from 'react';
import { Select } from 'antd';
import { Editor, EditorType } from '@/components/common/Editor';

const MyComponent = () => {
  const [content, setContent] = useState('');
  const [editorType, setEditorType] = useState<EditorType>(EditorType.RICH_TEXT);

  return (
    <>
      <Select
        value={editorType}
        onChange={setEditorType}
        style={{ width: 200, marginBottom: 16 }}
        options={[
          { value: EditorType.RICH_TEXT, label: 'Tiptap Editor（推荐）' },
          { value: EditorType.MARKDOWN, label: 'Markdown Editor' },
          { value: EditorType.CODE, label: 'Code Editor' },
          { value: EditorType.JSON, label: 'JSON Editor' },
          { value: EditorType.PLAIN_TEXT, label: 'Plain Text Editor' },
        ]}
      />
      <Editor
        value={content}
        onChange={setContent}
        editorType={editorType}
        height={600}
      />
    </>
  );
};
```

### 6. 单独使用特定编辑器

```tsx
import { useState } from 'react';
import {
  TiptapEditor,
  MarkdownEditor,
  JsonEditor,
  CodeEditor,
  PlainTextEditor,
} from '@/components/common/Editor';

const handleUploadImage = async (file: File): Promise<string> => {
  // ... 上传逻辑
  return uploadedUrl;
};

// Tiptap 富文本编辑器（推荐）
<TiptapEditor value={content} onChange={setContent} height={500} />;

// Markdown 编辑器
<MarkdownEditor value={content} onChange={setContent} height={500} />;

// JSON 编辑器
<JsonEditor value={content} onChange={setContent} height={500} />;

// 代码编辑器
<CodeEditor value={content} onChange={setContent} height={500} />;

// 纯文本编辑器
<PlainTextEditor value={content} onChange={setContent} height={500} />;
```

## Props 说明

### Editor 组件

| Prop | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| value | string | - | 编辑器内容（受控） |
| editorType | EditorType \| string | MARKDOWN | 编辑器类型 |
| height | string \| number | '100%' | 编辑器高度 |
| disabled | boolean | false | 是否禁用 |
| placeholder | string | - | 占位符文本 |
| uploadImage | (file: File) => Promise\<string\> | undefined | 图片上传回调（Tiptap 和 Markdown 支持） |
| markdownOptions | object | - | Markdown 编辑器配置 |
| jsonOptions | object | - | JSON 编辑器配置 |
| codeOptions | object | - | 代码编辑器配置 |
| onChange | (value: string) => void | - | 内容变化回调 |
| onReady | () => void | - | 编辑器加载完成回调 |

### Tiptap Editor 额外配置

```typescript
interface TiptapEditorProps {
  value: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  config?: Record<string, any>;
  showToolbar?: boolean;       // 显示工具栏（默认 true）
  showStatusBar?: boolean;     // 显示状态栏（默认 true）
  uploadImage?: (file: File) => Promise<string>;
  fullHeight?: boolean;        // 是否撑满容器（默认 true）
  onChange?: (value: string) => void;
  onReady?: (editor: any) => void;
}
```

**Tiptap Editor 功能特性：**
- ✅ 标题（H1/H2/H3）、段落格式
- ✅ 文本样式：加粗、斜体、删除线、下划线、行内代码
- ✅ 上标、下标
- ✅ 文字颜色、高亮背景
- ✅ **字体大小调整**（12px - 32px）
- ✅ **缩进控制**（增加/减少缩进）
- ✅ 列表：无序列表、有序列表、任务列表
- ✅ 引用块、代码块（支持语法高亮，200+ 种编程语言）
- ✅ 表格：插入、删除、行列操作、合并拆分、表头切换
- ✅ 水平分割线
- ✅ 文本对齐：左、中、右、两端对齐
- ✅ 插入链接、图片上传、视频插入
- ✅ 导入 Markdown 文档（.md 文件）
- ✅ 代码块内联语言选择器
- ✅ 插入视频（MP4、WebM，自定义宽度）
- ✅ 插入 Iframe（YouTube、Figma、CodePen 等嵌入式内容）
- ✅ 撤销/重做
- ✅ 清除格式、清空内容
- ✅ **全屏模式**（专注编辑）
- ✅ 暗黑模式适配
- ✅ 实时字数统计、光标位置显示

### Markdown 编辑器配置

```typescript
markdownOptions?: {
  hideModeSwitch?: boolean;           // 隐藏模式切换
  initialEditType?: 'markdown' | 'wysiwyg';  // 初始编辑模式
  previewStyle?: 'tab' | 'vertical';  // 预览样式
  toolbarItems?: string[][];           // 工具栏配置
}
```

**Markdown Editor 功能特性：**
- ✅ Markdown 语法支持（标题、列表、引用、代码块等）
- ✅ 实时预览（支持切换编辑/预览/双栏模式）
- ✅ 工具栏：格式化、插入表格、链接、图片、Emoji
- ✅ 数学公式支持（KaTeX）
- ✅ 流程图/时序图（Mermaid）
- ✅ 代码高亮
- ✅ 图片上传支持
- ✅ 暗黑模式适配

### JSON 编辑器配置

```typescript
jsonOptions?: {
  mode?: 'code' | 'form' | 'text' | 'tree' | 'view';
  modes?: string[];
  search?: boolean;
}
```

### Code 编辑器配置

```typescript
codeOptions?: {
  language?: string;       // 语言模式（默认 'javascript'）
  lineNumbers?: boolean;   // 显示行号（默认 true）
  tabSize?: number;        // Tab 缩进大小（默认 2）
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

## 在内部消息中的使用示例

```tsx
import { Form } from 'antd';
import { Editor, EditorType } from '@/components/common/Editor';

// 配合 DrawerForm / Form 使用
<Form.Item
  name="content"
  label="消息内容"
  rules={[{ required: true, message: '请输入消息内容' }]}
>
  <Editor
    editorType={EditorType.RICH_TEXT}
    height={400}
    placeholder="请输入消息内容"
  />
</Form.Item>
```

## 扩展开发

### 添加新的编辑器

1. 在 `src/` 下创建新编辑器组件（如 `NewEditor.tsx`）
2. 在 `types.ts` 中添加新的编辑器类型
3. 在 `Editor.tsx` 的 switch 语句中添加新的 case
4. 在 `index.ts` 中导出新组件
5. 通过 `React.lazy` 实现代码分割

## 注意事项

### 图片上传相关

1. **返回值要求**: `uploadImage` 函数必须返回 `Promise<string>`，字符串为图片的访问 URL
2. **支持编辑器**: 目前只有 **Tiptap Editor** 和 **Markdown Editor** 支持图片上传功能
3. **错误处理**: 建议在 `uploadImage` 函数中添加错误提示，提升用户体验
4. **文件验证**: 可以在 `uploadImage` 中添加文件大小、类型验证

### 文件验证示例

```typescript
const handleUploadImage = async (file: File): Promise<string> => {
  // 验证文件类型
  const allowedTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp'];
  if (!allowedTypes.includes(file.type)) {
    throw new Error('只支持 JPG、PNG、GIF、WebP 格式的图片');
  }

  // 验证文件大小（5MB）
  const maxSize = 5 * 1024 * 1024;
  if (file.size > maxSize) {
    throw new Error('图片大小不能超过 5MB');
  }

  // 执行上传...
};
```

### 受控组件说明

Editor 组件遵循 React 受控组件模式：
- 通过 `value` + `onChange` 实现双向绑定
- 配合 `Form.Item` 使用时，antd 会自动注入 `value` 和 `onChange`
- 所有子编辑器（Tiptap、Markdown、JSON、Code、PlainText）均支持受控模式

### 性能优化

- 所有子编辑器通过 `React.lazy` + `Suspense` 实现懒加载
- 只有实际使用的编辑器才会被加载，减少初始包体积
- Tiptap Editor 使用 `useMemo` 优化工具栏按钮的渲染性能
