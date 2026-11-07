import { Modal, Table, Tag, Typography, App } from 'antd';
import { HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import type { biservicev1_BiDashboardRevision as Revision } from '@/api/generated/admin/service/v1';
import { useBiDashboardRevisions, useRestoreBiDashboardRevision } from '@/api/hooks/bi-dashboard';

interface RevisionsDrawerProps {
  open: boolean;
  dashboardId: number | undefined;
  onClose: () => void;
}

/**
 * 版本历史弹窗：每次保存留旧版快照（新快照在前），恢复 = 快照写回主表。
 * 恢复前服务端会先以 RESTORE 来源留底当前版本——误恢复可再恢复回来（可逆）。
 */
const RevisionsDrawer = ({ open, dashboardId, onClose }: RevisionsDrawerProps) => {
  const { t } = useTranslation('biDashboard');
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data, isLoading } = useBiDashboardRevisions(dashboardId, open);

  const restoreMutation = useRestoreBiDashboardRevision({
    onSuccess: () => {
      message.success(t('revisionRestoreSuccess'));
      queryClient.invalidateQueries({ queryKey: ['biDashboardRevisions', dashboardId] });
      queryClient.invalidateQueries({ queryKey: ['listBiDashboards'] });
    },
    onError: (error: Error) => message.error(error.message || t('revisionRestoreFailed')),
  });

  const columns = [
    {
      title: t('revisionVersion'),
      dataIndex: 'version',
      width: 80,
      render: (v: number) => <Tag color="blue">v{v}</Tag>,
    },
    { title: t('name'), dataIndex: 'name', ellipsis: true },
    {
      title: t('revisionSource'),
      dataIndex: 'source',
      width: 90,
      render: (s: string) =>
        s === 'RESTORE' ? <Tag color="orange">{t('revisionSourceRestore')}</Tag> : <Tag>{t('revisionSourceSave')}</Tag>,
    },
    { title: t('createdAt'), dataIndex: 'createdAt', width: 160 },
    {
      title: t('action'),
      width: 80,
      render: (_: unknown, record: Revision) => (
        <Typography.Link
          onClick={() => {
            restoreMutation.mutate({ id: record.id });
          }}
        >
          <HistoryOutlined /> {t('revisionRestore')}
        </Typography.Link>
      ),
    },
  ];

  return (
    <Modal
      title={t('revisionTitle')}
      open={open}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
    >
      <Table
        columns={columns as any}
        dataSource={data?.items || []}
        loading={isLoading}
        rowKey="id"
        size="small"
        pagination={false}
        scroll={{ y: 420 }}
      />
    </Modal>
  );
};

export default RevisionsDrawer;
