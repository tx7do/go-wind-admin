import { history, useIntl } from '@umijs/max';
import { Button, Card, Result } from 'antd';
import React from 'react';

/**
 * 403 禁止访问页面
 * 当用户没有权限访问时显示
 */
const ForbiddenPage: React.FC = () => {
  const intl = useIntl();

  return (
    <Card variant="borderless" style={{ minHeight: 'calc(100vh - 112px)' }}>
      <Result
        status="403"
        title="403"
        subTitle={intl.formatMessage({ id: 'pages.403.subTitle' })}
        extra={
          <Button type="primary" onClick={() => history.push('/')}>
            {intl.formatMessage({ id: 'pages.403.buttonText' })}
          </Button>
        }
      />
    </Card>
  );
};

export default ForbiddenPage;
