import { Fragment, type MouseEventHandler, type ReactNode } from 'react';
import { App, Button, Dropdown, Popconfirm, Tooltip } from 'antd';
import { MoreOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

/**
 * 列表操作列统一组件（设计语言 §4「列表操作列」定式的代码化）：
 * 行内 link 弱化钮「图标+文本」，判活后 >3 个只留前 3 个，其余收进「更多」下拉。
 * 「更多」里放不下 Popconfirm，带 confirm 的项自动降级 modal.confirm（danger 项确认键同款 danger）。
 *
 * 用法：操作列 render 里以描述符数组声明动作，条件按钮用 hidden 剔除
 * （在计溢出前过滤，权限/角色隐藏后不足 3 个就不出「更多」）：
 *
 *   render: (_, record) => (
 *     <TableActions
 *       record={record}
 *       actions={[
 *         { key: 'edit', label: t('edit'), icon: <EditOutlined />, onClick: () => open(record) },
 *         { key: 'delete', label: t('delete'), icon: <DeleteOutlined />, danger: true,
 *           confirm: { title: t('deleteConfirmTitle') }, onClick: () => del(record) },
 *         ...(isPlatform ? [] : [{ key: 'reset', hidden: true, ... }]),
 *       ]}
 *     />
 *   ),
 *
 * 语义色：danger 红墨；success 走 --gowind-success-text 文字档（填充级 success 对白底
 * 对比度不足，见设计语言 §2.1）。「更多」菜单项仅识别 danger（antd 菜单无 success 语义）。
 * 按钮点击一律 stopPropagation，行内可展开/可选中行的页面无需各自处理。
 */
export interface TableActionConfirm {
  title: ReactNode;
  description?: ReactNode;
  /** 确认键 loading（如强制下线这类带请求的确认） */
  okLoading?: boolean;
}

export interface TableAction {
  key: string;
  label: ReactNode;
  icon?: ReactNode;
  onClick: (record: any) => void;
  danger?: boolean;
  success?: boolean;
  confirm?: TableActionConfirm;
  /** 行内按钮的 Tooltip 提示（仅行内；溢出进「更多」菜单后不渲染 Tooltip） */
  tooltip?: ReactNode;
  hidden?: boolean;
}

interface TableActionsProps {
  record: any;
  actions: TableAction[];
  /** 行内最多按钮数，超出收「更多」；默认 3（§4 定式值） */
  maxInline?: number;
}

export default function TableActions({ record, actions, maxInline = 3 }: TableActionsProps) {
  const { t } = useTranslation('common');
  const { modal } = App.useApp();

  const visible = (actions || []).filter((item) => !item.hidden);
  const inline = visible.slice(0, maxInline);
  const overflow = visible.slice(maxInline);

  const runItem = (item: TableAction) => item.onClick(record);

  const confirmInMenu = (item: TableAction) =>
    modal.confirm({
      title: item.confirm?.title,
      content: item.confirm?.description,
      okText: t('button.ok'),
      cancelText: t('button.cancel'),
      ...(item.danger || item.confirm?.okLoading
        ? { okButtonProps: { danger: item.danger, loading: item.confirm?.okLoading } }
        : {}),
      onOk: () => runItem(item),
    });

  const renderButton = (item: TableAction, onClick?: MouseEventHandler) => (
    <Button
      type="link"
      size="small"
      danger={item.danger}
      icon={item.icon}
      style={item.success ? { color: 'var(--gowind-success-text)' } : undefined}
      onClick={onClick}
    >
      {item.label}
    </Button>
  );

  const nodes = inline.map((item) => {
    const button = item.confirm ? (
      <Popconfirm
        title={item.confirm.title}
        description={item.confirm.description}
        okText={t('button.ok')}
        cancelText={t('button.cancel')}
        okButtonProps={
          item.confirm.okLoading ? { loading: item.confirm.okLoading } : undefined
        }
        onConfirm={(e) => {
          e?.stopPropagation();
          runItem(item);
        }}
        onCancel={(e) => e?.stopPropagation()}
      >
        {renderButton(item, (e) => e.stopPropagation())}
      </Popconfirm>
    ) : (
      renderButton(item, (e) => {
        e.stopPropagation();
        runItem(item);
      })
    );
    return (
      <Fragment key={item.key}>
        {item.tooltip ? <Tooltip title={item.tooltip}>{button}</Tooltip> : button}
      </Fragment>
    );
  });

  if (overflow.length > 0) {
    nodes.push(
      <Dropdown
        key="__more"
        menu={{
          items: overflow.map((item) => ({
            key: item.key,
            icon: item.icon,
            label: item.label,
            danger: item.danger,
            onClick: () => (item.confirm ? confirmInMenu(item) : runItem(item)),
          })),
        }}
      >
        <Tooltip title={t('button.more')}>
          <Button type="link" size="small" onClick={(e) => e.stopPropagation()}>
            <MoreOutlined />
          </Button>
        </Tooltip>
      </Dropdown>,
    );
  }

  return <>{nodes}</>;
}
