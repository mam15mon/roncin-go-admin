import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import type { TemplateProps } from '../../types';
import {
  buildSeaPersonnelSection,
  PersonnelAssignmentFields,
} from './SeaPersonnelSection';

// buildSeaPersonnelSection 仅消费 personnelOptions/creator/isDetail，
// 其余模板属性按 types.ts 补全为空实现。
const baseTemplateProps: TemplateProps = {
  cargoCategoryOptions: [],
  locationOptions: [],
  searchLocations: vi.fn().mockResolvedValue([]),
  currencyOptions: [],
  containerSpecOptions: [],
  searchCustomers: vi.fn().mockResolvedValue([]),
  searchShippingLines: vi.fn().mockResolvedValue([]),
  searchBookingAgents: vi.fn().mockResolvedValue([]),
  searchForeignAgents: vi.fn().mockResolvedValue([]),
  searchShippingAgents: vi.fn().mockResolvedValue([]),
  setCustomerCode: vi.fn(),
  personnelOptions: [],
};

/** 提成归属相关三岗：操作人员仅新建模式必填；业务/客服由客户带入且只读。 */
const COMMISSION_STAFF_LABELS = ['操作人员', '业务人员', '客服人员'] as const;
const OPTIONAL_STAFF_LABELS = ['关联人员', '单证人员', '商务人员'] as const;

function expectLabelRequired(label: string, required: boolean) {
  const labelElement = screen.getByText(label).closest('label');
  expect(labelElement).not.toBeNull();
  if (required) {
    expect(labelElement).toHaveClass('ant-form-item-required');
  } else {
    expect(labelElement).not.toHaveClass('ant-form-item-required');
  }
}

function SectionHost({
  isDetail,
  onFinish,
}: {
  isDetail?: boolean;
  onFinish: (values: unknown) => void;
}) {
  const section = buildSeaPersonnelSection({
    ...baseTemplateProps,
    isDetail,
  });
  return (
    <Form onFinish={(values) => onFinish(values)}>
      {section.content}
      <button type="submit">提交</button>
    </Form>
  );
}

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
            nameOnly
            options={[
              {
                userId: 'user-a',
                displayName: '张三',
                departmentNames: ['财务部', '核算组'],
              },
              { userId: 'user-b', displayName: '李四', departmentNames: [] },
            ]}
          />
        </Form>
      );
    }

    render(<Fixture />);
    const comboboxes = screen.getAllByRole('combobox');
    expect(comboboxes).toHaveLength(1);
    expect(comboboxes[0]).toHaveAttribute('autocomplete', 'off');
    expect(comboboxes[0]).toHaveAttribute('data-1p-ignore', 'true');
    expect(comboboxes[0]).toHaveAttribute('data-lpignore', 'true');
    expect(comboboxes[0]).toHaveAttribute('data-bwignore', 'true');
    expect(comboboxes[0]).toHaveAttribute('data-form-type', 'other');
    fireEvent.mouseDown(comboboxes[0]);
    expect(await screen.findByText('张三')).toBeInTheDocument();
    expect(screen.queryByText(/·/)).not.toBeInTheDocument();
    fireEvent.click(await screen.findByText('李四'));

    await waitFor(() => {
      expect(form?.getFieldValue('operatorUserId')).toBe('user-b');
      expect(form?.getFieldValue('operatorOrganizationId')).toBeUndefined();
    });
    expect(
      screen.getByRole('combobox').closest('.ant-select'),
    ).toHaveTextContent('李四');
  });
});

describe('buildSeaPersonnelSection 提成三岗必填', () => {
  it('新建模式下仅操作人员必填，业务/客服与其余人员保持选填且只读', () => {
    render(<SectionHost onFinish={vi.fn()} />);

    expectLabelRequired('操作人员', true);
    for (const label of ['业务人员', '客服人员', ...OPTIONAL_STAFF_LABELS]) {
      expectLabelRequired(label, false);
    }
    // 业务/客服由客户带入，选择器恒只读。
    for (const label of ['业务人员', '客服人员']) {
      const labelElement = screen.getByText(label).closest('label');
      const formItem = labelElement?.closest('.ant-form-item');
      expect(
        formItem?.querySelector('.ant-select-disabled'),
      ).not.toBeNull();
    }
  });

  it('详情模式下三岗不施加必填规则', () => {
    render(<SectionHost isDetail onFinish={vi.fn()} />);

    for (const label of COMMISSION_STAFF_LABELS) {
      expectLabelRequired(label, false);
    }
  });

  it('新建模式缺操作人员提交被阻断；业务/客服允许为空', async () => {
    const onFinish = vi.fn();
    render(<SectionHost onFinish={onFinish} />);

    fireEvent.click(screen.getByRole('button', { name: '提交' }));

    expect(await screen.findByText('请选择操作人员')).toBeInTheDocument();
    expect(screen.queryByText('请选择业务人员')).not.toBeInTheDocument();
    expect(screen.queryByText('请选择客服人员')).not.toBeInTheDocument();
    expect(onFinish).not.toHaveBeenCalled();
  });
});
