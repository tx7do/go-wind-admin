import { Card, Descriptions, Spin, Tag, Typography, Empty, theme } from 'antd';
import { useTranslation } from 'react-i18next';
import type { server_monitorservicev1_ServerMonitorInfo } from '@/api/generated/admin/service/v1';
import { useServerMonitorInfo } from '@/api/hooks/server-monitor';
import { formatDateTime } from '@/utils/date';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

const { Title, Text } = Typography;

/**
 * 服务监控页面（只读）：Go 运行时 / 数据库连接池 / 主机信息。
 * 数据来自 GET /admin/v1/server-monitor，无任何写操作。
 */
const ServerMonitor = () => {
  const { t } = useTranslation('server-monitor');
  const { token } = theme.useToken();

  const { data, isLoading } = useServerMonitorInfo({
    refetchInterval: 10_000, // 每 10 秒自动刷新
  });

  if (isLoading) {
    return (
      <ContentContainer heightMode="auto" scrollable padding="16px">
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      </ContentContainer>
    );
  }

  const info = data as server_monitorservicev1_ServerMonitorInfo | undefined;
  const go = info?.go;
  const db = info?.database;
  const host = info?.host;

  const fmtBytes = (bytes?: number) => {
    if (!bytes && bytes !== 0) return '-';
    if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
    if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(2)} MB`;
    if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${bytes} B`;
  };

  const fmtUptime = (seconds?: number) => {
    if (!seconds && seconds !== 0) return '-';
    const d = Math.floor(seconds / 86400);
    const h = Math.floor((seconds % 86400) / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (d > 0) return `${d}d ${h}h ${m}m`;
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m ${Math.floor(seconds % 60)}s`;
  };

  const renderRow = (label: string, value: React.ReactNode) => (
    <Descriptions.Item key={label} label={label}>
      {value ?? '-'}
    </Descriptions.Item>
  );

  return (
    <ContentContainer heightMode="fixed" scrollable padding="16px">
      {/* 数据库 */}
      <Card size="small" style={{ marginBottom: 16 }} title={<Text strong>{t('database')}</Text>} extra={
        db?.pingOk === true ? <Tag color="success">{t('pingOk')}</Tag>
          : db?.pingOk === false ? <Tag color="error">{t('pingFail')}</Tag>
            : null
      }>
        {db ? (
          <Descriptions column={2} size="small" bordered colon>
            {renderRow(t('dbDriver'), db.driver)}
            {renderRow(t('maxOpenConnections'), String(db.maxOpenConnections ?? '-'))}
            {renderRow(t('openConnections'), String(db.openConnections ?? '-'))}
            {renderRow(t('inUseConnections'), String(db.inUseConnections ?? '-'))}
            {renderRow(t('idleConnections'), String(db.idleConnections ?? '-'))}
            {db.pingError ? renderRow(t('pingError'), <span style={{ color: token.colorError }}>{db.pingError}</span>) : null}
          </Descriptions>
        ) : (
          <Empty description={t('noData')} />
        )}
      </Card>

      {/* Go 运行时 */}
      <Card size="small" style={{ marginBottom: 16 }} title={<Text strong>{t('goRuntime')}</Text>}>
        {go ? (
          <Descriptions column={2} size="small" bordered colon>
            {renderRow(t('goVersion'), go.version)}
            {renderRow(t('numGoroutine'), String(go.numGoroutine ?? '-'))}
            {/* memAllocBytes/memSysBytes/uptimeSeconds 为 int64（proto3 JSON 下是字符串），先转回 number */}
            {renderRow(t('memAlloc'), fmtBytes(Number(go.memAllocBytes ?? 0)))}
            {renderRow(t('memSys'), fmtBytes(Number(go.memSysBytes ?? 0)))}
            {renderRow(t('gcCycles'), String(go.gcCycles ?? '-'))}
            {renderRow(t('uptime'), `${fmtUptime(Number(go.uptimeSeconds ?? 0))} (${formatDateTime(go.startedAt as any)})`)}
          </Descriptions>
        ) : (
          <Empty description={t('noData')} />
        )}
      </Card>

      {/* 主机 */}
      <Card size="small" style={{ marginBottom: 16 }} title={<Text strong>{t('host')}</Text>}>
        {host ? (
          <Descriptions column={2} size="small" bordered colon>
            {renderRow(t('hostname'), host.hostname)}
            {renderRow(t('os'), `${host.os ?? '-'} / ${host.arch ?? '-'}`)}
            {renderRow(t('numCpu'), String(host.numCpu ?? '-'))}
          </Descriptions>
        ) : (
          <Empty description={t('noData')} />
        )}
      </Card>

      <Title level={5} type="secondary" style={{ marginTop: 16 }}>
        {t('disclaimer')}
      </Title>
    </ContentContainer>
  );
};

export default ServerMonitor;
