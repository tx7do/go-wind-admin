import type { CSSProperties } from 'react';
import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_PermissionAuditLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';
import { getActionMap } from './constants';

interface DetailDrawerProps {
  open: boolean;
  data: auditservicev1_PermissionAuditLog | null;
  onClose: () => void;
}

// 原值/新值的长文本代码块卡
const preTextStyle: CSSProperties = {
  margin: 0,
  maxHeight: 240,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontFamily: 'inherit',
};

/**
 * 权限审计日志详情抽屉（只读）。
 * 展示单条日志的全部字段，与列表共用操作类型的配色与名称映射。
 */
const PermissionAuditLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('permission-audit-log');
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
        <Descriptions.Item label={t('action')}>
          {(() => {
            const cfg = data?.action
              ? actionMap[data.action as keyof typeof actionMap]
              : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : data?.action || '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('targetType')}>
          {data?.targetType || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('targetName')}>
          {data?.targetName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('targetId')}>
          {data?.targetId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('operatorName')}>
          {data?.operatorName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('operatorId')}>
          {data?.operatorId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('reason')} span={2}>
          {data?.reason || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionChange')}</Divider>
      <Descriptions {...descsProps} column={1}>
        <Descriptions.Item label={t('oldValue')}>
          {data?.oldValue ? <pre style={preTextStyle}>{data.oldValue}</pre> : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('newValue')}>
          {data?.newValue ? <pre style={preTextStyle}>{data.newValue}</pre> : '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tenantId')}>
          {data?.tenantId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('requestId')}>
          {data?.requestId || '-'}
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

export default PermissionAuditLogDetailDrawer;
