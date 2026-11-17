import React, { useMemo, useState } from 'react';
import { Button, Empty, Input, Popover, Tabs, theme, Tooltip } from 'antd';
import { DownOutlined, SearchOutlined } from '@ant-design/icons';
import { Icon } from '@iconify/react';
import * as AntdIcons from '@ant-design/icons';
import { lucideIconNames } from './iconifyOffline';

interface IconPickerProps {
  /** 当前值：iconify 名称（lucide:xxx）或 Ant Design 组件名（XxxOutlined） */
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  disabled?: boolean;
}

/** Ant Design 图标导出名都以这三类后缀结尾，恰好把它们和非图标导出区分开 */
const ANTICON_PATTERN = /(Outlined|Filled|TwoTone)$/;

const GRID_MAX_HEIGHT = 260;
const POPOVER_WIDTH = 360;
const CELL_SIZE = 34;

/**
 * 图标选择器：输入框（可手输任意 iconify 名称）+ 弹层网格（Lucide / Ant Design
 * 双页签、按名搜索、点选回填）。渲染双格式与 iconResolver 同约定，菜单
 * meta.icon 存什么就能显示什么。
 */
const IconPicker: React.FC<IconPickerProps> = ({
  value,
  onChange,
  placeholder,
  searchPlaceholder,
  disabled,
}) => {
  const currentValue = value ?? '';
  const { token } = theme.useToken();
  const [open, setOpen] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [activeTab, setActiveTab] = useState<'lucide' | 'antd'>(() =>
    ANTICON_PATTERN.test(currentValue) ? 'antd' : 'lucide',
  );

  const antdIconNames = useMemo(
    () =>
      Object.keys(AntdIcons)
        .filter((name) => ANTICON_PATTERN.test(name))
        .sort(),
    [],
  );

  /** 双格式渲染：iconify 名称走 Icon，其余按 Ant Design 组件名解析 */
  const renderIcon = (name: string): React.ReactNode => {
    if (name.includes(':')) {
      return <Icon icon={name} width={16} height={16} />;
    }
    const Component = (AntdIcons as unknown as Record<string, React.ComponentType>)[name];
    return Component ? <Component /> : null;
  };

  const renderGrid = (names: string[], buildValue: (name: string) => string) => {
    const kw = keyword.trim().toLowerCase();
    const filtered = kw ? names.filter((name) => name.toLowerCase().includes(kw)) : names;
    if (filtered.length === 0) {
      return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} />;
    }
    return (
      <div style={{ maxHeight: GRID_MAX_HEIGHT, overflowY: 'auto', paddingRight: 4 }}>
        {filtered.map((name) => {
          const iconValue = buildValue(name);
          const selected = iconValue === currentValue;
          return (
            <Tooltip key={iconValue} title={iconValue}>
              <Button
                size="small"
                type={selected ? 'primary' : 'text'}
                variant={selected ? 'outlined' : undefined}
                disabled={disabled}
                aria-label={iconValue}
                onClick={() => {
                  onChange?.(iconValue);
                  setOpen(false);
                }}
                style={{ width: CELL_SIZE, height: CELL_SIZE, margin: 2, padding: 0 }}
              >
                <span style={{ display: 'inline-flex', fontSize: 17, lineHeight: 0 }}>
                  {renderIcon(iconValue)}
                </span>
              </Button>
            </Tooltip>
          );
        })}
      </div>
    );
  };

  return (
    <Popover
      trigger="click"
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setKeyword('');
      }}
      placement="bottomLeft"
      arrow={false}
      content={
        <div style={{ width: POPOVER_WIDTH }}>
          <Input
            allowClear
            value={keyword}
            placeholder={searchPlaceholder}
            prefix={<SearchOutlined style={{ color: token.colorTextQuaternary }} />}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <Tabs
            size="small"
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as 'lucide' | 'antd')}
            items={[
              {
                key: 'lucide',
                label: 'Lucide',
                children: renderGrid(lucideIconNames, (name) => `lucide:${name}`),
              },
              {
                key: 'antd',
                label: 'Ant Design',
                children: renderGrid(antdIconNames, (name) => name),
              },
            ]}
          />
        </div>
      }
    >
      <Input
        value={currentValue}
        placeholder={placeholder}
        disabled={disabled}
        allowClear
        onChange={(event) => onChange?.(event.target.value)}
        prefix={currentValue ? renderIcon(currentValue) : null}
        suffix={<DownOutlined style={{ color: token.colorTextQuaternary }} />}
      />
    </Popover>
  );
};

export default IconPicker;
