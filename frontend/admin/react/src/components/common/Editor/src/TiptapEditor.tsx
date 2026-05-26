import Color from '@tiptap/extension-color';
import Highlight from '@tiptap/extension-highlight';
import HorizontalRule from '@tiptap/extension-horizontal-rule';
import Image from '@tiptap/extension-image';
import Link from '@tiptap/extension-link';
import Placeholder from '@tiptap/extension-placeholder';
import Subscript from '@tiptap/extension-subscript';
import Superscript from '@tiptap/extension-superscript';
import { Table } from '@tiptap/extension-table';
import TableCell from '@tiptap/extension-table-cell';
import TableHeader from '@tiptap/extension-table-header';
import TableRow from '@tiptap/extension-table-row';
import TaskItem from '@tiptap/extension-task-item';
import TaskList from '@tiptap/extension-task-list';
import TextAlign from '@tiptap/extension-text-align';
import { TextStyle } from '@tiptap/extension-text-style';
import Underline from '@tiptap/extension-underline';
import StarterKit from '@tiptap/starter-kit';
import { EditorContent, useEditor } from '@tiptap/react';
import { all, createLowlight } from 'lowlight';
import { useCallback, useEffect, useMemo } from 'react';

import {
  CustomCodeBlockLowlight,
  CustomVideo,
  CustomIframe,
  LinkModal,
  CodeBlockModal,
  VideoModal,
  IframeModal,
  StatusBar,
  Toolbar,
  useTiptapEditor,
} from './TiptapEditor/index.ts';

import './tiptap-editor.css';

export interface TiptapEditorProps {
  value: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  config?: Record<string, any>;
  showToolbar?: boolean;
  showStatusBar?: boolean;
  uploadImage?: (file: File) => Promise<string>;
  fullHeight?: boolean;
  onChange?: (value: string) => void;
  onReady?: (editor: any) => void;
}

const lowlight = createLowlight(all);

const TiptapEditor: React.FC<TiptapEditorProps> = ({
  value,
  height = 500,
  disabled = false,
  placeholder,
  showToolbar = true,
  showStatusBar = true,
  uploadImage,
  onChange,
  onReady,
}) => {
  const {
    isDark,
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
    isFullscreen,
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
  } = useTiptapEditor({ value, disabled, placeholder, uploadImage, onChange, onReady });

  const editor = useEditor({
    ...editorConfig,
    extensions: [
      StarterKit.configure({
        link: false,
        underline: false,
        horizontalRule: false,
        codeBlock: false,
      }),
      Underline,
      Subscript,
      Superscript,
      TaskList,
      TaskItem.configure({ nested: true }),
      HorizontalRule,
      Highlight.configure({ multicolor: true }),
      Color,
      TextStyle,
      CustomCodeBlockLowlight.configure({ lowlight, defaultLanguage: 'javascript' }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      Placeholder.configure({ placeholder: placeholder || '' }),
      Link.configure({ openOnClick: false, autolink: true }),
      Image.configure({ inline: true }),
      CustomVideo,
      CustomIframe,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
  });

  // Sync external value
  useEffect(() => {
    if (editor) {
      syncValue(editor, value);
    }
  }, [value, editor, syncValue]);

  // Disabled changes
  useEffect(() => {
    editor?.setEditable(!disabled);
  }, [disabled, editor]);

  // Dark mode tracking
  useEffect(() => {
    return setupDarkModeTracking();
  }, [setupDarkModeTracking]);

  // Prevent body scroll when fullscreen
  useEffect(() => {
    if (isFullscreen) {
      document.body.classList.add('tiptap-editor-fullscreen-active');
    } else {
      document.body.classList.remove('tiptap-editor-fullscreen-active');
    }
    return () => {
      document.body.classList.remove('tiptap-editor-fullscreen-active');
    };
  }, [isFullscreen]);

  // Toolbar actions
  const isActive = useCallback(
    (name: string, options?: any) => {
      return editor?.isActive(name, options) || false;
    },
    [editor],
  );

  const toolbarActions = useMemo(
    () => createToolbarActions(editor),
    [editor, createToolbarActions],
  );

  // Status info
  const statusInfo = useMemo(() => getStatusInfo(editor), [editor, getStatusInfo]);

  const containerHeight = typeof height === 'number' ? `${height}px` : height;

  return (
    <div
      className={`tiptap-editor-wrapper${isDark ? ' tiptap-editor-dark' : ''}${isFullscreen ? ' tiptap-editor-fullscreen' : ''}`}
      data-theme={isDark ? 'dark' : 'light'}
      style={{ height: containerHeight }}
    >
      {/* Toolbar */}
      {showToolbar && <Toolbar editor={editor} isActive={isActive} actions={toolbarActions} isFullscreen={isFullscreen} />}

      {/* Editor Content */}
      <div className="tiptap-editor-content-wrapper">
        <EditorContent
          editor={editor}
          className={`tiptap-editor-content${isDark ? ' dark' : ''}`}
        />
      </div>

      {/* Status Bar */}
      {showStatusBar && (
        <StatusBar
          words={statusInfo.words}
          chars={statusInfo.chars}
          cursor={statusInfo.cursor}
          isDark={isDark}
        />
      )}

      {/* Hidden file inputs */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={(e) => handleImageUpload(e, editor)}
      />
      <input
        ref={markdownInputRef}
        type="file"
        accept=".md,text/markdown"
        style={{ display: 'none' }}
        onChange={(e) => handleMarkdownImport(e, editor)}
      />

      {/* Modals */}
      <LinkModal
        open={linkModalVisible}
        url={linkUrl}
        onOk={() => handleLinkOk(editor)}
        onCancel={() => {
          setLinkModalVisible(false);
          setLinkUrl('');
          editor?.chain().focus().run();
        }}
        onUrlChange={setLinkUrl}
      />

      <CodeBlockModal
        open={codeBlockModalVisible}
        language={codeBlockLanguage}
        content={codeBlockContent}
        onOk={() => handleCodeBlockOk(editor)}
        onCancel={() => {
          setCodeBlockModalVisible(false);
          setCodeBlockContent('');
        }}
        onLanguageChange={setCodeBlockLanguage}
        onContentChange={setCodeBlockContent}
      />

      <VideoModal
        open={videoModalVisible}
        url={videoUrl}
        width={videoWidth}
        onOk={() => handleVideoOk(editor)}
        onCancel={() => setVideoModalVisible(false)}
        onUrlChange={setVideoUrl}
        onWidthChange={setVideoWidth}
      />

      <IframeModal
        open={iframeModalVisible}
        url={iframeUrl}
        width={iframeWidth}
        height={iframeHeight}
        title={iframeTitle}
        allowFullscreen={iframeAllowFullscreen}
        onOk={() => handleIframeOk(editor)}
        onCancel={() => setIframeModalVisible(false)}
        onUrlChange={setIframeUrl}
        onWidthChange={setIframeWidth}
        onHeightChange={setIframeHeight}
        onTitleChange={setIframeTitle}
        onAllowFullscreenChange={setIframeAllowFullscreen}
      />
    </div>
  );
};

export default TiptapEditor;
