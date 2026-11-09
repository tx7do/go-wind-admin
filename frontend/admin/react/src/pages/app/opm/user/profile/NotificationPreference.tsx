import { useCallback, useEffect, useState } from 'react';
import { Button, Checkbox, TimePicker, Spin, Switch, Typography, App } from 'antd';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/api/client';
import dayjs, { type Dayjs } from 'dayjs';

const { Text, Title } = Typography;

/**
 * 个人中心「通知偏好」：实时推送静音时段 + 站内信分类退订（仅全员广播）。
 *
 * 后端语义（通知域 P3）：
 *  - 静音时段只抑制 SSE 实时推送（桌面通知/角标即时性），收件行照常落库；
 *  - 分类退订只约束全员广播，点对点定向发送不受影响。
 * 验证码等事务性出站邮件不经偏好层，不受本页任何配置影响。
 *
 * 静音时段使用两个独立的时间选择器（而非 RangePicker），以支持跨零点窗口（如 22:00 至 08:00）。
 * RangePicker 会强制 start < end，无法表达合法的跨天区间。
 */

const minutesToDayjs = (minutes: number) => dayjs().startOf('day').add(minutes, 'minute');

const dayjsToMinutes = (value: Dayjs | undefined): number | undefined => {
  if (!value?.isValid?.()) return undefined;
  return value.hour() * 60 + value.minute();
};

const NotificationPreference = () => {
  const { t } = useTranslation('profile');
  const { message } = App.useApp();
  const [saving, setSaving] = useState(false);

  const prefQuery = useQuery({
    queryKey: ['myNotificationPreference'],
    queryFn: () => apiClient.notificationPreferenceService.GetMyNotificationPreference({}),
  });
  const categoriesQuery = useQuery({
    queryKey: ['myNotifiableCategories'],
    queryFn: () => apiClient.notificationPreferenceService.ListMyNotificationCategories({}),
  });

  const [quietEnabled, setQuietEnabled] = useState<boolean>(false);
  const [quietStart, setQuietStart] = useState<Dayjs | undefined>(minutesToDayjs(1320));
  const [quietEnd, setQuietEnd] = useState<Dayjs | undefined>(minutesToDayjs(480));
  const [mutedCategoryIds, setMutedCategoryIds] = useState<string[]>([]); // 分类 id 为 uint64 字符串

  useEffect(() => {
    const pref = prefQuery.data;
    if (!pref) return;
    setQuietEnabled(!!pref.quietEnabled);
    setQuietStart(minutesToDayjs(pref.quietStartMinute ?? 1320));
    setQuietEnd(minutesToDayjs(pref.quietEndMinute ?? 480));
    setMutedCategoryIds(pref.mutedCategoryIds ?? []);
  }, [prefQuery.data]);

  const handleSave = useCallback(async () => {
    const startMinute = dayjsToMinutes(quietStart);
    const endMinute = dayjsToMinutes(quietEnd);
    if (quietEnabled && (startMinute === undefined || endMinute === undefined)) {
      message.warning(t('notification.quietTimeRequired'));
      return;
    }
    if (quietEnabled && startMinute === endMinute) {
      message.warning(t('notification.quietZeroWidth'));
      return;
    }

    setSaving(true);
    try {
      await apiClient.notificationPreferenceService.UpdateMyNotificationPreference({
        quietEnabled,
        quietStartMinute: startMinute,
        quietEndMinute: endMinute,
        mutedCategoryIds,
      });
      message.success(t('notification.saveSuccess'));
      prefQuery.refetch();
    } catch (error: any) {
      console.error('Failed to save notification preferences:', error);
      message.error(error.message || t('notification.saveFailed'));
    } finally {
      setSaving(false);
    }
  }, [quietEnabled, quietStart, quietEnd, mutedCategoryIds, prefQuery, message, t]);

  if (prefQuery.isLoading) {
    return <Spin style={{ display: 'block', margin: '48px auto' }} />;
  }

  const categoryOptions = (categoriesQuery.data?.items ?? []).map((c) => ({
    label: c.name ?? '',
    value: c.id ?? '',
  }));

  return (
    <div style={{ maxWidth: 560 }}>
      <Title level={5} style={{ marginBottom: 4 }}>{t('notification.quietTitle')}</Title>
      <Text type="secondary">{t('notification.quietDesc')}</Text>
      <div style={{ marginTop: 12, marginBottom: 16 }}>
        <Switch
          checked={quietEnabled}
          onChange={(checked) => setQuietEnabled(checked)}
          checkedChildren={t('notification.on')}
          unCheckedChildren={t('notification.off')}
        />
        <span style={{ marginLeft: 8 }}>{t('notification.quietEnable')}</span>
      </div>
      {quietEnabled && (
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
            {t('notification.quietTimeRange')}
          </Text>
          {/* 两个独立的时间选择器，支持跨零点窗口（如 22:00 至 08:00） */}
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <TimePicker
              value={quietStart}
              onChange={(value) => setQuietStart(value ?? undefined)}
              format="HH:mm"
              minuteStep={5}
              allowClear={false}
              placeholder={t('notification.startTime')}
              style={{ width: 130 }}
            />
            <span>-</span>
            <TimePicker
              value={quietEnd}
              onChange={(value) => setQuietEnd(value ?? undefined)}
              format="HH:mm"
              minuteStep={5}
              allowClear={false}
              placeholder={t('notification.endTime')}
              style={{ width: 130 }}
            />
          </div>
        </div>
      )}

      <Title level={5} style={{ marginBottom: 4, marginTop: 24 }}>{t('notification.muteTitle')}</Title>
      <Text type="secondary">{t('notification.muteDesc')}</Text>
      <div style={{ marginTop: 12, marginBottom: 16 }}>
        {categoryOptions.length > 0 ? (
          <Checkbox.Group
            options={categoryOptions}
            value={mutedCategoryIds}
            onChange={(values) => setMutedCategoryIds(values as string[])}
            style={{ display: 'flex', flexWrap: 'wrap', gap: '8px 16px' }}
          />
        ) : (
          <Text type="secondary">{t('notification.noCategories')}</Text>
        )}
      </div>

      <Button type="primary" onClick={handleSave} loading={saving}>
        {t('notification.save')}
      </Button>

      <Text type="secondary" style={{ display: 'block', marginTop: 16, fontSize: 12 }}>
        {t('notification.transactionalNote')}
      </Text>
    </div>
  );
};

export default NotificationPreference;
