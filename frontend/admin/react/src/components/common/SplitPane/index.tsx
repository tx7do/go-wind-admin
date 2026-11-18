import { Children, isValidElement, useLayoutEffect, useRef, useState } from 'react';
import type { CSSProperties, ReactNode } from 'react';

import { Splitter } from 'antd';

/**
 * 三端统一的堆叠闸（权威值与推导见 docs/design-language.md §4）。
 *
 * 判据取「右栏按声明比例分到的宽度」而不是视口宽：各端 chrome（react 侧栏 224 + 页边距 32、
 * ele 210 + 约 32、vben 224 + 内容 p-4）不同且侧栏可折叠/全屏，分栏比例又每页不同
 * （55% / 60% / 75%），只有栏位的绝对宽度是三端同一的量。
 */
const RIGHT_PANE_MIN_WIDTH = 720;

/** 堆叠模式下每栏最小高度（px），与 ele 端 ProSplitter 同值 */
const MIN_PANEL_HEIGHT = 480;

export interface SplitPanePanelProps {
  children?: ReactNode;
  className?: string;
  style?: CSSProperties;
  collapsible?: boolean;
  size?: string;
  min?: string;
  max?: string;
  defaultSize?: string;
  resizable?: boolean;
}

/**
 * 栏位描述：宽屏时原样交给 antd Splitter.Panel，堆叠时只取 children。
 * 自身不渲染任何东西，装配错误由 SplitPane 抛出。
 */
function SplitPanePanel(_props: SplitPanePanelProps) {
  return null;
}

export interface SplitPaneProps {
  children: ReactNode;
  /** 右栏分配宽低于该值时改为上下堆叠 */
  stackBelowPaneWidth?: number;
  /** 堆叠模式下每栏最小高度 */
  minPanelHeight?: number;
}

function readPanels(children: ReactNode, count: number): SplitPanePanelProps[] {
  const list = Children.toArray(children);
  if (list.length !== count) {
    throw new Error(`SplitPane 需要且仅需要 ${count} 个 SplitPane.Panel，实际 ${list.length} 个`);
  }
  return list.map((child, index) => {
    if (!isValidElement(child) || child.type !== SplitPanePanel) {
      throw new Error(`SplitPane 的第 ${index + 1} 个子项不是 SplitPane.Panel`);
    }
    return child.props as SplitPanePanelProps;
  });
}

/**
 * 左右分栏容器，窄容器时自动改为上下堆叠。
 *
 * 窄容器下百分比栏位放不下内嵌表格（右栏 60% 时表格列宽合计已超栏宽），
 * 分栏读起来只剩两三列；堆叠后每栏独占整行、表格各自横向滚动。
 */
export const SplitPane = ({
  children,
  stackBelowPaneWidth = RIGHT_PANE_MIN_WIDTH,
  minPanelHeight = MIN_PANEL_HEIGHT,
}: SplitPaneProps) => {
  const panels = readPanels(children, 2);

  const leftSize = panels[0].defaultSize ?? panels[0].size;
  if (leftSize !== undefined && !leftSize.endsWith('%')) {
    throw new Error(`SplitPane 左栏尺寸需为百分比字符串（收到 "${leftSize}"），否则右栏占比无从推算`);
  }
  const rightRatio = 1 - (leftSize === undefined ? 50 : Number.parseFloat(leftSize) / 100);

  const hostRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const stacked = containerWidth > 0 && containerWidth * rightRatio < stackBelowPaneWidth;

  useLayoutEffect(() => {
    const el = hostRef.current;
    if (!el) return undefined;
    // 挂载即量一次：useState 初值 0 会让首帧按宽屏渲染，量完再翻会闪一帧分栏
    setContainerWidth(el.clientWidth);
    if (typeof ResizeObserver === 'undefined') return undefined;
    const observer = new ResizeObserver(() => setContainerWidth(el.clientWidth));
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <div
      ref={hostRef}
      style={{
        display: 'flex',
        flexDirection: 'column',
        flex: 1,
        minHeight: 0,
        gap: stacked ? 12 : 0,
        // 堆叠后两栏各占 minPanelHeight，超出固定高度的内容区会被父级 overflow:hidden 裁掉
        overflow: stacked ? 'auto' : 'hidden',
      }}
    >
      {stacked ? (
        panels.map((panel, index) => (
          <div
            key={index}
            style={{
              display: 'flex',
              flex: '1 1 50%',
              flexDirection: 'column',
              minHeight: minPanelHeight,
            }}
          >
            {/* 内层承接原 Panel 的 flex 列样式并撑满栏位：少了这一层，堆叠下的表格
                只按内容高度排，栏位下方留白（同 ele 端 __panel > :deep(*) 的做法） */}
            <div style={{ flex: '1 1 auto', minHeight: 0, ...panel.style }}>{panel.children}</div>
          </div>
        ))
      ) : (
        <Splitter style={{ height: '100%', flex: 1, minHeight: 0 }}>
          {panels.map((panel, index) => (
            <Splitter.Panel key={index} {...panel} />
          ))}
        </Splitter>
      )}
    </div>
  );
};

SplitPane.Panel = SplitPanePanel;

export default SplitPane;
