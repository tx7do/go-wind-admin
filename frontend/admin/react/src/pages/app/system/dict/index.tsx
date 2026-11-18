import { useState } from 'react';

import SplitPane from '@/components/common/SplitPane';
import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import DictTypeList from './DictTypeList';
import DictEntryList from './DictEntryList';

/**
 * 字典管理页面
 * 使用 SplitPane 实现左右分栏布局：左侧字典类型，右侧字典条目（窄容器自动上下堆叠）
 */
const DictManagement = () => {
  const [currentTypeId, setCurrentTypeId] = useState<number | null>(null);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <SplitPane>
        <SplitPane.Panel
          collapsible
          defaultSize="45%"
          min="25%"
          max="75%"
          style={{ display: 'flex', flexDirection: 'column' }}
        >
          <DictTypeList currentTypeId={currentTypeId} onTypeSelect={setCurrentTypeId} />
        </SplitPane.Panel>
        <SplitPane.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <DictEntryList typeId={currentTypeId} />
        </SplitPane.Panel>
      </SplitPane>
    </ContentContainer>
  );
};

export default DictManagement;
