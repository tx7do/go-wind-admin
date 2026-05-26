import type { Level } from '@tiptap/extension-heading';
import { useCallback, useMemo, useRef, useState } from 'react';
import { Modal } from 'antd';
import { ExclamationCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marked } from 'marked';
import { isDarkMode } from '../utils';

interface UseTiptapEditorProps {
  value: string;
  disabled?: boolean;
  placeholder?: string;
  uploadImage?: (file: File) => Promise<string>;
  onChange?: (value: string) => void;
  onReady?: (editor: any) => void;
}

export const useTiptapEditor = ({
  value,
  disabled = false,
  uploadImage,
  onChange,
  onReady,
}: UseTiptapEditorProps) => {
  const { t } = useTranslation();
  const [isDark, setIsDark] = useState(isDarkMode());
  const contentRef = useRef(value);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const markdownInputRef = useRef<HTMLInputElement>(null);
  const isInternalUpdate = useRef(false);

  // Modal states
  const [linkModalVisible, setLinkModalVisible] = useState(false);
  const [linkUrl, setLinkUrl] = useState('');
  const [codeBlockModalVisible, setCodeBlockModalVisible] = useState(false);
  const [codeBlockLanguage, setCodeBlockLanguage] = useState('javascript');
  const [codeBlockContent, setCodeBlockContent] = useState('');
  const [videoModalVisible, setVideoModalVisible] = useState(false);
  const [videoUrl, setVideoUrl] = useState('');
  const [videoWidth, setVideoWidth] = useState('100%');
  const [iframeModalVisible, setIframeModalVisible] = useState(false);
  const [iframeUrl, setIframeUrl] = useState('');
  const [iframeWidth, setIframeWidth] = useState('100%');
  const [iframeHeight, setIframeHeight] = useState('500px');
  const [iframeTitle, setIframeTitle] = useState('');
  const [iframeAllowFullscreen, setIframeAllowFullscreen] = useState(true);
  const [textColor, setTextColor] = useState('#000000');
  const [highlightColor, setHighlightColor] = useState('#FFFF00');
  const [fontSize, setFontSize] = useState('16px');
  const [lineHeight, setLineHeight] = useState('1.5');
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Dark mode tracking
  const setupDarkModeTracking = useCallback(() => {
    const checkDark = () => setIsDark(isDarkMode());
    checkDark();
    const observer = new MutationObserver(checkDark);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class', 'data-theme'],
    });
    return () => observer.disconnect();
  }, []);

  // Editor config
  const editorConfig = useMemo(
    () => ({
      content: value,
      editable: !disabled,
      autofocus: 'end' as const,
      editorProps: {
        attributes: { class: 'prose dark:prose-invert focus:outline-none min-h-full' },
      },
      onCreate: ({ editor: e }: { editor: any }) => {
        onReady?.(e);
      },
      onUpdate: ({ editor: e }: { editor: any }) => {
        if (isInternalUpdate.current) {
          isInternalUpdate.current = false;
          return;
        }
        const html = e.getHTML();
        contentRef.current = html;
        onChange?.(html);
      },
    }),
    [value, disabled, onReady, onChange],
  );

  // Sync external value
  const syncValue = useCallback((editor: any, newValue: string) => {
    if (editor && newValue !== contentRef.current) {
      isInternalUpdate.current = true;
      editor.commands.setContent(newValue);
      contentRef.current = newValue;
    }
  }, []);

  // Toolbar actions
  const createToolbarActions = useCallback(
    (editor: any) => ({
      toggleBold: () => editor?.chain().focus().toggleBold().run(),
      toggleItalic: () => editor?.chain().focus().toggleItalic().run(),
      toggleStrike: () => editor?.chain().focus().toggleStrike().run(),
      toggleUnderline: () => editor?.chain().focus().toggleUnderline().run(),
      toggleCode: () => editor?.chain().focus().toggleCode().run(),
      toggleHeading: (level: Level) => editor?.chain().focus().toggleHeading({ level }).run(),
      toggleBulletList: () => editor?.chain().focus().toggleBulletList().run(),
      toggleOrderedList: () => editor?.chain().focus().toggleOrderedList().run(),
      toggleTaskList: () => editor?.chain().focus().toggleTaskList().run(),
      insertCodeBlock: () => {
        setCodeBlockLanguage('javascript');
        setCodeBlockContent('');
        setCodeBlockModalVisible(true);
      },
      toggleBlockquote: () => editor?.chain().focus().toggleBlockquote().run(),
      toggleSubscript: () => editor?.chain().focus().toggleSubscript().run(),
      toggleSuperscript: () => editor?.chain().focus().toggleSuperscript().run(),
      setParagraph: () => editor?.chain().focus().setParagraph().run(),
      clearFormatting: () => editor?.chain().focus().unsetAllMarks().clearNodes().run(),
      insertHorizontalRule: () => editor?.chain().focus().setHorizontalRule().run(),
      insertTable: () =>
        editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(),
      deleteTable: () => editor?.chain().focus().deleteTable().run(),
      addRowBefore: () => editor?.chain().focus().addRowBefore().run(),
      addRowAfter: () => editor?.chain().focus().addRowAfter().run(),
      deleteRow: () => editor?.chain().focus().deleteRow().run(),
      addColumnBefore: () => editor?.chain().focus().addColumnBefore().run(),
      addColumnAfter: () => editor?.chain().focus().addColumnAfter().run(),
      deleteColumn: () => editor?.chain().focus().deleteColumn().run(),
      mergeCells: () => editor?.chain().focus().mergeCells().run(),
      splitCell: () => editor?.chain().focus().splitCell().run(),
      toggleHeaderRow: () => editor?.chain().focus().toggleHeaderRow().run(),
      toggleHeaderColumn: () => editor?.chain().focus().toggleHeaderColumn().run(),
      toggleHeaderCell: () => editor?.chain().focus().toggleHeaderCell().run(),
      setAlign: (align: 'center' | 'justify' | 'left' | 'right') =>
        editor?.chain().focus().setTextAlign(align).run(),
      setTextColor: (color: string) => editor?.chain().focus().setColor(color).run(),
      setHighlight: (color: string) => editor?.chain().focus().toggleHighlight({ color }).run(),
      setFontSize: (size: string) => {
        editor?.chain().focus().setMark('textStyle', { fontSize: size }).run();
        setFontSize(size);
      },
      setLineHeight: (height: string) => {
        editor?.chain().focus().setLineHeight(height).run();
        setLineHeight(height);
      },
      indent: () => editor?.chain().focus().sinkListItem('listItem').run(),
      outdent: () => editor?.chain().focus().liftListItem('listItem').run(),
      toggleFullscreen: () => setIsFullscreen((prev) => !prev),
      uploadImage: () => fileInputRef.current?.click(),
      insertVideo: () => {
        setVideoUrl('');
        setVideoWidth('100%');
        setVideoModalVisible(true);
      },
      insertIframe: () => {
        setIframeUrl('');
        setIframeWidth('100%');
        setIframeHeight('500px');
        setIframeTitle('');
        setIframeAllowFullscreen(true);
        setIframeModalVisible(true);
      },
      importMarkdown: () => markdownInputRef.current?.click(),
      setLinkModalVisible,
      undo: () => editor?.chain().focus().undo().run(),
      redo: () => editor?.chain().focus().redo().run(),
      clearContent: () => {
        Modal.confirm({
          title: t('common.confirm', '确认'),
          icon: <ExclamationCircleOutlined />,
          content: t('editor:clear_content_confirm', '确定要清空所有内容吗？'),
          okText: t('common.confirm', '确定'),
          cancelText: t('common.cancel', '取消'),
          onOk() {
            editor?.commands.setContent('');
          },
        });
      },
    }),
    [t],
  );

  // Image upload handler
  const handleImageUpload = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>, editor: any) => {
      const file = event.target.files?.[0];
      if (!file || !uploadImage) return;
      try {
        const url = await uploadImage(file);
        if (url && editor) {
          editor.chain().focus().setImage({ src: url }).run();
        }
      } catch (error) {
        console.error('Image upload failed:', error);
      } finally {
        event.target.value = '';
      }
    },
    [uploadImage],
  );

  // Markdown import handler
  const handleMarkdownImport = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>, editor: any) => {
      const file = event.target.files?.[0];
      if (!file) return;
      try {
        const md = await file.text();
        const html = marked.parse(md);
        if (editor) {
          editor.commands.setContent(String(html));
        }
      } catch (error) {
        console.error('Markdown import failed:', error);
      } finally {
        event.target.value = '';
      }
    },
    [],
  );

  // Modal handlers
  const handleLinkOk = useCallback(
    (editor: any) => {
      const url = linkUrl.trim();
      if (url && editor) {
        editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run();
      }
      setLinkModalVisible(false);
      setLinkUrl('');
    },
    [linkUrl],
  );

  const handleCodeBlockOk = useCallback(
    (editor: any) => {
      const code = codeBlockContent.trim();
      if (code && editor) {
        editor
          .chain()
          .focus()
          .insertContent({
            type: 'codeBlock',
            attrs: { language: codeBlockLanguage },
            content: [{ type: 'text', text: code }],
          })
          .run();
      }
      setCodeBlockModalVisible(false);
      setCodeBlockContent('');
    },
    [codeBlockContent, codeBlockLanguage],
  );

  const handleVideoOk = useCallback(
    (editor: any) => {
      const url = videoUrl.trim();
      if (url && editor) {
        (editor.chain().focus() as any).setVideo({ src: url, width: videoWidth }).run();
      }
      setVideoModalVisible(false);
    },
    [videoUrl, videoWidth],
  );

  const handleIframeOk = useCallback(
    (editor: any) => {
      const url = iframeUrl.trim();
      if (url && editor) {
        (editor.chain().focus() as any)
          .setIframe({
            src: url,
            width: iframeWidth,
            height: iframeHeight,
            title: iframeTitle || undefined,
            allowfullscreen: iframeAllowFullscreen,
          })
          .run();
      }
      setIframeModalVisible(false);
    },
    [iframeUrl, iframeWidth, iframeHeight, iframeTitle, iframeAllowFullscreen],
  );

  // Status info
  const getStatusInfo = useCallback((editor: any) => {
    if (!editor) return { chars: 0, words: 0, cursor: '0:0' };
    const text = editor.getText();
    const chars = text.length;
    const words = text.trim() ? text.trim().split(/\s+/).length : 0;
    const { from } = editor.state.selection;
    const doc = editor.state.doc;
    let col = 1;
    let line = 1;
    let pos = 1;
    doc.descendants((node: any) => {
      if (node.isText) {
        const t = node.text || '';
        for (const ch of t) {
          if (pos === from) return false;
          if (ch === '\n') {
            line++;
            col = 1;
          } else {
            col++;
          }
          pos++;
        }
      }
      return true;
    });
    return { chars, words, cursor: `${line}:${col}` };
  }, []);

  return {
    isDark,
    contentRef,
    fileInputRef,
    markdownInputRef,
    linkModalVisible,
    setLinkModalVisible,
    linkUrl,
    setLinkUrl,
    codeBlockModalVisible,
    setCodeBlockModalVisible,
    codeBlockLanguage,
    setCodeBlockLanguage,
    codeBlockContent,
    setCodeBlockContent,
    videoModalVisible,
    setVideoModalVisible,
    videoUrl,
    setVideoUrl,
    videoWidth,
    setVideoWidth,
    iframeModalVisible,
    setIframeModalVisible,
    iframeUrl,
    setIframeUrl,
    iframeWidth,
    setIframeWidth,
    iframeHeight,
    setIframeHeight,
    iframeTitle,
    setIframeTitle,
    iframeAllowFullscreen,
    setIframeAllowFullscreen,
    textColor,
    setTextColor,
    highlightColor,
    setHighlightColor,
    fontSize,
    setFontSize,
    lineHeight,
    setLineHeight,
    isFullscreen,
    setIsFullscreen,
    editorConfig,
    syncValue,
    createToolbarActions,
    handleImageUpload,
    handleMarkdownImport,
    handleLinkOk,
    handleCodeBlockOk,
    handleVideoOk,
    handleIframeOk,
    getStatusInfo,
    setupDarkModeTracking,
  };
};
