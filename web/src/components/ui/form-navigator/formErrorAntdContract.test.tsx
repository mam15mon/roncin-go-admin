import { fireEvent, render, waitFor } from '@testing-library/react';
import { Button, Form, Input } from 'antd';
import { describe, expect, it } from 'vitest';
import { ANT_FORM_ITEM_ERROR_SELECTOR } from './formErrorUtils';

/**
 * antd 升级哨兵：formErrorUtils 的错误定位与统计依赖
 * `.ant-form-item-has-error` 这一 antd 内部类名（非公开 API）。
 * antd 大版本升级若移除该类名，本测试显式失败，提示同步迁移
 * formErrorUtils 的选择器，而不是让表单错误导航在线上静默失效。
 */
describe('antd 表单错误类名契约（formErrorUtils 升级哨兵）', () => {
  it('antd Form 校验失败时仍输出 ant-form-item-has-error 类名', async () => {
    const { getByRole } = render(
      <Form>
        <Form.Item
          name="orderNo"
          rules={[{ required: true, message: '请输入订单编号' }]}
        >
          <Input />
        </Form.Item>
        <Button type="primary" htmlType="submit">
          提交
        </Button>
      </Form>,
    );

    fireEvent.click(getByRole('button', { name: /提\s*交/ }));

    await waitFor(() => {
      expect(
        document.querySelector(ANT_FORM_ITEM_ERROR_SELECTOR),
      ).not.toBeNull();
    });
  });
});
