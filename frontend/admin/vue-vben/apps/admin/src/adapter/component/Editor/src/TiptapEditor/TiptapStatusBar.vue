<script setup lang="ts">
defineProps<{
  isDark: boolean;
  statusInfo: { chars: number; cursor: string; words: number };
}>();
</script>

<template>
  <div
    class="pointer-events-none absolute bottom-3 right-4 flex select-none items-center gap-2 text-xs text-[var(--tte-text-muted)]"
  >
    <span>{{ statusInfo.words }} {{ $t('common.editor.words') }}</span>
    <span class="text-[var(--tte-statusbar-border)]">|</span>
    <span>{{ statusInfo.chars }} {{ $t('common.editor.chars') }}</span>
    <span class="text-[var(--tte-statusbar-border)]">|</span>
    <span>Ln {{ statusInfo.cursor }}</span>
    <span
      class="rounded bg-[var(--tte-code-bg)] px-2 py-0.5 text-[11px] text-[var(--tte-text-muted)]"
    >
      MD
    </span>
  </div>
</template>
