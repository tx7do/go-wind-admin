import { useEffect, useState } from 'react';
import { Button, Empty, Input, Modal, Popconfirm, Table, Tag, Typography, Upload, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, FileAddOutlined, UploadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import type { aiservicev1_AiDoc as AiDoc, aiservicev1_AiKnowledgeBase as AiKnowledgeBase } from '@/api/generated/admin/service/v1';
import { fetchListAiDocs, useDeleteAiDoc, useUploadAiDoc, useUploadDocFile } from '@/api/hooks/ai-knowledge';

// 大文件安全转 base64（分块避免 btoa 栈溢出）
function bytesToBase64(buf: ArrayBuffer): string {
  const bytes = new Uint8Array(buf);
  let binary = '';
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(binary);
}

interface DocsDrawerProps {
  open: boolean;
  data?: AiKnowledgeBase;
  onClose: () => void;
}

export default function DocsDrawer({ open, data, onClose }: DocsDrawerProps) {
  const { t } = useTranslation('aiKnowledge');
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const [docs, setDocs] = useState<AiDoc[]>([]);
  const [loading, setLoading] = useState(false);
  const [name, setName] = useState('');
  const [content, setContent] = useState('');

  const uploadMutation = useUploadAiDoc();
  const uploadFileMutation = useUploadDocFile();
  const deleteMutation = useDeleteAiDoc();

  const loadDocs = (baseId: string) => {
    setLoading(true);
    fetchListAiDocs(baseId)
      .then((res) => setDocs((res.items ?? []) as AiDoc[]))
      .catch((error: Error) => {
        console.error('fetch ai docs failed:', error);
        message.error(t('fetchFailed'));
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    if (open && data?.id) {
      loadDocs(data.id);
    }
    if (!open) {
      setName('');
      setContent('');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, data?.id]);

  const handleUpload = async () => {
    if (!data?.id || !name.trim() || !content.trim()) return;
    try {
      const res = await uploadMutation.mutateAsync({ baseId: data.id, name: name.trim(), content });
      message.success(t('uploadSuccess', { chunks: res.chunkCount ?? 0 }));
      setName('');
      setContent('');
      loadDocs(data.id);
      queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
    } catch (error) {
      console.error('upload ai doc failed:', error);
      message.error((error as Error).message || t('uploadFailed'));
    }
  };

  const handleDelete = async (doc: AiDoc) => {
    if (!data?.id) return;
    try {
      await deleteMutation.mutateAsync({ baseId: data.id, id: doc.id! });
      message.success(t('deleteSuccess'));
      loadDocs(data.id);
      queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
    } catch (error) {
      console.error('delete ai doc failed:', error);
      message.error((error as Error).message || t('fetchFailed'));
    }
  };

  // 文件上传：读 base64 交后端抽取入库（不自动上传）
  const handleFileUpload = async (file: File) => {
    if (!data?.id) return false;
    try {
      const b64 = bytesToBase64(await file.arrayBuffer());
      const res = await uploadFileMutation.mutateAsync({
        baseId: data.id,
        fileName: file.name,
        contentBase64: b64,
      });
      message.success(t('uploadSuccess', { chunks: res.chunkCount ?? 0 }));
      loadDocs(data.id);
      queryClient.invalidateQueries({ queryKey: ['listAiKnowledgeBases'] });
    } catch (error) {
      console.error('upload ai doc file failed:', error);
      message.error((error as Error).message || t('uploadFailed'));
    }
    return false;
  };

  const columns: ColumnsType<AiDoc> = [
    { title: t('docName'), dataIndex: 'name', ellipsis: true },
    {
      title: t('docStatus'),
      dataIndex: 'status',
      width: 90,
      render: (_, record) =>
        record.status === 'READY' ? (
          <Tag color="success">READY</Tag>
        ) : (
          <Tag color="error">{record.status || 'FAILED'}</Tag>
        ),
    },
    { title: t('docChunks'), dataIndex: 'chunkCount', width: 80 },
    {
      title: t('action'),
      width: 80,
      render: (_, record) => (
        <Popconfirm title={t('deleteDocConfirm')} onConfirm={() => handleDelete(record)}>
          <Button type="link" size="small" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      ),
    },
  ];

  return (
    <Modal
      title={`${t('docsTitle', { name: data?.name ?? '' })}`}
      open={open}
      onCancel={onClose}
      footer={null}
      width={680}
      destroyOnHidden
    >
      {/* 上传区：纯文本直接入库（切片 → 向量化） */}
      <div className="mb-4 space-y-2">
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('docNamePlaceholder')}
          maxLength={100}
        />
        <Input.TextArea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder={t('docContentPlaceholder')}
          rows={5}
          maxLength={50000}
          showCount
        />
        <div className="flex items-center gap-2">
          <Button
            type="primary"
            icon={<UploadOutlined />}
            loading={uploadMutation.isPending}
            disabled={!name.trim() || !content.trim()}
            onClick={handleUpload}
          >
            {t('upload')}
          </Button>
          <Upload
            beforeUpload={handleFileUpload}
            showUploadList={false}
            disabled={uploadFileMutation.isPending}
            accept=".txt,.md,.markdown,.csv,.log,.json,.xml,.yml,.yaml,.html,.htm,.docx,.pdf"
          >
            <Button icon={<FileAddOutlined />} loading={uploadFileMutation.isPending}>
              {t('uploadFile')}
            </Button>
          </Upload>
          <Typography.Text type="secondary" className="!text-xs">
            {t('supportedFormats')}
          </Typography.Text>
        </div>
      </div>

      <Table<AiDoc>
        rowKey="id"
        size="small"
        loading={loading}
        columns={columns}
        dataSource={docs}
        pagination={false}
        locale={{ emptyText: <Empty description={t('noDocs')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
      />
    </Modal>
  );
}
