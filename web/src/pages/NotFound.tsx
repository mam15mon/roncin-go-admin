import { history } from '@/router/history';
import { Button, Result } from 'antd';
import React from 'react';

export default function NotFound() {
  return (
    <Result
      status="404"
      title="404"
      subTitle="页面不存在"
      extra={
        <Button type="primary" onClick={() => history.push('/welcome')}>
          返回工作台
        </Button>
      }
    />
  );
}
