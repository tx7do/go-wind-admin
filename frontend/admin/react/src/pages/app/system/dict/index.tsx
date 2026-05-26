import { Splitter } from 'antd';
import { useState } from 'react';

import ContentContainer from '@/layouts/components/PageContainer/ContentContainer';
import DictTypeList from './DictTypeList';
import DictEntryList from './DictEntryList';

/**
 * 字典管理页面
 * 使用 Splitter 实现左右分栏布局：左侧字典类型，右侧字典条目
 */
const DictManagement = () => {
  const [currentTypeId, setCurrentTypeId] = useState<number | null>(null);

  return (
    <ContentContainer heightMode="fixed" padding="16px" bottomMargin={0}>
      <Splitter style={{ height: '100%', flex: 1, minHeight: 0 }}>
        <Splitter.Panel collapsible defaultSize="40%" min="25%" max="55%" style={{ display: 'flex', flexDirection: 'column' }}>
          <DictTypeList currentTypeId={currentTypeId} onTypeSelect={setCurrentTypeId} />
        </Splitter.Panel>
        <Splitter.Panel style={{ display: 'flex', flexDirection: 'column' }}>
          <DictEntryList typeId={currentTypeId} />
        </Splitter.Panel>
      </Splitter>
    </ContentContainer>
  );
};

export default DictManagement;
