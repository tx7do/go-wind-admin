<template>
  <div
    class="flex items-center justify-between px-3 py-1 text-xs"
    :class="
      isDark
        ? 'text-[#94a3b8] bg-[#1e293b] border-t border-[#334155]'
        : 'text-[#64748b] bg-[#f8fafc] border-t border-[#e5e7eb]'
    "
  >
    <div class="flex gap-2 items-center">
      <span>{{ statusInfo.words }} {{ $t("common.editor.words") }}</span>
      <span :class="isDark ? 'text-[#475569]' : 'text-[#cbd5e1]'">|</span>
      <span>{{ statusInfo.chars }} {{ $t("common.editor.chars") }}</span>
      <span :class="isDark ? 'text-[#475569]' : 'text-[#cbd5e1]'">|</span>
      <span>Ln {{ statusInfo.cursor }}</span>
    </div>
    <div>
      <span
        class="px-2 py-0.5 text-[11px] rounded"
        :class="isDark ? 'text-[#94a3b8] bg-[#334155]' : 'text-[#475569] bg-[#e2e8f0]'"
      >
        {{ isDark ? `🌙 ${$t("preferences.theme.dark")}` : `☀️ ${$t("preferences.theme.light")}` }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  isDark: boolean;
  statusInfo: { chars: number; words: number; cursor: string };
}>();
</script>
