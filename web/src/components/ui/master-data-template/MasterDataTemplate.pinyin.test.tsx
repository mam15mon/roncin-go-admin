import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, expect, it } from 'vitest';
import { MasterDataTemplate } from './MasterDataTemplate';

afterEach(cleanup);
it('全量主数据列表本地搜索支持名称全拼和首字母', () => {
  render(
    <MasterDataTemplate
      title="货币管理"
      formFields={[]}
      searchPlaceholder="名称搜索"
      items={[
        { id: '1', code: 'CNY', name: '人民币', enabled: true },
        { id: '2', code: 'USD', name: '美元', enabled: true },
      ]}
    />,
  );
  const input = screen.getByPlaceholderText('名称搜索');
  for (const keyword of ['renminbi', 'rmb']) {
    fireEvent.change(input, { target: { value: keyword } });
    expect(screen.getByText('人民币')).toBeInTheDocument();
    expect(screen.queryByText('美元')).not.toBeInTheDocument();
  }
});
