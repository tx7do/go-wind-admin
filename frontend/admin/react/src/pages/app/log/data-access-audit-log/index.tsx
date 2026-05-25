import { useRef } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Tag, App } from 'antd';
import { useTranslation } from 'react-i18next';
import type { auditservicev1_DataAccessAuditLog as DataAccessAuditLog } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { TABLE } from '@/config/constants';
import { fetchListDataAccessAuditLogs } from '@/api/hooks/data-access-audit-log';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import {
  getAccessTypeMap,
  getAccessTypeOptions,
  SUCCESS_STATUS_LIST,
  successToColor,
  successToName,
} from './constants';

/**
 * 数据访问审计日志页面
 */
const DataAccessAuditLogPage = () => {
  const { t } = useTranslation('data-access-audit-log');
  const actionRef = useRef<ActionType>(null);
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  const accessTypeMap = getAccessTypeMap(t);
  const accessTypeOptions = getAccessTypeOptions(t);

  const columns: ProColumns<DataAccessAuditLog>[] = [
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
      title: t('success'),
      dataIndex: 'success',
      width: 100,
      valueType: 'select',
      valueEnum: Object.fromEntries(
        SUCCESS_STATUS_LIST.map((item) => [
          item.value,
          { text: item.label, status: item.value === 'true' ? 'Success' : 'Error' },
        ]),
      ),
      render: (_, record) => (
        <Tag color={successToColor(record.success)}>{successToName(record.success)}</Tag>
      ),
    },
    {
      title: t('accessType'),
      dataIndex: 'accessType',
      width: 100,
      valueType: 'select',
      fieldProps: {
        options: accessTypeOptions,
      },
      render: (_, record) => {
        const config = accessTypeMap[record.accessType as keyof typeof accessTypeMap];
        if (!config) return record.accessType || '-';
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('tableName'),
      dataIndex: 'tableName',
      width: 180,
      ellipsis: true,
    },
    {
      title: t('dataCategory'),
      dataIndex: 'dataCategory',
      width: 150,
      hideInSearch: true,
      ellipsis: true,
    },
    {
      title: t('latencyMs'),
      dataIndex: 'latencyMs',
      width: 100,
      hideInSearch: true,
    },
    {
      title: t('username'),
      dataIndex: 'username',
      width: 120,
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
        <ProTable<DataAccessAuditLog>
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
                  Object.entries(params).filter(([key]) => !['current', 'pageSize'].includes(key)),
                ),
                orderBy:
                  sorter && Object.keys(sorter).length > 0
                    ? Object.entries(sorter).map(([key, value]) =>
                        value === 'ascend' ? key : `-${key}`,
                      )
                    : undefined,
              });

              const response = await fetchListDataAccessAuditLogs(query);

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
            x: 1400,
          }}
        />
      </div>
    </ContentContainer>
  );
};

export default DataAccessAuditLogPage;
