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

    <!-- Status Bar -->
    <TiptapStatusBar
      v-if="showStatusBar"
      :is-dark="isDark"
      :status-info="statusInfo"
    />

    <!-- Link Input Dialog -->
    <ElDialog
      v-model="linkModalVisible"
      :title="$t('common.title.insert_url')"
      width="400px"
      :close-on-click-modal="false"
      @close="handleLinkCancel"
    >
      <ElInput
        v-model="linkUrl"
        :placeholder="$t('common.placeholder.input_url')"
        clearable
        @keyup.enter="handleLinkOk"
      />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleLinkCancel">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" @click="handleLinkOk">{{ $t('common.confirm') }}</el-button>
        </div>
      </template>
    </ElDialog>

    <!-- Code Block Insert Dialog -->
    <ElDialog
      v-model="codeBlockModalVisible"
      :title="$t('common.editor.insertCodeBlock')"
      width="600px"
      :close-on-click-modal="false"
      @close="handleCodeBlockCancel"
    >
      <div class="code-block-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.codeLanguage') }}
          </label>
          <ElSelect
            v-model="codeBlockLanguage"
            :placeholder="$t('common.editor.selectLanguage')"
            filterable
            class="language-select"
          >
            <ElOption
              v-for="lang in languages"
              :key="lang.value"
              :label="lang.label"
              :value="lang.value"
            />
          </ElSelect>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.codeContent') }}
          </label>
          <textarea
            v-model="codeBlockContent"
            :placeholder="$t('common.editor.codeContentPlaceholder')"
            class="code-textarea"
            rows="10"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleCodeBlockCancel">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" @click="handleCodeBlockOk">{{ $t('common.confirm') }}</el-button>
        </div>
      </template>
    </ElDialog>

    <!-- Video Insert Dialog -->
    <ElDialog
      v-model="videoModalVisible"
      :title="$t('common.editor.insertVideo')"
      width="500px"
      :close-on-click-modal="false"
      @close="handleVideoCancel"
    >
      <div class="video-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.videoUrl') }}
          </label>
          <ElInput
            v-model="videoUrl"
            :placeholder="$t('common.editor.videoUrlPlaceholder')"
            clearable
            @keyup.enter="handleVideoOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.videoWidth') }}
          </label>
          <ElSelect v-model="videoWidth" class="width-select">
            <ElOption value="100%" label="100%" />
            <ElOption value="75%" label="75%" />
            <ElOption value="50%" label="50%" />
            <ElOption value="640px" label="640px" />
            <ElOption value="800px" label="800px" />
          </ElSelect>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleVideoCancel">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" @click="handleVideoOk">{{ $t('common.confirm') }}</el-button>
        </div>
      </template>
    </ElDialog>

    <!-- Iframe Insert Dialog -->
    <ElDialog
      v-model="iframeModalVisible"
      :title="$t('common.editor.insertIframe')"
      width="500px"
      :close-on-click-modal="false"
      @close="handleIframeCancel"
    >
      <div class="iframe-modal">
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.iframeUrl') }}
          </label>
          <ElInput
            v-model="iframeUrl"
            :placeholder="$t('common.editor.iframeUrlPlaceholder')"
            clearable
            @keyup.enter="handleIframeOk"
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.iframeWidth') }}
          </label>
          <ElSelect v-model="iframeWidth" class="width-select">
            <ElOption value="100%" label="100%" />
            <ElOption value="75%" label="75%" />
            <ElOption value="50%" label="50%" />
            <ElOption value="640px" label="640px" />
            <ElOption value="800px" label="800px" />
          </ElSelect>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.iframeHeight') }}
          </label>
          <ElSelect v-model="iframeHeight" class="height-select">
            <ElOption value="500px" label="500px" />
            <ElOption value="300px" label="300px" />
            <ElOption value="100%" label="100%" />
          </ElSelect>
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.iframeTitle') }}
          </label>
          <ElInput
            v-model="iframeTitle"
            :placeholder="$t('common.editor.iframeTitlePlaceholder')"
            clearable
          />
        </div>
        <div class="modal-field">
          <label class="field-label">
            {{ $t('common.editor.allowFullscreen') }}
          </label>
          <div style="display: flex; gap: 12px; align-items: center">
            <ElSwitch v-model="iframeAllowFullscreen" />
            <span>{{
              iframeAllowFullscreen
                ? $t('common.editor.allowFullscreenEnabled')
                : $t('common.editor.allowFullscreenDisabled')
            }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleIframeCancel">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" @click="handleIframeOk">{{ $t('common.confirm') }}</el-button>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, toRef } from "vue";

import { $t } from "@/core/i18n";

import { EditorContent } from "@tiptap/vue-3";
import { marked } from "marked";
import { ElDialog, ElInput, ElSelect, ElOption, ElSwitch } from "element-plus";

import { languages } from "./constants";
import { useEditorModals } from "./composables/useEditorModals";
import { useToolbarActions } from "./composables/useToolbarActions";
import { useTiptapEditor } from "./composables/useTiptapEditor";
import TiptapStatusBar from "./TiptapStatusBar.vue";
import TiptapToolbar from "./TiptapToolbar.vue";

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
  placeholder: $t("common.editor.please_input_content"),
  config: () => ({}),
  showToolbar: true,
  showStatusBar: true,
  uploadImage: undefined,
  fullHeight: true,
});

const emit = defineEmits<{
  (e: "change", value: string): void;
  (e: "imageUpload", file: File): void;
  (e: "ready", editor: any): void;
  (e: "update:modelValue", value: string): void;
}>();

// ---- 编辑器核心 ----
const { editor, isDark, textColor, highlightColor } = useTiptapEditor({
  modelValue: toRef(props, "modelValue"),
  disabled: toRef(props, "disabled"),
  placeholder: toRef(props, "placeholder"),
  showToolbar: toRef(props, "showToolbar"),
  showStatusBar: toRef(props, "showStatusBar"),
  uploadImage: toRef(props, "uploadImage"),
  fullHeight: toRef(props, "fullHeight"),
  height: toRef(props, "height"),
  config: toRef(props, "config"),
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
  markdownInputRef
);

// ---- 辅助 ----
const isActive = (name: string, options?: any) => {
  if (!editor.value) return false;
  return editor.value.isActive(name, options);
};

// 状态栏计算
const statusInfo = computed(() => {
  if (!editor.value) return { chars: 0, words: 0, cursor: "0:0" };

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
      const nodeText = node.text || "";
      for (const element of nodeText) {
        if (pos === from) {
          return false;
        }
        if (element === "\n") {
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
    emit("imageUpload", file!);
    return;
  }

  try {
    const url = await props.uploadImage(file);
    if (url && editor.value) {
      editor.value.chain().focus().setImage({ src: url }).run();
    }
  } catch (error) {
    console.error("Image upload failed:", error);
  } finally {
    input.value = "";
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
    console.error("Markdown import failed:", error);
  } finally {
    input.value = "";
  }
};

// 销毁清理
onUnmounted(() => {
  editor.value?.destroy();
});
</script>

<style scoped src="./tiptap.scss"></style>
