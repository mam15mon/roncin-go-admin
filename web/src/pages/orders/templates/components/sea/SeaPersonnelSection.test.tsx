import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import type { FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { PersonnelAssignmentFields } from './SeaPersonnelSection';

describe('订单人员选择', () => {
  it('切换所属公司时清空不再匹配的人员，不自动选择首项', async () => {
    let form: FormInstance | undefined;

    function Fixture() {
      const [instance] = Form.useForm();
      form = instance;
      return (
        <Form
          form={instance}
          initialValues={{ organizationId: 'org-a', userId: 'user-a' }}
        >
          <PersonnelAssignmentFields
            label="销售"
            userField="userId"
            organizationField="organizationId"
            options={[
              {
                organizationId: 'org-a',
                organizationName: '公司甲',
                userId: 'user-a',
                displayName: '用户甲',
              },
              {
                organizationId: 'org-b',
                organizationName: '公司乙',
                userId: 'user-b',
                displayName: '用户乙',
              },
            ]}
          />
        </Form>
      );
    }

    render(<Fixture />);
    fireEvent.mouseDown(screen.getAllByRole('combobox')[0]);
    fireEvent.click(await screen.findByText('公司乙'));

    await waitFor(() => expect(form?.getFieldValue('userId')).toBeUndefined());
  });

  it('选择人员时自动联动带出所属公司', async () => {
    let form: FormInstance | undefined;

    function Fixture() {
      const [instance] = Form.useForm();
      form = instance;
      return (
        <Form form={instance}>
          <PersonnelAssignmentFields
            label="操作"
            userField="operatorUserId"
            organizationField="operatorOrganizationId"
            options={[
              {
                organizationId: 'org-b',
                organizationName: '公司乙',
                userId: 'user-b',
                displayName: '用户乙',
              },
            ]}
          />
        </Form>
      );
    }

    render(<Fixture />);
    const comboboxes = screen.getAllByRole('combobox');
    expect(comboboxes).toHaveLength(2);

    // 点击第二个 combobox（人员选择）
    fireEvent.mouseDown(comboboxes[1]);
    fireEvent.click(await screen.findByText('用户乙'));

    await waitFor(() => {
      expect(form?.getFieldValue('operatorUserId')).toBe('user-b');
      expect(form?.getFieldValue('operatorOrganizationId')).toBe('org-b');
    });
  });
});
