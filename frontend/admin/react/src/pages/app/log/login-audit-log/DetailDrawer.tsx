import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_LoginAuditLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';
import { getStatusMap, getActionTypeMap, getRiskLevelMap } from './constants';

interface DetailDrawerProps {
  open: boolean;
  data: auditservicev1_LoginAuditLog | null;
  onClose: () => void;
}

/**
 * 登录审计日志详情抽屉（只读）。
 * 展示单条日志的全部字段，与列表共用状态/动作/风险等级的配色与名称映射。
 */
const LoginAuditLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('login-audit-log');
  const statusMap = getStatusMap(t);
  const actionTypeMap = getActionTypeMap(t);
  const riskLevelMap = getRiskLevelMap(t);

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
        <Descriptions.Item label={t('status')}>
          {(() => {
            const cfg =
              data?.status ? statusMap[data.status as keyof typeof statusMap] : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('username')}>
          {data?.username || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('userId')}>
          {data?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('actionType')}>
          {(() => {
            const cfg =
              data?.actionType
                ? actionTypeMap[data.actionType as keyof typeof actionTypeMap]
                : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('loginMethod')}>
          {data?.loginMethod || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('riskLevel')}>
          {(() => {
            const cfg =
              data?.riskLevel
                ? riskLevelMap[data.riskLevel as keyof typeof riskLevelMap]
                : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('riskScore')}>
          {data?.riskScore ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tenantName')}>
          {data?.tenantName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('geoLocation')} span={2}>
          {data?.geoLocation?.province || ''} {data?.geoLocation?.city || ''}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionDevice')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('platform')}>
          {data?.deviceInfo?.platform || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('osName')}>
          {data?.deviceInfo?.osName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('browserName')}>
          {data?.deviceInfo?.browserName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('userAgent')} span={2}>
          {data?.deviceInfo?.userAgent || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('mfaStatus')}>
          {data?.mfaStatus || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sessionId')}>
          {data?.sessionId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('failureReason')} span={2}>
          {data?.failureReason || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('riskFactors')} span={2}>
          {(data?.riskFactors && data.riskFactors.length > 0) ? (
            <span>
              {data.riskFactors.map((factor, idx) => (
                <Tag key={idx} style={{ margin: '0 4px 4px 0' }}>
                  {factor}
                </Tag>
              ))}
            </span>
          ) : (
            '-'
          )}
        </Descriptions.Item>
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

export default LoginAuditLogDetailDrawer;
