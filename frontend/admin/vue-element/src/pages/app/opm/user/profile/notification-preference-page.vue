<template>
  <div class="notification-preference-page">
    <!-- 静音时段 -->
    <h3 class="section-title">{{ $t("pages.user.profile.notification.quietTitle") }}</h3>
    <div class="section-hint">{{ $t("pages.user.profile.notification.quietDesc") }}</div>
    <div class="section-row">
      <el-switch v-model="quietEnabled" />
      <span class="switch-label">{{ $t("pages.user.profile.notification.quietEnable") }}</span>
    </div>
    <div v-if="quietEnabled" class="section-row">
      <span class="field-label">{{ $t("pages.user.profile.notification.quietTimeRange") }}</span>
      <!-- 两个独立单点选择器而非 is-range：EP 的 range 时间选择器对 start>end（跨零点）的
           合法窗口会在挂载时把值改写成 00:00/23:00（实测 EP 2.14），单点选择器无此问题 -->
      <el-time-picker
        v-model="quietStart"
        format="HH:mm"
        :minute-step="5"
        :clearable="false"
        :placeholder="$t('pages.user.profile.notification.startTime')"
        style="width: 120px"
      />
      <span>-</span>
      <el-time-picker
        v-model="quietEnd"
        format="HH:mm"
        :minute-step="5"
        :clearable="false"
        :placeholder="$t('pages.user.profile.notification.endTime')"
        style="width: 120px"
      />
    </div>

    <!-- 分类退订 -->
    <h3 class="section-title mt">{{ $t("pages.user.profile.notification.muteTitle") }}</h3>
    <div class="section-hint">{{ $t("pages.user.profile.notification.muteDesc") }}</div>
    <div v-if="categoryOptions.length > 0" class="section-row checkbox-block">
      <el-checkbox-group v-model="mutedCategoryIds">
        <el-checkbox v-for="item in categoryOptions" :key="item.value" :value="item.value">
          {{ item.label }}
        </el-checkbox>
      </el-checkbox-group>
    </div>
    <div v-else class="section-hint">{{ $t("pages.user.profile.notification.noCategories") }}</div>

    <el-button type="primary" :loading="saving" @click="handleSave">
      {{ $t("pages.user.profile.notification.save") }}
    </el-button>

    <div class="section-hint footer-hint">
      {{ $t("pages.user.profile.notification.transactionalNote") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { ElMessage } from "element-plus";
import {
  useMyNotificationPreference,
  useMyNotifiableCategories,
  useUpdateMyNotificationPreference,
} from "@/api/composables/notification-preference";
import { $t } from "@/core/i18n";

/**
 * 个人中心「通知偏好」（通知域 P3）：
 *  - 静音时段只抑制 SSE 实时推送，消息仍进收件箱；
 *  - 分类退订只约束全员广播，点对点定向发送不受影响。
 * 验证码等事务性出站邮件不经偏好层，不受本页任何配置影响。
 */

const quietEnabled = ref(false);
// 起/止各自独立绑定（el-time-picker 单点）：跨零点窗口（start>end）是合法配置
const quietStart = ref<Date>(minutesToDate(1320));
const quietEnd = ref<Date>(minutesToDate(480));
const mutedCategoryIds = ref<string[]>([]);
const saving = ref(false);

const { data: prefData } = useMyNotificationPreference();
const { data: categoriesData } = useMyNotifiableCategories();

const updateMutation = useUpdateMyNotificationPreference();

const categoryOptions = computed(() =>
  (categoriesData.value?.items ?? []).map((c) => ({
    label: c.name ?? "",
    value: c.id ?? 0,
  }))
);

function minutesToDate(minutes: number): Date {
  const d = new Date();
  d.setHours(Math.floor(minutes / 60), minutes % 60, 0, 0);
  return d;
}

function dateToMinutes(value: Date | undefined): number | undefined {
  if (!(value instanceof Date) || Number.isNaN(value.getTime())) return undefined;
  return value.getHours() * 60 + value.getMinutes();
}

// 服务端状态到达/变化时回填本地编辑态（保存成功后的 refetch 也会走这里）
watch(
  prefData,
  (pref) => {
    if (!pref) return;
    quietEnabled.value = !!pref.quietEnabled;
    quietStart.value = minutesToDate(pref.quietStartMinute ?? 1320);
    quietEnd.value = minutesToDate(pref.quietEndMinute ?? 480);
    mutedCategoryIds.value = [...(pref.mutedCategoryIds ?? [])];
  },
  { immediate: true }
);

async function handleSave() {
  const startMinute = dateToMinutes(quietStart.value);
  const endMinute = dateToMinutes(quietEnd.value);
  if (quietEnabled.value && (startMinute === undefined || endMinute === undefined)) {
    ElMessage.warning($t("pages.user.profile.notification.quietTimeRequired"));
    return;
  }
  if (quietEnabled.value && startMinute === endMinute) {
    ElMessage.warning($t("pages.user.profile.notification.quietZeroWidth"));
    return;
  }

  saving.value = true;
  try {
    await updateMutation.mutateAsync({
      quietEnabled: quietEnabled.value,
      quietStartMinute: startMinute,
      quietEndMinute: endMinute,
      mutedCategoryIds: mutedCategoryIds.value,
    });
    ElMessage.success($t("pages.user.profile.notification.saveSuccess"));
  } catch (error: any) {
    console.error("Failed to save notification preferences:", error);
    ElMessage.error(error?.message || $t("pages.user.profile.notification.saveFailed"));
  } finally {
    saving.value = false;
  }
}
</script>

<style lang="scss" scoped>
.notification-preference-page {
  max-width: 560px;
}

.section-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);

  &.mt {
    margin-top: 24px;
  }
}

.section-hint {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}

.section-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  margin-bottom: 16px;

  .switch-label {
    font-size: 14px;
    color: var(--el-text-color-regular);
  }

  .field-label {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-right: 8px;
  }

  &.checkbox-block {
    display: block;

    :deep(.el-checkbox-group) {
      display: flex;
      flex-wrap: wrap;
      gap: 8px 16px;
    }
  }
}

.footer-hint {
  margin-top: 16px;
}
</style>
