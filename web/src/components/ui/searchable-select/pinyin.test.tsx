import { ProForm } from '@ant-design/pro-components';
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, expect, it } from 'vitest';
import { ProFormSearchableSelect, SearchableSelect } from './SearchableSelect';
import { defaultSelectFilterOption } from './utils';

afterEach(cleanup);
const options = [
  {
    label: '海运费 (OF)',
    value: 'fee-1',
    nameEn: 'Ocean Freight',
    aliasName: '运费别名',
  },
  { label: '拖车费 (TC)', value: 'fee-2' },
];

it('带 JSX 标签、英文名与别名的候选同样支持拼音', () => {
  expect(
    defaultSelectFilterOption('hyf', {
      label: (
        <span>
          海运费 <b>(OF)</b>
        </span>
      ),
    }),
  ).toBe(true);
  expect(defaultSelectFilterOption('freight', options[0])).toBe(true);
  expect(defaultSelectFilterOption('yfbm', options[0])).toBe(true);
  expect(
    defaultSelectFilterOption('object', { label: <span>海运费</span> }),
  ).toBe(false);
});

it.each(['普通下拉', 'Pro 表单下拉'])(
  '%s 输入全拼和首字母后仍展示中文选项',
  async (kind) => {
    render(
      kind === '普通下拉' ? (
        <SearchableSelect options={options} />
      ) : (
        <ProForm submitter={false}>
          <ProFormSearchableSelect name="fee" options={options} />
        </ProForm>
      ),
    );
    const input = screen.getByRole('combobox');
    fireEvent.mouseDown(input);
    for (const keyword of ['haiyunfei', 'hyf', 'yfbm', 'freight']) {
      fireEvent.change(input, { target: { value: keyword } });
      await waitFor(() => {
        expect(
          document.querySelector('.ant-select-item-option-content')
            ?.textContent,
        ).toBe('海运费 (OF)');
        expect(
          document.querySelectorAll('.ant-select-item-option'),
        ).toHaveLength(1);
      });
    }
  },
);

it('远程返回的中文候选不会被本地拼音匹配二次过滤', async () => {
  render(
    <ProForm submitter={false}>
      <ProFormSearchableSelect
        name="remote"
        options={options}
        fieldProps={{ filterOption: false }}
      />
    </ProForm>,
  );
  const input = screen.getByRole('combobox');
  fireEvent.mouseDown(input);
  fireEvent.change(input, { target: { value: '服务端已匹配的关键词' } });
  await waitFor(() =>
    expect(document.querySelectorAll('.ant-select-item-option')).toHaveLength(
      2,
    ),
  );
});
