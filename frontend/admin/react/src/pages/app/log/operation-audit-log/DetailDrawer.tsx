import type { CSSProperties } from 'react';
import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_OperationAuditLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';
import { getActionMap, successToColor, successToName } from './constants';

interface DetailDrawerProps {
  open: boolean;
  data: auditservicev1_OperationAuditLog | null;
  onClose: () => void;
}

// 变更前/后的长文本代码块卡
const preTextStyle: CSSProperties = {
  margin: 0,
  maxHeight: 240,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontFamily: 'inherit',
};

/**
 * 操作审计日志详情抽屉（只读）。
 * 展示单条日志的全部字段，与列表共用操作类型/成功状态的配色与名称映射。
 */
const OperationAuditLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('operation-audit-log');
  const actionMap = getActionMap(t);

  const descsProps: DescriptionsProps = {
    column: 2,
    bordered: true,
    size: 'small',
  };

  return (
    <Drawer
      open={open}
      onClose={onClose}
      size={780}
      title={t('common:modal.view', { moduleName: t('title') })}
      destroyOnHidden
    >
      <Divider titlePlacement="left">{t('sectionBasic')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('createdAt')}>
          {formatDateTime(data?.createdAt) || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('success')}>
          {data?.success === undefined || data?.success === null ? (
            '-'
          ) : (
            <Tag color={successToColor(data.success)}>
              {successToName(t, data.success)}
            </Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('action')}>
          {(() => {
            const cfg = data?.action
              ? actionMap[data.action as keyof typeof actionMap]
              : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : data?.action || '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('sensitiveLevel')}>
          {data?.sensitiveLevel && data.sensitiveLevel !== 'SENSITIVE_LEVEL_UNSPECIFIED'
            ? t(`sensitiveLevel.${data.sensitiveLevel}`)
            : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('resourceType')}>
          {data?.resourceType || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('resourceId')}>
          {data?.resourceId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('username')}>
          {data?.username || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('userId')}>
          {data?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tenantName')}>
          {data?.tenantName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('geoLocation')} span={2}>
          {data?.geoLocation
            ? `${data.geoLocation.province || ''} ${data.geoLocation.city || ''}`.trim() || '-'
            : '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionData')}</Divider>
      <Descriptions {...descsProps} column={1}>
        <Descriptions.Item label={t('beforeData')}>
          {data?.beforeData ? <pre style={preTextStyle}>{data.beforeData}</pre> : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('afterData')}>
          {data?.afterData ? <pre style={preTextStyle}>{data.afterData}</pre> : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('failureReason')}>
          {data?.failureReason || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('requestId')}>
          {data?.requestId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('traceId')}>
          {data?.traceId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('logHash')} span={2}>
          {data?.logHash || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('signature')} span={2}>
          {data?.signature || '-'}
        </Descriptions.Item>
      </Descriptions>
    </Drawer>
  );
};

export default OperationAuditLogDetailDrawer;
