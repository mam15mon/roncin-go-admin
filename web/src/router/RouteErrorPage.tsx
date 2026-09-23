import { ReloadOutlined } from '@ant-design/icons';
import { Button, Result } from 'antd';

export function RouteErrorPage() {
  return (
    <Result
      status="error"
      title="页面暂时无法打开"
      subTitle="请重新加载后重试。若问题持续，请联系系统管理员。"
      extra={
        <Button
          type="primary"
          icon={<ReloadOutlined aria-hidden />}
          href={window.location.href}
        >
          重新加载
        </Button>
      }
    />
  );
}
