import { useMemo, useState } from 'react';
import { App, Button, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import { useTranslation } from 'react-i18next';
import type { PaginationQuery } from '@/core';
import { exportAuditLogs, type AuditExportFormat } from '@/utils/csv';
import { serverExportFile } from '@/utils/server-export';

/**
 * 列表导出按钮（CSV / Excel 下拉）：内置分页聚合（默认上限 1 万行），
 * 按 ProColumns（dataIndex/title，自动剔除 hideInTable 与操作列）生成文件并触发下载。
 * fetcher 契约与审计导出一致：(query: PaginationQuery) => Promise<{ items, total }>。
 *
 * serverExport 提供时追加「服务端全量 (XLSX)」项：由后端生成（上限远大于客户端
 * 聚合），query 经 PaginationQuery 同源序列化透传——导出的即当前搜索看到的。
 * 仅接入与该列表读接口同权限语义的 :export 锚点（平台闸/租户视角由后端定）。
 */
export interface TableExportServerExport {
  /** 后端 :export 锚点（如 'admin/v1/ai/usage-logs:export'） */
  url: string
  /** 构造带当前搜索条件的查询（分页缺省=后端统一控行数） */
  buildQuery: () => PaginationQuery
}

export interface TableExportButtonProps {
  fetcher: (query: PaginationQuery) => Promise<{ items?: any[]; total?: number | string }>;
  /** ProColumns[]（或 {title, dataIndex}[]），自动过滤出可导出列 */
  columns: any[];
  filename?: string;
  maxRows?: number;
  /**
   * 强制附加的查询条件（与页面 request 闭包同源注入，如收件箱的 recipient_user_id）。
   * 导出走本组件自己的聚合查询、不经过表格 request，页面在闭包里塞的隐式过滤
   * 这里感知不到；不传会把隐私过滤一起丢掉，导出范围越权扩大。
   */
  params?: Record<string, any>;
  /** 可选：服务端全量导出锚点与查询构造（不提供则菜单不含该项） */
  serverExport?: TableExportServerExport;
}

export default function TableExportButton({
  fetcher,
  columns,
  filename = 'export',
  maxRows = 10_000,
  params,
  serverExport,
}: TableExportButtonProps) {
  const { t } = useTranslation('common');
  const { message } = App.useApp();
  const [exporting, setExporting] = useState(false);

  const exportColumns = useMemo(
    () =>
      (columns || [])
        .filter(
          (c: any) =>
            c.dataIndex && !c.hideInTable && c.valueType !== 'option',
        )
        .map((c: any) => ({
          key: String(c.dataIndex),
          title:
            typeof c.title === 'string' ? c.title : String(c.dataIndex),
        })),
    [columns],
  );

  const handleExport = async (format: AuditExportFormat) => {
    setExporting(true);
    try {
      const count = await exportAuditLogs(
        { fetcher, filename, columns: exportColumns, maxRows, params },
        format,
      );
      message.success(t('export.success', { count }));
    } catch (err: any) {
      message.error(`${t('export.failed')}：${err?.message ?? ''}`);
    } finally {
      setExporting(false);
    }
  };

  const handleServerExport = async () => {
    setExporting(true);
    try {
      await serverExportFile(serverExport!.url, serverExport!.buildQuery());
      message.success(t('export.serverSuccess'));
    } catch (err: any) {
      message.error(`${t('export.serverFailed')}：${err?.message ?? ''}`);
    } finally {
      setExporting(false);
    }
  };

  const items: MenuProps['items'] = [
    { key: 'csv', label: t('export.csv') },
    { key: 'xlsx', label: t('export.xlsx') },
  ];
  if (serverExport) {
    // 分隔线把服务端全量与客户端聚合两项隔开（口径同审计六页/vben 端）
    items.push({ type: 'divider' });
    items.push({ key: 'server-xlsx', label: t('export.serverXlsx') });
  }

  return (
    <Dropdown
      menu={{
        items,
        onClick: ({ key }) => {
          if (key === 'server-xlsx') {
            handleServerExport();
            return;
          }
          handleExport(key as AuditExportFormat);
        },
      }}
    >
      <Button loading={exporting}>
        {t('export.title')}
      </Button>
    </Dropdown>
  );
}
