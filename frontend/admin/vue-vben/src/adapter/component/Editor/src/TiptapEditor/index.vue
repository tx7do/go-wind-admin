<script setup lang="ts">
import { computed, onUnmounted, ref, toRef } from 'vue';

import { EditorContent } from '@tiptap/vue-3';
import { Button, Input, Modal, Select, Switch } from 'ant-design-vue';
import { marked } from 'marked';

import { $t } from '#/locales';

import { useEditorModals } from './composables/useEditorModals';
import { useTiptapEditor } from './composables/useTiptapEditor';
import { useToolbarActions } from './composables/useToolbarActions';
import { languages } from './constants';
import TiptapStatusBar from './TiptapStatusBar.vue';
import TiptapToolbar from './TiptapToolbar.vue';

interface Props {
  modelValue: string;
  height?: number | string;
  disabled?: boolean;
  placeholder?: string;
  config?: Record<string, any>;
  showToolbar?: boolean;
  showStatusBar?: boolean;
  uploadImage?: (file: File) => Promise<string>;
  fullHeight?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  height: 500,
  disabled: false,
  placeholder: $t('ui.editor.please_input_content'),
  config: () => ({}),
  showToolbar: true,
  showStatusBar: true,
  uploadImage: undefined,
  fullHeight: true,
});

const emit = defineEmits<{
  (e: 'change', value: string): void;
  (e: 'imageUpload', file: File): void;
  (e: 'ready', editor: any): void;
  (e: 'update:modelValue', value: string): void;
}>();

// ---- 编辑器核心 ----
const { editor, isDark, textColor, highlightColor } = useTiptapEditor({
  modelValue: toRef(props, 'modelValue'),
  disabled: toRef(props, 'disabled'),
  placeholder: toRef(props, 'placeholder'),
  showToolbar: toRef(props, 'showToolbar'),
  showStatusBar: toRef(props, 'showStatusBar'),
  uploadImage: toRef(props, 'uploadImage'),
  fullHeight: toRef(props, 'fullHeight'),
  height: toRef(props, 'height'),
  config: toRef(props, 'config'),
  emit,
});

// ---- Modal 管理 ----
const {
  linkModalVisible,
  linkUrl,
  openLinkModal,
  handleLinkOk,
  handleLinkCancel,
  codeBlockModalVisible,
  codeBlockLanguage,
  codeBlockContent,
  openCodeBlockModal,
  handleCodeBlockOk,
  handleCodeBlockCancel,
  videoModalVisible,
  videoUrl,
  videoWidth,
  openVideoModal,
  handleVideoOk,
  handleVideoCancel,
  iframeModalVisible,
  iframeUrl,
  iframeWidth,
  iframeHeight,
  iframeTitle,
  iframeAllowFullscreen,
  openIframeModal,
  handleIframeOk,
  handleIframeCancel,
} = useEditorModals(editor);

// ---- 工具栏操作 ----
const fileInputRef = ref<HTMLInputElement>();
const markdownInputRef = ref<HTMLInputElement>();

const toolbarActions = useToolbarActions(
  editor,
  { openLinkModal, openCodeBlockModal, openVideoModal, openIframeModal },
  fileInputRef,
  markdownInputRef,
);

// ---- 辅助 ----
const isActive = (name: string, options?: any) => {
  if (!editor.value) return false;
  return editor.value.isActive(name, options);
};

