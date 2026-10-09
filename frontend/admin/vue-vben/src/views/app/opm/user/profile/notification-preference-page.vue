<script lang="ts" setup>
import { computed, ref, watch } from 'vue';

import { Page } from '@vben/common-ui';
import { $t } from '@vben/locales';

import {
  Button,
  Checkbox,
  CheckboxGroup,
  Switch,
  TimePicker,
  message,
} from 'ant-design-vue';

import {
  useMyNotificationPreference,
  useMyNotifiableCategories,
  useUpdateMyNotificationPreference,
} from '#/api';

/**
 * 个人中心「通知偏好」（通知域 P3）：
 *  - 静音时段只抑制 SSE 实时推送，消息仍进收件箱；
 *  - 分类退订只约束全员广播，点对点定向发送不受影响。
 * 验证码等事务性出站邮件不经偏好层，不受本页任何配置影响。
 */

const quietEnabled = ref(false);
// 起/止各自独立绑定（"HH:mm" 字符串）：跨零点窗口（start>end）是合法配置
const quietStart = ref<string>('22:00');
const quietEnd = ref<string>('08:00');
const mutedCategoryIds = ref<number[]>([]);
const saving = ref(false);

const { data: prefData } = useMyNotificationPreference();
const { data: categoriesData } = useMyNotifiableCategories();

const updateMutation = useUpdateMyNotificationPreference();

const categoryOptions = computed(() =>
  (categoriesData.value?.items ?? []).map((c) => ({
    label: c.name ?? '',
    value: c.id ?? 0,
  })),
);

function timeToMinutes(value: string | undefined): number | undefined {
  if (!value) return undefined;
  const parts = value.split(':');
  const h = Number(parts[0]);
  const m = Number(parts[1]);
  if (Number.isNaN(h) || Number.isNaN(m)) return undefined;
  return h * 60 + m;
}

function minutesToTime(minutes: number): string {
  const h = Math.floor(minutes / 60)
    .toString()
    .padStart(2, '0');
  const m = (minutes % 60).toString().padStart(2, '0');
  return `${h}:${m}`;
}

// 服务端状态到达/变化时回填本地编辑态（保存成功后的 refetch 也会走这里）
watch(
  prefData,
  (pref) => {
    if (!pref) return;
    quietEnabled.value = !!pref.quietEnabled;
    quietStart.value = minutesToTime(pref.quietStartMinute ?? 1320);
    quietEnd.value = minutesToTime(pref.quietEndMinute ?? 480);
    mutedCategoryIds.value = [...(pref.mutedCategoryIds ?? [])];
  },
  { immediate: true },
);

async function handleSave() {
  const startMinute = timeToMinutes(quietStart.value);
  const endMinute = timeToMinutes(quietEnd.value);
  if (quietEnabled.value && (startMinute === undefined || endMinute === undefined)) {
    message.warning($t('page.user.profile.notification.quietTimeRequired'));
    return;
  }
  if (quietEnabled.value && startMinute === endMinute) {
    message.warning($t('page.user.profile.notification.quietZeroWidth'));
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
    message.success($t('page.user.profile.notification.saveSuccess'));
  } catch (error: any) {
    message.error(error?.message || $t('page.user.profile.notification.saveFailed'));
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <Page auto-content-height>
    <div class="notification-preference-page">
      <h3 class="section-title">{{ $t('page.user.profile.notification.quietTitle') }}</h3>
      <div class="section-hint">
        {{ $t('page.user.profile.notification.quietDesc') }}
      </div>
      <div class="section-row">
        <Switch v-model:checked="quietEnabled" />
        <span class="switch-label">
          {{ $t('page.user.profile.notification.quietEnable') }}
        </span>
      </div>
      <div v-if="quietEnabled" class="section-row">
        <span class="field-label">
          {{ $t('page.user.profile.notification.quietTimeRange') }}
        </span>
        <!-- 两个独立单点选择器而非 TimeRangePicker：antd-vue 的范围时间选择器会把
             start>end（跨零点）的合法窗口交换/改写（实测 4.2.6），单点选择器无此问题；
             valueFormat 字符串绑定，省 dayjs 转换 -->
        <TimePicker
          v-model:value="quietStart"
          value-format="HH:mm"
          format="HH:mm"
          :minute-step="5"
          :allow-clear="false"
          :placeholder="$t('page.user.profile.notification.startTime')"
          style="width: 120px"
        />
        <span>-</span>
        <TimePicker
          v-model:value="quietEnd"
          value-format="HH:mm"
          format="HH:mm"
          :minute-step="5"
          :allow-clear="false"
          :placeholder="$t('page.user.profile.notification.endTime')"
          style="width: 120px"
        />
      </div>

      <h3 class="section-title mt">
        {{ $t('page.user.profile.notification.muteTitle') }}
      </h3>
      <div class="section-hint">
        {{ $t('page.user.profile.notification.muteDesc') }}
      </div>
      <div v-if="categoryOptions.length > 0" class="section-row checkbox-block">
        <CheckboxGroup v-model:value="mutedCategoryIds">
          <div class="category-list">
            <Checkbox v-for="item in categoryOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </Checkbox>
          </div>
        </CheckboxGroup>
      </div>
      <div v-else class="section-hint">
        {{ $t('page.user.profile.notification.noCategories') }}
      </div>

      <Button type="primary" :loading="saving" @click="handleSave">
        {{ $t('page.user.profile.notification.save') }}
      </Button>

      <div class="section-hint footer-hint">
        {{ $t('page.user.profile.notification.transactionalNote') }}
      </div>
    </div>
  </Page>
</template>

<style lang="scss" scoped>
.notification-preference-page {
  max-width: 560px;
}

.section-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;

  &.mt {
    margin-top: 24px;
  }
}

.section-hint {
  font-size: 12px;
  color: hsl(var(--foreground) / 60%);
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
  }

  .field-label {
    font-size: 13px;
    color: hsl(var(--foreground) / 60%);
    margin-right: 8px;
  }

  &.checkbox-block {
    display: block;

    .category-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      align-items: flex-start;
    }
  }
}

.footer-hint {
  margin-top: 16px;
}
</style>
