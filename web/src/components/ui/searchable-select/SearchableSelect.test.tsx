import { ProForm } from '@ant-design/pro-components';
import { cleanup, render, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { ProFormSearchableSelect, SearchableSelect } from './SearchableSelect';
import { defaultSelectFilterOption } from './utils';

describe('SearchableSelect & defaultSelectFilterOption', () => {
  afterEach(() => {
    cleanup();
  });

  it('defaultSelectFilterOption 能够正确按 label, code, name, value 进行不区分大小写匹配', () => {
    const opt = {
      label: '海运订舱 (BOOKING)',
      value: 'uuid-1',
      code: 'BOOKING',
      name: '海运订舱',
    };

    // 匹配中文
    expect(defaultSelectFilterOption('订舱', opt)).toBe(true);
    // 匹配英文代码（不区分大小写）
    expect(defaultSelectFilterOption('book', opt)).toBe(true);
    expect(defaultSelectFilterOption('BOOKING', opt)).toBe(true);
    // 不匹配
    expect(defaultSelectFilterOption('报关', opt)).toBe(false);
    // 空输入
    expect(defaultSelectFilterOption('', opt)).toBe(true);
  });

  it('渲染 SearchableSelect 组件', () => {
    render(
      <SearchableSelect
        placeholder="请选择服务类型"
        options={[
          { label: '订舱', value: '1' },
          { label: '拖车', value: '2' },
        ]}
      />,
    );

    expect(screen.getByText('请选择服务类型')).toBeInTheDocument();
  });

  it('在 ProForm 中渲染 ProFormSearchableSelect', () => {
    render(
      <ProForm>
        <ProFormSearchableSelect
          name="serviceType"
          label="对应服务类型"
          placeholder="请搜索并选择"
          options={[
            { label: '订舱 (BOOKING)', value: 'BOOKING' },
            { label: '拖车 (TRUCKING)', value: 'TRUCKING' },
          ]}
        />
      </ProForm>,
    );

    expect(screen.getByText('对应服务类型')).toBeInTheDocument();
    expect(screen.getByText('请搜索并选择')).toBeInTheDocument();
  });
});
