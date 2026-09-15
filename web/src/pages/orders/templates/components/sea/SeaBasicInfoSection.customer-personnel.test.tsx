import { ProForm } from '@ant-design/pro-components';
import { render, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { PartnerAssignmentRole } from '@/enums.generated';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  extractPersonnelFromPartnerAssignments,
  SeaCustomerField,
} from './SeaBasicInfoSection';

vi.mock('@/services/roncin/partnerService', () => ({
  partnerServiceGetPartner: vi.fn(),
}));

let capturedProps: any = null;
vi.mock('../../../components/PartnerQuickAddSelect', () => ({
  default: (props: any) => {
    capturedProps = props;
    return (
      <div data-testid="mock-partner-quick-add">
        <button
          type="button"
          data-testid="trigger-change"
          onClick={() =>
            props.onPartnerChange?.({
              label: '测试委托单位',
              value: 'cust-1',
              code: 'CUST001',
            })
          }
        >
          Select Customer
        </button>
        <button
          type="button"
          data-testid="trigger-clear"
          onClick={() => props.onPartnerChange?.(undefined)}
        >
          Clear Customer
        </button>
      </div>
    );
  },
}));

describe('extractPersonnelFromPartnerAssignments', () => {
  it('当 assignments 为空或未提供时返回空对象', () => {
    expect(extractPersonnelFromPartnerAssignments(undefined)).toEqual({});
    expect(extractPersonnelFromPartnerAssignments([])).toEqual({});
  });

  it('正确映射所有客商岗位角色到订单表单内部信息字段', () => {
    const assignments: API.PartnerAssignment[] = [
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CREATOR,
        userId: 'user-creator',
        organizationId: 'org-creator',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
        userId: 'user-op',
        organizationId: 'org-op',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
        userId: 'user-sales',
        organizationId: 'org-sales',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_CUSTOMER_SERVICE,
        userId: 'user-cs',
        organizationId: 'org-cs',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_FINANCE,
        userId: 'user-finance',
        organizationId: 'org-finance',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_COMMERCIAL,
        userId: 'user-comm',
        organizationId: 'org-comm',
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT,
        userId: 'user-contact-2',
        organizationId: 'org-contact-2',
        sortOrder: 1,
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_INTERNAL_CONTACT,
        userId: 'user-contact-1',
        organizationId: 'org-contact-1',
        sortOrder: 0,
      },
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_DOCUMENT,
        userId: 'user-doc',
        organizationId: 'org-doc',
      },
    ];

    const result = extractPersonnelFromPartnerAssignments(assignments);

    expect(result).toEqual({
      operatorUserId: 'user-op',
      operatorOrganizationId: 'org-op',
      salesUserId: 'user-sales',
      salesOrganizationId: 'org-sales',
      customerServiceUserId: 'user-cs',
      customerServiceOrganizationId: 'org-cs',
      commercialUserId: 'user-comm',
      commercialOrganizationId: 'org-comm',
      associateUserId: 'user-contact-1',
      associateOrganizationId: 'org-contact-1',
      associate2UserId: 'user-contact-2',
      associate2OrganizationId: 'org-contact-2',
      documentUserId: 'user-doc',
      documentOrganizationId: 'org-doc',
    });

    // 确保创建人和财务人员未泄漏到订单表单字段
    expect(result).not.toHaveProperty('creatorUserId');
    expect(result).not.toHaveProperty('financeUserId');
  });

  it('当 assignment 缺少 organizationId 时，从 personnelOptions 兜底回补', () => {
    const assignments: API.PartnerAssignment[] = [
      {
        role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
        userId: 'user-op',
      },
    ];

    const personnelOptions = [
      {
        userId: 'user-op',
        organizationId: 'org-op-fallback',
      },
    ];

    const result = extractPersonnelFromPartnerAssignments(
      assignments,
      personnelOptions,
    );

    expect(result).toEqual({
      operatorUserId: 'user-op',
      operatorOrganizationId: 'org-op-fallback',
    });
  });
});

