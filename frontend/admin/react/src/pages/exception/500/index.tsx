import {history, useIntl} from '@umijs/max';
import {Button, Card, Result} from 'antd';
import React from 'react';

/**
 * 500 服务器错误页面
 * 当服务器发生内部错误时显示
 */
const ServerErrorPage: React.FC = () => {
  const intl = useIntl();

  return (
    <Card variant="borderless" style={{minHeight: 'calc(100vh - 112px)'}}>
      <Result
        status="500"
        title="500"
        subTitle={intl.formatMessage({id: 'pages.500.subTitle'})}
        extra={
          <Button type="primary" onClick={() => history.push('/')}>
            {intl.formatMessage({id: 'pages.500.buttonText'})}
          </Button>
        }
      />
    </Card>
  );
};

export default ServerErrorPage;
