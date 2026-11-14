import type { CSSProperties } from 'react';
import { Drawer, Descriptions, Divider, Tag } from 'antd';
import type { DescriptionsProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { permissionservicev1_PolicyEvaluationLog } from '@/api/generated/admin/service/v1';
import { formatDateTime } from '@go-wind/admin-react';

interface DetailDrawerProps {
  open: boolean;
  data: permissionservicev1_PolicyEvaluationLog | null;
  onClose: () => void;
}

// 评估结果 → Tag 预设色（与列表页 resultToColor 同语义）
function resultToColor(result: boolean | undefined): string {
  if (result === true) return 'success';
  if (result === false) return 'error';
  return 'default';
}

// 请求方法 → Tag 预设色（与列表 requestMethod 列的 colorMap 同源）
const METHOD_COLOR_MAP: Record<string, string> = {
  GET: 'success',
  POST: 'processing',
  PUT: 'warning',
  DELETE: 'error',
  PATCH: 'default',
  HEAD: 'default',
  OPTIONS: 'default',
};

// 评估上下文/生效详情/数据范围 SQL 的长文本代码块卡
const preTextStyle: CSSProperties = {
  margin: 0,
  maxHeight: 240,
  overflow: 'auto',
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
  fontFamily: 'inherit',
};

/**
 * 策略评估日志详情抽屉（只读）。
 * 展示单条日志的全部字段；评估上下文等长文本用 pre 代码块卡承载。
 */
const PolicyEvaluationLogDetailDrawer = ({ open, data, onClose }: DetailDrawerProps) => {
  const { t } = useTranslation('policy-evaluation-log');

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
        <Descriptions.Item label={t('result')}>
          {data?.result === undefined || data?.result === null ? (
            '-'
          ) : (
            <Tag color={resultToColor(data.result)}>
              {t(data.result ? 'result.true' : 'result.false')}
            </Tag>
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('requestMethod')}>
          {data?.requestMethod ? (
            <Tag color={METHOD_COLOR_MAP[data.requestMethod] || 'default'}>
              {data.requestMethod}
            </Tag>
          ) : (
            '-'
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('userId')}>
          {data?.userId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('requestPath')} span={2}>
          {data?.requestPath || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('permissionId')}>
          {data?.permissionId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('policyId')}>
          {data?.policyId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('membershipId')}>
          {data?.membershipId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('tenantId')}>
          {data?.tenantId ?? '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('ipAddress')}>
          {data?.ipAddress || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('traceId')}>
          {data?.traceId || '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionEval')}</Divider>
      <Descriptions {...descsProps} column={1}>
        <Descriptions.Item label={t('evaluationContext')}>
          {data?.evaluationContext ? (
            <pre style={preTextStyle}>{data.evaluationContext}</pre>
          ) : (
            '-'
          )}
        </Descriptions.Item>
        <Descriptions.Item label={t('effectDetails')}>
          {data?.effectDetails ? <pre style={preTextStyle}>{data.effectDetails}</pre> : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('scopeSql')}>
          {data?.scopeSql ? <pre style={preTextStyle}>{data.scopeSql}</pre> : '-'}
        </Descriptions.Item>
      </Descriptions>

      <Divider titlePlacement="left">{t('sectionOther')}</Divider>
      <Descriptions {...descsProps}>
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

export default PolicyEvaluationLogDetailDrawer;