// 状态栏计算
const statusInfo = computed(() => {
  if (!editor.value) return { chars: 0, words: 0, cursor: '0:0' };

  const text = editor.value.getText();
  const chars = text.length;
  const words = text.trim() ? text.trim().split(/\s+/).length : 0;

  const { from } = editor.value.state.selection;
  const doc = editor.value.state.doc;
  let col = 1;
  let line = 1;
  let pos = 1;
  doc.descendants((node) => {
    if (node.isText) {
      const nodeText = node.text || '';
      for (const element of nodeText) {
        if (pos === from) {
          return false;
        }
        if (element === '\n') {
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
});

// 图片上传处理
const handleImageUpload = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file || !props.uploadImage) {
    emit('imageUpload', file!);
    return;
  }

  try {
    const url = await props.uploadImage(file);
    if (url && editor.value) {
      editor.value.chain().focus().setImage({ src: url }).run();
    }
  } catch (error) {
    console.error('Image upload failed:', error);
  } finally {
    input.value = '';
  }
};

// Markdown 导入处理
const handleMarkdownImport = async (event: Event) => {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }

  try {
    const markdown = await file.text();
    const html = marked.parse(markdown);
    if (editor.value) {
      editor.value.commands.setContent(String(html));
    }
  } catch (error) {
    console.error('Markdown import failed:', error);
  } finally {
    input.value = '';
  }
};

// 销毁清理
onUnmounted(() => {
  editor.value?.destroy();
});
</script>

<template>
  <!-- prettier-ignore -->
  <div
    class="tiptap-editor-wrapper"
    :class="{ 'tiptap-editor-dark': isDark }"
    :data-theme="isDark ? 'dark' : 'light'"
    :style="{ height: typeof props.height === 'number' ? `${props.height}px` : props.height }"
  >
    <!-- Toolbar -->
    <TiptapToolbar
      v-if="showToolbar"
      :editor="editor"
      :is-dark="isDark"
      :is-active="isActive"
      :actions="toolbarActions"
      :text-color="textColor"
      :highlight-color="highlightColor"
      :open-link-modal="openLinkModal"
      @image-upload="handleImageUpload"
      @markdown-import="handleMarkdownImport"
    />

    <!-- Editor Content -->
    <div class="tiptap-editor-content-wrapper">
      <EditorContent
        :editor="editor"
        class="tiptap-editor-content"
        :class="{ dark: isDark }"
      />
    </div>

    <!-- Status Bar (绝对定位) -->
    <TiptapStatusBar
      v-if="showStatusBar"
      :is-dark="isDark"
      :status-info="statusInfo"
    />

    <!-- Link Input Dialog -->
    <Modal
      v-model:open="linkModalVisible"
      :title="$t('ui.title.insert_url')"
      :mask-closable="false"
      @cancel="handleLinkCancel"
    >
      <Input
        v-model:value="linkUrl"
        :placeholder="$t('ui.placeholder.input_url')"
        allow-clear
        @keyup.enter="handleLinkOk"
      />
      <template #footer>
        <Button @click="handleLinkCancel">{{ $t('common.cancel') }}</Button>
        <Button type="primary" @click="handleLinkOk">{{ $t('common.confirm') }}</Button>
      </template>
    </Modal>

    <!-- Code Block Insert Dialog -->
    <Modal
      v-model:open="codeBlockModalVisible"
      :title="$t('ui.editor.insertCodeBlock')"
      width="600px"
      :mask-closable="false"
      @cancel="handleCodeBlockCancel"
    >
      <div class="code-block-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.codeLanguage') }}
          </label>
          <Select
            v-model:value="codeBlockLanguage"
            :placeholder="$t('ui.editor.selectLanguage')"
            show-search
            :filter-option="true"
            class="language-select"
          >
            <Select.Option
              v-for="lang in languages"
              :key="lang.value"
              :value="lang.value"
            >
              {{ lang.label }}
            </Select.Option>
          </Select>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.codeContent') }}
          </label>
          <textarea
            v-model="codeBlockContent"
            :placeholder="$t('ui.editor.codeContentPlaceholder')"
            class="code-textarea"
            rows="10"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <Button @click="handleCodeBlockCancel">{{ $t('common.cancel') }}</Button>
        <Button type="primary" @click="handleCodeBlockOk">{{ $t('common.confirm') }}</Button>
      </template>
    </Modal>

    <!-- Video Insert Dialog -->
    <Modal
      v-model:open="videoModalVisible"
      :title="$t('ui.editor.insertVideo')"
      width="500px"
      :mask-closable="false"
      @cancel="handleVideoCancel"
    >
      <div class="video-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.videoUrl') }}
          </label>
          <Input
            v-model:value="videoUrl"
            :placeholder="$t('ui.editor.videoUrlPlaceholder')"
            allow-clear
            @keyup.enter="handleVideoOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.videoWidth') }}
          </label>
          <Select v-model:value="videoWidth" class="width-select">
            <Select.Option value="100%">100%</Select.Option>
            <Select.Option value="75%">75%</Select.Option>
            <Select.Option value="50%">50%</Select.Option>
            <Select.Option value="640px">640px</Select.Option>
            <Select.Option value="800px">800px</Select.Option>
          </Select>
        </div>
      </div>
      <template #footer>
        <Button @click="handleVideoCancel">{{ $t('common.cancel') }}</Button>
        <Button type="primary" @click="handleVideoOk">{{ $t('common.confirm') }}</Button>
      </template>
    </Modal>

    <!-- Iframe Insert Dialog -->
    <Modal
      v-model:open="iframeModalVisible"
      :title="$t('ui.editor.insertIframe')"
      width="500px"
      :mask-closable="false"
      @cancel="handleIframeCancel"
    >
      <div class="iframe-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeUrl') }}
          </label>
          <Input
            v-model:value="iframeUrl"
            :placeholder="$t('ui.editor.iframeUrlPlaceholder')"
            allow-clear
            @keyup.enter="handleIframeOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeWidth') }}
          </label>
          <Select v-model:value="iframeWidth" class="width-select">
            <Select.Option value="100%">100%</Select.Option>
            <Select.Option value="75%">75%</Select.Option>
            <Select.Option value="50%">50%</Select.Option>
            <Select.Option value="640px">640px</Select.Option>
            <Select.Option value="800px">800px</Select.Option>
          </Select>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeHeight') }}
          </label>
          <Select v-model:value="iframeHeight" class="height-select">
            <Select.Option value="500px">500px</Select.Option>
            <Select.Option value="300px">300px</Select.Option>
            <Select.Option value="100%">100%</Select.Option>
          </Select>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.iframeTitle') }}
          </label>
          <Input
            v-model:value="iframeTitle"
            :placeholder="$t('ui.editor.iframeTitlePlaceholder')"
            allow-clear
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('ui.editor.allowFullscreen') }}
          </label>
          <div style="display: flex; gap: 12px; align-items: center">
            <Switch v-model:checked="iframeAllowFullscreen" />
            <span>{{
              iframeAllowFullscreen
                ? $t('ui.editor.allowFullscreenEnabled')
                : $t('ui.editor.allowFullscreenDisabled')
            }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <Button @click="handleIframeCancel">{{ $t('common.cancel') }}</Button>
        <Button type="primary" @click="handleIframeOk">{{ $t('common.confirm') }}</Button>
      </template>
    </Modal>
  </div>
</template>

<style scoped src="./tiptap.scss"></style>
