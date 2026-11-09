import { useState } from 'react';
import { Card, Descriptions, Spin, Table, Tag, Typography, Empty, theme } from 'antd';
import type { TableColumnsType } from 'antd';
import { useTranslation } from 'react-i18next';
import type {
  redis_cacheservicev1_RedisCacheMonitorInfo,
  redis_cacheservicev1_SlowLogEntry,
} from '@/api/generated/admin/service/v1';
import { useRedisCacheMonitorInfo } from '@/api/hooks/redis-cache-monitor';
import { formatDateTime } from '@/utils/date';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';

const { Title, Text } = Typography;

/**
 * Redis 缓存监控页面（只读）
 * 展示 Redis INFO（按 section 拆分的键值对）、DBSIZE、SLOWLOG。
 * 数据通过 GET /admin/v1/redis-cache-monitor 拉取，无任何写操作。
 */
const RedisCacheMonitor = () => {
  const { t } = useTranslation('redis-cache-monitor');
  const { token } = theme.useToken();

  const { data, isLoading } = useRedisCacheMonitorInfo();
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  if (isLoading) {
    return (
      <ContentContainer heightMode="auto" scrollable padding="16px">
        <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
          <Spin size="large" />
        </div>
      </ContentContainer>
    );
  }

  const info = data as redis_cacheservicev1_RedisCacheMonitorInfo | undefined;
  const sections = info?.sections ?? [];
  const dbSize = Number(info?.dbSize ?? 0); // uint64 → string，转回 number 供比较/展示
  const slowlog = info?.slowlog ?? [];

  return (
    // 长卡片流页面：fixed 模式的高度约束链（flex:1 占满内容区）与表格页一致、可靠；
    // scrollable 让容器自身 overflow:auto 整页滚动——十几个 INFO 卡片+慢日志表
    // 超出视口时必须能滚，默认 fixed 是 overflow:hidden 会直接裁掉内容。
    <ContentContainer heightMode="fixed" scrollable padding="16px">
      {/* 单值指标：DBSIZE */}
      <Card
        size="small"
        style={{ marginBottom: 16 }}
        title={
          <Text strong>
            {t('dbSize')}
          </Text>
        }
      >
        <Descriptions column={1} size="small" bordered colon>
          <Descriptions.Item label={t('dbSize')}>
            <Tag color={dbSize > 0 ? 'orange' : 'default'}>{String(dbSize)}</Tag>
          </Descriptions.Item>
        </Descriptions>
      </Card>

      {/* INFO sections：每个 section 一张可折叠卡片 */}
      {sections.length === 0 ? (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Empty description={t('noInfo')} />
        </Card>
      ) : (
        sections.map((section, idx) => {
          const sectionName = section?.name ?? `${t('section')} ${idx}`;
          const key = `${sectionName}-${idx}`;
          const expanded = expandedSections[key] ?? false;
          return (
            <Card
              key={key}
              size="small"
              style={{ marginBottom: 16 }}
              title={
                <div
                  style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}
                  onClick={() =>
                    setExpandedSections((prev) => ({ ...prev, [key]: !prev[key] }))
                  }
                >
                  <Text strong>{sectionName}</Text>
                  <Tag>{(section?.entries ?? []).length}</Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {expanded ? t('collapse') : t('expand')}
                  </Text>
                </div>
              }
            >
              {expanded ? (
                (section?.entries ?? []).length === 0 ? (
                  <Empty description={t('noEntries')} />
                ) : (
                  <Descriptions
                    column={1}
                    size="small"
                    bordered
                    colon
                    contentStyle={{ wordBreak: 'break-all' }}
                    labelStyle={{ width: 240 }}
                  >
                    {(section?.entries ?? []).map((entry, eIdx) => (
                      <Descriptions.Item key={eIdx} label={entry?.key ?? ''}>
                        <span style={{ wordBreak: 'break-all' }}>{entry?.value ?? ''}</span>
                      </Descriptions.Item>
                    ))}
                  </Descriptions>
                )
              ) : (
                // 收起态用轻量提示行，不用 Empty 插画——Empty 语义是“无数据”，
                // 大片收起时会让整页看起来像加载失败/空表
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('collapsed')}
                </Text>
              )}
            </Card>
          );
        })
      )}

      {/* SLOWLOG：静态数据表 */}
      <Card
        size="small"
        title={
          <Text strong>
            {t('slowlog')}
          </Text>
        }
      >
        <Table<redis_cacheservicev1_SlowLogEntry>
          rowKey={(record) =>
            // Redis SLOWLOG 的 entry id 单次查询内唯一，作为稳定 key；
            // antd v6 已弃用 rowKey 函数的 index 参数，不能再用它兜底。
            String(record?.id ?? '')
          }
          dataSource={slowlog}
          pagination={false}
          size="small"
          scroll={{ x: 'max-content' }}
          locale={{ emptyText: <Empty description={t('noSlowlog')} /> }}
          columns={slowLogColumns(t, token.colorTextSecondary)}
        />
      </Card>

      <Title level={5} type="secondary" style={{ marginTop: 16 }}>
        {t('disclaimer')}
      </Title>
    </ContentContainer>
  );
};

function slowLogColumns(
  t: (key: string) => string,
  _secondaryColor: string,
): TableColumnsType<redis_cacheservicev1_SlowLogEntry> {
  return [
    {
      title: t('slowlogId'),
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      render: (val) => formatDateTime(val as any),
    },
    {
      title: t('durationUsec'),
      dataIndex: 'durationUsec',
      width: 120,
    },
    {
      title: t('clientAddr'),
      dataIndex: 'clientAddr',
      width: 200,
    },
    {
      title: t('clientName'),
      dataIndex: 'clientName',
      width: 120,
    },
    {
      title: t('args'),
      dataIndex: 'args',
      render: (args: unknown) => {
        const arr = (args as string[] | undefined) ?? [];
        return (
          <span style={{ wordBreak: 'break-all', whiteSpace: 'pre-wrap' }}>
            {arr.join(' ') || '-'}
          </span>
        );
      },
    },
  ];
}

export default RedisCacheMonitor;
