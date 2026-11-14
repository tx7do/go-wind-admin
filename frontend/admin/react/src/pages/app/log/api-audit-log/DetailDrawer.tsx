import type { CSSProperties } from 'react';
import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_ApiAuditLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';

interface DetailDrawerProps {
  open: boolean;
  data: auditservicev1_ApiAuditLog | null;
  onClose: () => void;
}

/**
 * 成功状态 → Tag 预设色（与列表页 successToColor 同语义）
 */
function successToColor(success: boolean | undefined): string {
  if (success === true) return 'success';
  if (success === false) return 'error';
  return 'default';
}

// 请求方法 → Tag 预设色（与列表 httpMethod 列的 colorMap 同源）
const METHOD_COLOR_MAP: Record<string, string> = {
  GET: 'success',
  POST: 'processing',
  PUT: 'warning',
  DELETE: 'error',
};

// 请求体/请求头/响应内容的长文本代码块卡
const preTextStyle: CSSProperties = {
  margin: 0,
  maxHeight: 240,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontFamily: 'inherit',
};

/**
 * API 审计日志详情抽屉（只读）。
 * 展示单条日志的全部字段；请求体/请求头/响应内容用 pre 代码块卡承载长文本。
 */
const ApiAuditLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('api-audit-log');

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
              {t(data.success ? 'success.true' : 'success.false')}
              {data.statusCode ? ` (${data.statusCode})` : ''}
            </Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('username')}>
          {data?.username || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('userId')}>
          {data?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('httpMethod')}>
          {data?.httpMethod ? (
            <Tag color={METHOD_COLOR_MAP[data.httpMethod] || 'default'}>
              {data.httpMethod}
            </Tag>
          ) : (
            '-'
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('statusCode')}>
          {data?.statusCode ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('path')} span={2}>
          {data?.path || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('requestUri')} span={2}>
          {data?.requestUri || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('latencyMs')}>
          {data?.latencyMs ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tenantName')}>
          {data?.tenantName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('geoLocation')}>
          {data?.geoLocation?.province || ''}
          {data?.geoLocation?.city || ''}
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

      <Divider titlePlacement="left">{t('sectionApi')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('apiModule')}>
          {data?.apiModule || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('apiOperation')}>
          {data?.apiOperation || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('apiDescription')} span={2}>
          {data?.apiDescription || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('referer')} span={2}>
          {data?.referer || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('reason')} span={2}>
          {data?.reason || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('spanId')}>
          {data?.spanId || '-'}
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
        <Descriptions.Item label={t('requestBody')} span={2}>
          {data?.requestBody ? (
            <pre style={preTextStyle}>{data.requestBody}</pre>
          ) : (
            '-'
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('requestHeader')} span={2}>
          {data?.requestHeader ? (
            <pre style={preTextStyle}>{data.requestHeader}</pre>
          ) : (
            '-'
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('response')} span={2}>
          {data?.response ? <pre style={preTextStyle}>{data.response}</pre> : '-'}
        </Descriptions.Item>
      </Descriptions>
    </Drawer>
  );
};

export default ApiAuditLogDetailDrawer;
