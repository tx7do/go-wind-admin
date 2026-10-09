import type { Editor } from '@tiptap/vue-3';

import { nextTick, type Ref, ref, watch } from 'vue';

/**
 * 管理 Tiptap 编辑器中所有 Modal 的状态与操作
 * - Link: 插入/取消链接
 * - CodeBlock: 插入代码块
 * - Video: 插入视频
 * - Iframe: 插入 iframe
 */
export function useEditorModals(editor: Ref<Editor | undefined>) {
  // ---- Link Modal ----
  const linkModalVisible = ref(false);
  const linkUrl = ref('');

  const openLinkModal = () => {
    linkUrl.value = '';
    linkModalVisible.value = true;
  };

  const handleLinkOk = () => {
    const url = linkUrl.value.trim();
    if (url && editor.value) {
      editor.value
        .chain()
        .focus()
        .extendMarkRange('link')
        .setLink({ href: url })
        .run();
    }
    linkModalVisible.value = false;
    linkUrl.value = '';
  };

  const handleLinkCancel = () => {
    linkModalVisible.value = false;
    linkUrl.value = '';
    editor.value?.chain().focus().run();
  };

  // ---- CodeBlock Modal ----
  const codeBlockModalVisible = ref(false);
  const codeBlockLanguage = ref('javascript');
  const codeBlockContent = ref('');

  const openCodeBlockModal = () => {
    codeBlockLanguage.value = 'javascript';
    codeBlockContent.value = '';
    codeBlockModalVisible.value = true;
  };

  const handleCodeBlockOk = () => {
    const code = codeBlockContent.value.trim();
    if (code && editor.value) {
      editor.value
        .chain()
        .focus()
        .insertContent({
          type: 'codeBlock',
          attrs: { language: codeBlockLanguage.value },
          content: [{ type: 'text', text: code }],
        })
        .run();
    }
    codeBlockModalVisible.value = false;
    codeBlockContent.value = '';
    codeBlockLanguage.value = 'javascript';
  };

  const handleCodeBlockCancel = () => {
    codeBlockModalVisible.value = false;
    codeBlockContent.value = '';
    codeBlockLanguage.value = 'javascript';
    editor.value?.chain().focus().run();
  };

  // ---- Video Modal ----
  const videoModalVisible = ref(false);
  const videoUrl = ref('');
  const videoWidth = ref('100%');

  const openVideoModal = () => {
    videoUrl.value = '';
    videoWidth.value = '100%';
    videoModalVisible.value = true;
  };

  const handleVideoOk = () => {
    const url = videoUrl.value.trim();
    if (url && editor.value) {
      (editor.value.chain().focus() as any)
        .setVideo({ src: url, width: videoWidth.value })
        .run();
    }
    videoModalVisible.value = false;
    videoUrl.value = '';
    videoWidth.value = '100%';
  };

  const handleVideoCancel = () => {
    videoModalVisible.value = false;
    videoUrl.value = '';
    videoWidth.value = '100%';
  };

  // ---- Iframe Modal ----
  const iframeModalVisible = ref(false);
  const iframeUrl = ref('');
  const iframeWidth = ref('100%');
  const iframeHeight = ref('500px');
  const iframeTitle = ref('');
  const iframeAllowFullscreen = ref(true);

  const openIframeModal = () => {
    iframeUrl.value = '';
    iframeWidth.value = '100%';
    iframeHeight.value = '500px';
    iframeTitle.value = '';
    iframeAllowFullscreen.value = true;
    iframeModalVisible.value = true;
  };

  const handleIframeOk = () => {
    const url = iframeUrl.value.trim();
    if (url && editor.value) {
      (editor.value.chain().focus() as any)
        .setIframe({
          src: url,
          width: iframeWidth.value,
          height: iframeHeight.value,
          title: iframeTitle.value || undefined,
          allowfullscreen: iframeAllowFullscreen.value,
        })
        .run();
    }
    iframeModalVisible.value = false;
    iframeUrl.value = '';
    iframeWidth.value = '100%';
    iframeHeight.value = '500px';
    iframeTitle.value = '';
    iframeAllowFullscreen.value = true;
  };

  const handleIframeCancel = () => {
    iframeModalVisible.value = false;
    iframeUrl.value = '';
    iframeWidth.value = '100%';
    iframeHeight.value = '500px';
    iframeTitle.value = '';
    iframeAllowFullscreen.value = true;
    editor.value?.chain().focus().run();
  };

  // ---- Link Modal 自动聚焦 ----
  watch(linkModalVisible, (visible) => {
    if (visible) {
      nextTick(() => {
        const input = document.querySelector(
          '.ant-modal:last-of-type .ant-input',
        ) as HTMLInputElement;
        input?.focus();
      });
    }
  });

  return {
    // Link
    linkModalVisible,
    linkUrl,
    openLinkModal,
    handleLinkOk,
    handleLinkCancel,
    // CodeBlock
    codeBlockModalVisible,
    codeBlockLanguage,
    codeBlockContent,
    openCodeBlockModal,
    handleCodeBlockOk,
    handleCodeBlockCancel,
    // Video
    videoModalVisible,
    videoUrl,
    videoWidth,
    openVideoModal,
    handleVideoOk,
    handleVideoCancel,
    // Iframe
    iframeModalVisible,
    iframeUrl,
    iframeWidth,
    iframeHeight,
    iframeTitle,
    iframeAllowFullscreen,
    openIframeModal,
    handleIframeOk,
    handleIframeCancel,
  };
}
