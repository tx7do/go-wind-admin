import { useRef, useState } from 'react';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import TableExportButton from '@/components/common/TableExportButton';
import TableActions from '@/components/common/TableActions';
import ListTable from '@/components/common/ListTable';
import { Button, Tag, App, Upload } from 'antd';
import {
  DeleteOutlined,
  DownloadOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { storageservicev1_File as FileItem } from '@/api/generated/admin/service/v1';
import { PaginationQuery } from '@/core';
import { fetchListFiles, useDeleteFile } from '@/api/hooks/file';
import { useUploadFile, useDownloadFile } from '@/api/hooks/file-transfer';
import { useProTableScrollY } from '@/hooks/useProTableScrollY';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import { getProviderMap, formatFileSize } from './constants';
import FileDrawer from './components/FileDrawer';
import { searchOptionRender } from '@/components/common/searchSubmitter';
import { ColumnSlidersIcon } from '@/components/common/columnSettingIcon';

/**
 * 文件管理页面
 */
const FileManagement = () => {
  const { t } = useTranslation('file');
  const actionRef = useRef<ActionType>(null);
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const containerRef = useRef<HTMLDivElement>(null);
  const tableScrollY = useProTableScrollY(containerRef);

  // Drawer 状态管理
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedFile, setSelectedFile] = useState<FileItem | undefined>();

  const providerMap = getProviderMap(t);

  // 删除操作
  const deleteMutation = useDeleteFile({
    onSuccess: () => {
      message.success(t('deleteSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listFiles'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('deleteFailed'));
    },
  });

  // 上传操作
  const uploadMutation = useUploadFile({
    onSuccess: () => {
      message.success(t('uploadSuccess'));
      queryClient.invalidateQueries({ queryKey: ['listFiles'] });
    },
    onError: (error: Error) => {
      message.error(error.message || t('uploadFailed'));
    },
  });

  // 下载操作
  const downloadMutation = useDownloadFile();

  // 处理上传
  const handleUpload = (file: File) => {
    uploadMutation.mutate({
      bucketName: 'images',
      fileDirectory: 'temp',
      file,
      method: 'post',
    });
    return false; // 阻止 antd 默认上传行为
  };

  // 处理下载
  const handleDownload = (record: FileItem) => {
    const objectName = record.fileDirectory && record.saveFileName
      ? `${record.fileDirectory}/${record.saveFileName}`
      : '';
    if (record.bucketName && objectName) {
      downloadMutation.mutate({
        bucketName: record.bucketName as string,
        objectName,
        preferPresignedUrl: true,
      });
    }
  };

  // 列配置
  const columns: ProColumns<FileItem>[] = [
    {
      title: t('serial'),
      dataIndex: 'id',
      width: 80,
      hideInSearch: true,
      render: (_, record, index) => {
        const pagination = record.id !== undefined ? actionRef.current?.pageInfo : undefined;
        const page = pagination?.current || 1;
        const pageSize = pagination?.pageSize || 10;
        return (page - 1) * pageSize + index + 1;
      },
    },
    {
      title: t('fileName'),
      dataIndex: 'fileName',
      hideInSearch: true,
    },
    {
      title: t('saveFileName'),
      dataIndex: 'saveFileName',
    },
    {
      title: t('fileDirectory'),
      dataIndex: 'fileDirectory',
      hideInSearch: true,
    },
    {
      title: t('size'),
      dataIndex: 'size',
      width: 120,
      hideInSearch: true,
      render: (_, record) => record.sizeFormat || formatFileSize(record.size),
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      valueType: 'dateTime',
      hideInSearch: true,
      sorter: true,
    },
    {
      title: t('provider'),
      dataIndex: 'provider',
      width: 120,
      hideInSearch: true,
      render: (_, record) => {
        const provider = record.provider as keyof typeof providerMap;
        const config = providerMap[provider] || { text: provider, color: 'default' };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('action'),
      valueType: 'option',
      width: 150,
      render: (_, record) => (
        <TableActions
          record={record}
          actions={[
            {
              key: 'download',
              label: t('download'),
              icon: <DownloadOutlined />,
              onClick: () => handleDownload(record),
            },
            {
              key: 'delete',
              label: t('common:button.delete'),
              icon: <DeleteOutlined />,
              danger: true,
              confirm: {
                title: t('deleteConfirmTitle'),
                description: t('deleteConfirmDesc', { moduleName: t('moduleName') }),
              },
              onClick: () => record.id && deleteMutation.mutate({ id: record.id }),
            },
          ]}
        />
      ),
    },
  ];

  return (
    <>
      <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
        <div ref={containerRef} className="page-container-content">
          <ListTable<FileItem>
            actionRef={actionRef}
            columns={columns}
            request={async (params, sorter, _filter) => {
              const query = new PaginationQuery({
                paging: {
                  page: params.current || 1,
                  pageSize: params.pageSize || 10,
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
                    : ['-created_at'],
              });

              const response = await fetchListFiles(query);

              return {
                data: response.items || [],
                total: Number(response.total ?? 0),
                success: true,
              };
            }}
            rowKey="id"
            search={{ optionRender: searchOptionRender,
              labelWidth: 'auto',
              defaultCollapsed: false,
            }}
            pagination={false}
            toolBarRender={() => [
              <TableExportButton key="export" fetcher={fetchListFiles} columns={columns} filename="files" />,
              <Upload
                key="upload"
                showUploadList={false}
                beforeUpload={handleUpload}
                accept="*"
              >
                <Button
                  type="primary"
                  icon={<UploadOutlined />}
                  loading={uploadMutation.isPending}
                >
                  {t('upload')}
                </Button>
              </Upload>,
            ]}
            options={{
              density: true,
              fullScreen: true,
              setting: { settingIcon: <ColumnSlidersIcon /> },
              reload: true,
            }}
            size="middle"
            bordered
            cardBordered={false}
            scroll={{
              y: tableScrollY,
              x: 1000,
            }}
          />
        </div>
      </ContentContainer>

      {/* 文件编辑 Drawer */}
      <FileDrawer
        open={drawerOpen}
        mode="create"
        data={selectedFile}
        onClose={() => {
          setDrawerOpen(false);
          setSelectedFile(undefined);
        }}
        onSuccess={() => {
          actionRef.current?.reload();
        }}
      />
    </>
  );
};

export default FileManagement;
