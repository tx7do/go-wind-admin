import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Tag, App } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_LoginAuditLog as LoginAuditLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListLoginAuditLogs } from '@/api/hooks/login-audit-log';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import {
  getStatusMap,
  getStatusOptions,
  getActionTypeMap,
  getActionTypeOptions,
  getRiskLevelMap,
  getRiskLevelOptions,
} from './constants';

/**
 * 登录审计日志页面
 */
const LoginAuditLogPage = () => {
  const { t } = useTranslation('login-audit-log');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const statusMap = getStatusMap(t);
  const actionTypeMap = getActionTypeMap(t);
  const riskLevelMap = getRiskLevelMap(t);

  const columns: ProColumns<LoginAuditLog>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 60,
      hideInSearch: true,
      render: (_, _record, index) => {
        const pagination = actionRef.current?.pageInfo;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || 20;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 170,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: t('status'),
      dataIndex: 'status',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: getStatusOptions(t),
      },
      render: (_, record) => {
        const config = statusMap[record.status as keyof typeof statusMap];
        if (!config) return record.status || '-';
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('username'),
      dataIndex: 'username',
      width: 120,
    },
    {
      title: t('actionType'),
      dataIndex: 'actionType',
      width: 120,
      valueType: 'select',
      fieldProps: {
        options: getActionTypeOptions(t),
      },
      render: (_, record) => {
        const config = actionTypeMap[record.actionType as keyof typeof actionTypeMap];
        if (!config) return record.actionType || '-';
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('riskLevel'),
      dataIndex: 'riskLevel',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: getRiskLevelOptions(t),
      },
      render: (_, record) => {
        const config = riskLevelMap[record.riskLevel as keyof typeof riskLevelMap];
        if (!config) return record.riskLevel || '-';
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('platform'),
      dataIndex: 'deviceInfo',
      width: 180,
      hideInSearch: true,
      render: (_, record) => {
        const info = record.deviceInfo as any;
        if (!info) return '-';
        return `${info.osName || ''} ${info.browserName || ''}`.trim() || '-';
      },
    },
    {
      title: t('geoLocation'),
      dataIndex: 'geoLocation',
      width: 150,
      hideInSearch: true,
      render: (_, record) => {
        const geo = record.geoLocation as any;
        if (!geo) return '-';
        return `${geo.province || ''} ${geo.city || ''}`.trim() || '-';
      },
    },
    {
      title: t('ipAddress'),
      dataIndex: 'ipAddress',
      width: 140,
    },
  ];

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <div ref={containerRef} className="page-container-content">
        <ProTable<LoginAuditLog>
          actionRef={actionRef}
          columns={columns}
          request={async (params, sorter) => {
            try {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 20,
                },
                formValues: Object.fromEntries(
                  Object.entries(params).filter(
                    ([key]) => !['current', 'pageSize'].includes(key),
                  ),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? key : `-${key}`,
                      )
                    : undefined,
              });

              const response = await fetchListLoginAuditLogs(query);

              return {
                data: response.items || [],
                total: response.total || 0,
                success: true,
              };
            } catch (error: any) {
              message.error(error.message || t('fetchFailed'));
              return {
                data: [],
                total: 0,
                success: false,
              };
            }
          }}
          rowKey="id"
          search={{
            labelWidth: 'auto',
            defaultCollapsed: false,
          }}
          pagination={{
            defaultPageSize: TABLE.DEFAULT_PAGE_SIZE,
            showSizeChanger: true,
            showQuickJumper: true,
            position: ['bottomRight'],
          }}
          options={{
            density: true,
            fullScreen: true,
            setting: true,
            reload: true,
          }}
          size="middle"
          bordered
          cardBordered={false}
          scroll={{
            y: tableScrollY,
            x: 1300,
          }}
        />
      </div>
    </ContentContainer>
  );
};

export default LoginAuditLogPage;
