export enum EditorType {
  CODE = 'EDITOR_TYPE_CODE',
  JSON = 'EDITOR_TYPE_JSON_BLOCK',
  MARKDOWN = 'EDITOR_TYPE_MARKDOWN',
  PLAIN_TEXT = 'EDITOR_TYPE_PLAIN_TEXT',
  RICH_TEXT = 'EDITOR_TYPE_RICH_TEXT',
  VISUAL_BUILDER = 'EDITOR_TYPE_VISUAL_BUILDER',
}

export interface EditorProps {
  modelValue: string;
  editorType?: EditorType | string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  uploadImage?: (file: File) => Promise<string>;
  // Markdown specific
  markdownOptions?: {
    hideModeSwitch?: boolean;
    initialEditType?: 'markdown' | 'wysiwyg';
    previewStyle?: 'tab' | 'vertical';
    toolbarItems?: string[][];
  };
  // JSON Editor specific
  jsonOptions?: {
    mode?: 'code' | 'form' | 'text' | 'tree' | 'view';
    modes?: string[];
    search?: boolean;
  };
  // Code Editor specific
  codeOptions?: {
    language?: string;
    lineNumbers?: boolean;
    tabSize?: number;
    /** 关闭 hljs 自动语言探测，强制使用 language 指定的高亮（探测表未收录的语言如 lua 会被误判为 plaintext） */
    autoDetectLanguage?: boolean;
  };
}

export interface EditorEmits {
  (e: 'change' | 'update:modelValue', value: string): void;
  (e: 'ready'): void;
}