describe('SeaCustomerField 联动带出内部信息', () => {
  beforeEach(() => {
    capturedProps = null;
    vi.clearAllMocks();
  });

  it('选择委托单位后异步获取客商档案并自动反填人员信息', async () => {
    const setCustomerCode = vi.fn();
    const mockPartner = {
      id: 'cust-1',
      code: 'CUST001',
      legalName: '测试委托单位',
      assignments: [
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
          userId: 'sales-1',
          organizationId: 'org-sales',
        },
        {
          role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_OPERATOR,
          userId: 'operator-1',
          organizationId: 'org-operator',
        },
      ],
    };

    vi.mocked(partnerServiceGetPartner).mockResolvedValue({
      data: mockPartner,
    } as any);

    let formInstance: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formInstance = form;
      return (
        <ProForm form={form} submitter={false}>
          <SeaCustomerField
            searchCustomers={vi.fn().mockResolvedValue([])}
            setCustomerCode={setCustomerCode}
            personnelOptions={[]}
          />
        </ProForm>
      );
    };

    const { getByTestId } = render(<TestComponent />);

    getByTestId('trigger-change').click();

    expect(setCustomerCode).toHaveBeenCalledWith('CUST001');

    await waitFor(() => {
      expect(partnerServiceGetPartner).toHaveBeenCalledWith({ id: 'cust-1' });
    });

    await waitFor(() => {
      expect(formInstance.getFieldValue('salesUserId')).toBe('sales-1');
      expect(formInstance.getFieldValue('salesOrganizationId')).toBe('org-sales');
      expect(formInstance.getFieldValue('operatorUserId')).toBe('operator-1');
      expect(formInstance.getFieldValue('operatorOrganizationId')).toBe(
        'org-operator',
      );
    });
  });

  it('清空委托单位时平稳兼容，不触发档案请求', async () => {
    const setCustomerCode = vi.fn();

    const TestComponent = () => {
      const [form] = Form.useForm();
      return (
        <ProForm form={form} submitter={false}>
          <SeaCustomerField
            searchCustomers={vi.fn().mockResolvedValue([])}
            setCustomerCode={setCustomerCode}
            personnelOptions={[]}
          />
        </ProForm>
      );
    };

    const { getByTestId } = render(<TestComponent />);

    getByTestId('trigger-clear').click();

    expect(setCustomerCode).toHaveBeenCalledWith(undefined);
    expect(partnerServiceGetPartner).not.toHaveBeenCalled();
  });

  it('在途竞态防护：后选择的客户响应先返回时不被陈旧响应覆盖', async () => {
    let resolveFirst: any;
    const firstPromise = new Promise((resolve) => {
      resolveFirst = resolve;
    });
    const secondPromise = Promise.resolve({
      data: {
        id: 'cust-2',
        assignments: [
          {
            role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
            userId: 'sales-2',
            organizationId: 'org-sales-2',
          },
        ],
      },
    });

    vi.mocked(partnerServiceGetPartner)
      .mockImplementationOnce(() => firstPromise as any)
      .mockImplementationOnce(() => secondPromise as any);

    let formInstance: any;
    const TestComponent = () => {
      const [form] = Form.useForm();
      formInstance = form;
      return (
        <ProForm form={form} submitter={false}>
          <SeaCustomerField
            searchCustomers={vi.fn().mockResolvedValue([])}
            setCustomerCode={vi.fn()}
            personnelOptions={[]}
          />
        </ProForm>
      );
    };

    render(<TestComponent />);

    // 触发第一个客户选择
    capturedProps.onPartnerChange({ value: 'cust-1', code: 'C1' });
    // 快速切换到第二个客户
    capturedProps.onPartnerChange({ value: 'cust-2', code: 'C2' });

    // 等待第二个客户处理完毕
    await waitFor(() => {
      expect(formInstance.getFieldValue('salesUserId')).toBe('sales-2');
    });

    // 随后第一个请求才返回迟到的旧数据
    resolveFirst({
      data: {
        id: 'cust-1',
        assignments: [
          {
            role: PartnerAssignmentRole.PARTNER_ASSIGNMENT_ROLE_SALES,
            userId: 'sales-1-stale',
            organizationId: 'org-sales-1',
          },
        ],
      },
    });

    // 验证旧数据已被丢弃，依然保持销售人员 2
    await new Promise((r) => setTimeout(r, 20));
    expect(formInstance.getFieldValue('salesUserId')).toBe('sales-2');
  });
});
