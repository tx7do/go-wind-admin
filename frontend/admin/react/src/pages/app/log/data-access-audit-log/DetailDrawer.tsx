import type { CSSProperties } from 'react';
import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_DataAccessAuditLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';
import {
  getAccessTypeMap,
  getDataCategoryMap,
  successToColor,
  successToName,
} from './constants';

interface DetailDrawerProps {
  open: boolean;
  data: auditservicev1_DataAccessAuditLog | null;
  onClose: () => void;
}

// SQL 语句等长文本代码块卡
const preTextStyle: CSSProperties = {
  margin: 0,
  maxHeight: 240,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontFamily: 'inherit',
};

/**
 * 数据访问审计日志详情抽屉（只读）。
 * 展示单条日志的全部字段，与列表共用访问类型/数据分类/成功状态的配色与名称映射。
 */
const DataAccessAuditLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('data-access-audit-log');
  const accessTypeMap = getAccessTypeMap(t);
  const dataCategoryMap = getDataCategoryMap(t);

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
        <Descriptions.Item label={t('accessType')}>
          {(() => {
            const cfg = data?.accessType
              ? accessTypeMap[data.accessType as keyof typeof accessTypeMap]
              : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : data?.accessType || '-';
          })()}
        </Descriptions.Item>
        <Descriptions.Item label={t('sensitiveLevel')}>
          {data?.sensitiveLevel && data.sensitiveLevel !== 'SENSITIVE_LEVEL_UNSPECIFIED'
            ? t(`sensitiveLevel.${data.sensitiveLevel}`)
            : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tableName')}>
          {data?.tableName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('dataCategory')}>
          {(() => {
            const cfg = data?.dataCategory
              ? dataCategoryMap[data.dataCategory as keyof typeof dataCategoryMap]
              : undefined;
            return cfg ? <Tag color={cfg.color}>{cfg.text}</Tag> : data?.dataCategory || '-';
          })()}
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
        <Descriptions.Item label={t('latencyMs')}>
          {data?.latencyMs ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionData')}</Divider>
      <Descriptions {...descsProps}>
        <Descriptions.Item label={t('dbUser')}>
          {data?.dbUser || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('dataSource')}>
          {data?.dataSource || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sqlDigest')} span={2}>
          {data?.sqlDigest || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sqlText')} span={2}>
          {data?.sqlText ? <pre style={preTextStyle}>{data.sqlText}</pre> : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('dataId')}>
          {data?.dataId || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('affectedRows')}>
          {data?.affectedRows ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('dataMasked')}>
          {data?.dataMasked === true
            ? t('dataMasked.true')
            : data?.dataMasked === false
              ? t('dataMasked.false')
              : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('businessPurpose')}>
          {data?.businessPurpose || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('maskingRules')} span={2}>
          {data?.maskingRules || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
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

export default DataAccessAuditLogDetailDrawer;
