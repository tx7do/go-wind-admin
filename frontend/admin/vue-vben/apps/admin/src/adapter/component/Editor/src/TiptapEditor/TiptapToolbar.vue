<script setup lang="ts">
import type { Editor } from '@tiptap/vue-3';

defineProps<{
  actions: Record<string, any>;
  editor: Editor | undefined;
  highlightColor: string;
  isActive: (name: string, options?: any) => boolean;
  isDark: boolean;
  openLinkModal: () => void;
  textColor: string;
}>();

const emit = defineEmits<{
  (e: 'image-upload', event: Event): void;
  (e: 'markdown-import', event: Event): void;
}>();

const fileInputRef = defineModel<File | undefined>('fileInputRef');
const markdownInputRef = defineModel<HTMLInputElement | undefined>(
  'markdownInputRef',
);
</script>

<template>
  <!-- prettier-ignore -->
  <div
    class="flex flex-wrap gap-1 items-center px-3 py-2 select-none bg-[var(--tte-toolbar-bg)] border-b border-[var(--tte-toolbar-border)]"
  >
    <div class="flex gap-0.5 items-center">
      <button
        type="button"
        class="tb"
        :class="{ 'tb-active': isActive('heading', { level: 1 }) }"
        :title="$t('common.editor.h1')"
        @click="actions.toggleHeading(1)"
      >
H1
</button>
      <button
        type="button"
        class="tb"
        :class="{ 'tb-active': isActive('heading', { level: 2 }) }"
        :title="$t('common.editor.h2')"
        @click="actions.toggleHeading(2)"
      >
H2
</button>
      <button
        type="button"
        class="tb"
        :class="{ 'tb-active': isActive('heading', { level: 3 }) }"
        :title="$t('common.editor.h3')"
        @click="actions.toggleHeading(3)"
      >
H3
</button>
      <button
        type="button"
        class="tb"
        :class="{ 'tb-active': isActive('paragraph') }"
        :title="$t('common.editor.paragraph')"
        @click="actions.setParagraph"
      >
