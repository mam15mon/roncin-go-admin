import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { Form } from 'antd';
import { describe, expect, it } from 'vitest';
import { PersonnelAssignmentFields } from './SeaPersonnelSection';

describe('订单人员选择', () => {
  it('只选择人员，不暴露经营归属组织字段', async () => {
    let form: FormInstance | undefined;

    function Fixture() {
      const [instance] = Form.useForm();
      form = instance;
      return (
        <Form form={instance}>
          <PersonnelAssignmentFields
            label="操作"
            userField="operatorUserId"
            options={[
              { userId: 'user-a', displayName: '用户甲' },
              { userId: 'user-b', displayName: '用户乙' },
            ]}
          />
        </Form>
      );
    }

    render(<Fixture />);
    const comboboxes = screen.getAllByRole('combobox');
    expect(comboboxes).toHaveLength(1);
    fireEvent.mouseDown(comboboxes[0]);
    fireEvent.click(await screen.findByText('用户乙'));

    await waitFor(() => {
      expect(form?.getFieldValue('operatorUserId')).toBe('user-b');
      expect(form?.getFieldValue('operatorOrganizationId')).toBeUndefined();
    });
  });
});