P
</button>
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>

    <div class="flex gap-0.5 items-center">
      <button type="button" class="tb" :class="{ 'tb-active': isActive('bold') }" :title="$t('common.editor.bold')" @click="actions.toggleBold">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 4h8a4 4 0 014 4 4 4 0 01-4 4H6z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12h9a4 4 0 014 4 4 4 0 01-4 4H6z" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('italic') }" :title="$t('common.editor.italic')" @click="actions.toggleItalic">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 4h4m-2 0v16m-4 0h8" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('strike') }" :title="$t('common.editor.strike')" @click="actions.toggleStrike">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M6 6a6 6 0 0112 0M6 18a6 6 0 0012 0" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('underline') }" :title="$t('common.editor.underline')" @click="actions.toggleUnderline">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 3v8a6 6 0 006 6h0a6 6 0 006-6V3M3 21h18" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('code') }" :title="$t('common.editor.code')" @click="actions.toggleCode">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /></svg>
      </button>
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>
    
    <div class="flex gap-0.5 items-center">
      <button type="button" class="tb" :title="$t('common.editor.subscript')" @click="actions.toggleSubscript">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><text x="3" y="12" font-size="16" font-weight="bold">X</text><text x="14" y="18" font-size="10">2</text></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('superscript') }" :title="$t('common.editor.superscript')" @click="actions.toggleSuperscript">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><text x="3" y="16" font-size="16" font-weight="bold">X</text><text x="14" y="8" font-size="10">2</text></svg>
      </button>
      <input
        :value="textColor"
        type="color"
        class="w-7 h-7 p-0.5 cursor-pointer rounded transition-all duration-150 border border-[var(--tte-border-secondary)] hover:border-[var(--tte-text-secondary)]"
        :title="$t('common.editor.textColor')"
        @change="actions.setTextColor(($event.target as HTMLInputElement).value)"
      />
      <input
        :value="highlightColor"
        type="color"
        class="w-7 h-7 p-0.5 cursor-pointer rounded transition-all duration-150 border border-[var(--tte-border-secondary)] hover:border-[var(--tte-text-secondary)]"
        :title="$t('common.editor.highlightColor')"
        @change="actions.setHighlight(($event.target as HTMLInputElement).value)"
      />
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>
    
    <div class="flex gap-0.5 items-center">
      <button type="button" class="tb" :class="{ 'tb-active': isActive('bulletList') }" :title="$t('common.editor.bulletList')" @click="actions.toggleBulletList">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /><circle cx="3" cy="6" r="1" fill="currentColor" /><circle cx="3" cy="12" r="1" fill="currentColor" /><circle cx="3" cy="18" r="1" fill="currentColor" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('orderedList') }" :title="$t('common.editor.orderedList')" @click="actions.toggleOrderedList">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /><text x="2" y="8" font-size="8" fill="currentColor">1.</text><text x="2" y="14" font-size="8" fill="currentColor">2.</text><text x="2" y="20" font-size="8" fill="currentColor">3.</text></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('taskList') }" :title="$t('common.editor.taskList')" @click="actions.toggleTaskList">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="4" width="4" height="4" rx="1" /><path d="M10 6h10" stroke-width="2" /><rect x="3" y="12" width="4" height="4" rx="1" /><path d="M10 14h10" stroke-width="2" /><path d="M4 6l1 1 2-2" stroke-width="2" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('blockquote') }" :title="$t('common.editor.blockquote')" @click="actions.toggleBlockquote">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h8M8 14h6M4 6h16v12H4z" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('codeBlock') }" :title="$t('common.editor.insertCodeBlock')" @click="actions.insertCodeBlock">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" /><rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" fill="none" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.insertTable')" @click="actions.insertTable">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="3" width="18" height="18" rx="2" stroke-width="2" /><line x1="12" y1="3" x2="12" y2="21" stroke-width="2" /><line x1="3" y1="12" x2="21" y2="12" stroke-width="2" /></svg>
      </button>
      <button v-if="isActive('table')" type="button" class="tb text-red-500" :title="$t('common.editor.deleteTable')" @click="actions.deleteTable">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.insertHorizontalRule')" @click="actions.insertHorizontalRule">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><line x1="4" y1="12" x2="20" y2="12" stroke-width="2" /></svg>
      </button>
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>

    <div class="flex gap-0.5 items-center">
      <button type="button" class="tb" :class="{ 'tb-active': isActive('textAlign', { textAlign: 'left' }) }" :title="$t('common.editor.left')" @click="actions.setAlign('left')">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h10M4 18h14" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('textAlign', { textAlign: 'center' }) }" :title="$t('common.editor.center')" @click="actions.setAlign('center')">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M7 12h10M5 18h14" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('textAlign', { textAlign: 'right' }) }" :title="$t('common.editor.right')" @click="actions.setAlign('right')">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M10 12h10M6 18h14" /></svg>
      </button>
      <button type="button" class="tb" :class="{ 'tb-active': isActive('textAlign', { textAlign: 'justify' }) }" :title="$t('common.editor.justify')" @click="actions.setAlign('justify')">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /></svg>
      </button>
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>

    <div v-if="isActive('table')" class="flex gap-0.5 items-center">
      <button type="button" class="tb" :title="$t('common.editor.addRowBefore')" @click="actions.addRowBefore">R+</button>
      <button type="button" class="tb" :title="$t('common.editor.addRowAfter')" @click="actions.addRowAfter">+R</button>
      <button type="button" class="tb" :title="$t('common.editor.deleteRow')" @click="actions.deleteRow">R-</button>
      <button type="button" class="tb" :title="$t('common.editor.addColumnBefore')" @click="actions.addColumnBefore">C+</button>
      <button type="button" class="tb" :title="$t('common.editor.addColumnAfter')" @click="actions.addColumnAfter">+C</button>
      <button type="button" class="tb" :title="$t('common.editor.deleteColumn')" @click="actions.deleteColumn">C-</button>
      <button type="button" class="tb" :title="$t('common.editor.mergeCells')" @click="actions.mergeCells">M</button>
      <button type="button" class="tb" :title="$t('common.editor.splitCell')" @click="actions.splitCell">S</button>
      <button type="button" class="tb" :title="$t('common.editor.toggleHeaderRow')" @click="actions.toggleHeaderRow">HR</button>
      <button type="button" class="tb" :title="$t('common.editor.toggleHeaderColumn')" @click="actions.toggleHeaderColumn">HC</button>
      <button type="button" class="tb" :title="$t('common.editor.toggleHeaderCell')" @click="actions.toggleHeaderCell">H</button>
    </div>

    <div v-if="isActive('table')" class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>

    <div class="flex gap-0.5 items-center">
      <button type="button" class="tb" :disabled="!editor?.can().undo()" :title="$t('common.editor.undo')" @click="actions.undo">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a5 5 0 015 5v2M3 10l3-3m-3 3l3 3" /></svg>
      </button>
      <button type="button" class="tb" :disabled="!editor?.can().redo()" :title="$t('common.editor.redo')" @click="actions.redo">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10H11a5 5 0 00-5 5v2M21 10l-3-3m3 3l-3 3" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.clearFormatting')" @click="actions.clearFormatting">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M8 6l8 12M6 18h12" /></svg>
      </button>
      <button type="button" class="tb text-red-500" :title="$t('common.editor.clearContent')" @click="actions.clearContent">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
      </button>
    </div>

    <div class="w-px h-4 mx-1.5 bg-[var(--tte-toolbar-border)]"></div>

    <div class="flex gap-0.5 items-center">
      <button
        type="button"
        class="tb"
        :class="{ 'tb-active': isActive('link') }"
        :title="isActive('link') ? $t('common.editor.removeUrl') : $t('common.editor.insertUrl')"
        @click="isActive('link') ? editor?.chain().focus().unsetLink().run() : openLinkModal()"
      >
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.uploadImage')" @click="actions.uploadImage">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.insertVideo')" @click="actions.insertVideo">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.insertIframe')" @click="actions.insertIframe">
        <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor"><rect x="3" y="3" width="18" height="18" rx="2" stroke-width="2" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h8M12 8v8" /></svg>
      </button>
      <button type="button" class="tb" :title="$t('common.editor.importMarkdown')" @click="actions.importMarkdown">MD</button>
      <input ref="fileInputRef" type="file" accept="image/*" class="hidden" @change="emit('image-upload', $event)" />
      <input ref="markdownInputRef" type="file" accept=".md,text/markdown" class="hidden" @change="emit('markdown-import', $event)" />
    </div>
  </div>
</template>

<style scoped>
/* ---- 工具栏按钮（颜色由 --tte-* CSS 变量统一管理，亮暗模式自动切换） ---- */
.tb {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--tte-toolbar-btn);
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 4px;
  transition: all 0.15s ease;
}

.tb:hover:not(:disabled) {
  color: var(--tte-toolbar-btn-hover-color) !important;
  background-color: var(--tte-toolbar-btn-hover-bg) !important;
}

.tb-active {
  color: var(--tte-toolbar-btn-active) !important;
  background-color: var(--tte-toolbar-btn-active-bg) !important;
}

.tb:disabled {
  color: var(--tte-toolbar-btn-disabled) !important;
  cursor: not-allowed;
  opacity: 0.4;
}

/* ---- 图标尺寸 ---- */
.tb .icon {
  width: 16px;
  height: 16px;
}
</style>
