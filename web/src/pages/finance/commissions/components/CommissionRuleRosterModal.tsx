import { App, Button, DatePicker, Form, Modal, Select, Table, Tag } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React from 'react';
import {
  settlementServiceAssignCommissionRuleEmployees,
  settlementServiceRemoveCommissionRuleEmployees,
} from '@/services/roncin/settlementService';
import { formatDate } from '@/utils/format';
import type { EmployeeOption } from './commissionRuleShared';

/** 名单段状态：按变更日期与分配区间判定，仅用于展示。 */
const assignmentStatus = (record: API.CommissionRuleAssignmentProjection) => {
  const today = dayjs().format('YYYY-MM-DD');
  if (record.effectiveTo && record.effectiveTo < today) {
    return { text: '已终止', color: 'default' };
  }
  if (record.effectiveFrom && record.effectiveFrom > today) {
    return { text: '未生效', color: 'processing' };
  }
  return { text: '生效中', color: 'success' };
};

type CommissionRuleRosterModalProps = {
  rosterRule: API.FinanceCommissionRule | undefined;
  employeeOptions: EmployeeOption[];
  searchEmployees: (
    keyword: string,
    targetOrganizationId: string | undefined,
    apply: (options: EmployeeOption[]) => void,
  ) => Promise<void>;
  onEmployeeOptionsChange: React.Dispatch<
    React.SetStateAction<EmployeeOption[]>
  >;
  onCancel: () => void;
  onChanged: () => void;
  onError: (error: any, fallback: string) => void;
};

/** 提成方案名单管理弹窗：当前名单展示 + 批量加入 / 移除表单 */
export default function CommissionRuleRosterModal({
  rosterRule,
  employeeOptions,
  searchEmployees,
  onEmployeeOptionsChange,
  onCancel,
  onChanged,
  onError,
}: CommissionRuleRosterModalProps) {
  return (
    <Modal
      title={`名单管理 - ${rosterRule?.name ?? ''}`}
      open={Boolean(rosterRule)}
      width={760}
      destroyOnHidden
      footer={null}
      onCancel={onCancel}
    >
      {rosterRule && (
        <>
          <Table<API.CommissionRuleAssignmentProjection>
            size="small"
            bordered
            rowKey="id"
            pagination={false}
            dataSource={rosterRule.assignments ?? []}
            columns={[
              {
                title: '员工',
                dataIndex: 'employeeName',
                render: (_, record) => record.employeeName || '-',
              },
              {
                title: '有效区间',
                key: 'range',
                render: (_, record) =>
                  `${record.effectiveFrom || '-'} ~ ${record.effectiveTo || '长期'}`,
              },
              {
                title: '状态',
                key: 'status',
                width: 90,
                render: (_, record) => {
                  const status = assignmentStatus(record);
                  return <Tag color={status.color}>{status.text}</Tag>;
                },
              },
              {
                title: '记录时间',
                key: 'recordedAt',
                width: 110,
                render: (_, record) => formatDate(record.effectiveFrom),
              },
            ]}
            locale={{ emptyText: '暂无员工分配' }}
            style={{ marginBottom: 16 }}
          />
          <RosterChangeForms
            rosterRule={rosterRule}
            employeeOptions={employeeOptions}
            searchEmployees={(keyword) => {
              void searchEmployees(
                keyword ?? '',
                rosterRule.organizationId,
                onEmployeeOptionsChange,
              );
            }}
            onChanged={onChanged}
            onError={onError}
          />
        </>
      )}
    </Modal>
  );
}

type RosterChangeFormsProps = {
  rosterRule: API.FinanceCommissionRule;
  employeeOptions: EmployeeOption[];
  searchEmployees: (keyword?: string) => void;
  onChanged: () => void;
  onError: (error: any, fallback: string) => void;
};

type RosterChangeValues = {
  employeeIds?: string[];
  changeDate?: { format: (format: string) => string };
};

/** RosterChangeForms 名单批量加入与移除表单：默认按当天生效，禁止回溯日期。 */
function RosterChangeForms({
  rosterRule,
  employeeOptions,
  searchEmployees,
  onChanged,
  onError,
}: RosterChangeFormsProps) {
  const { message } = App.useApp();
  const [addForm] = Form.useForm<RosterChangeValues>();
  const [removeForm] = Form.useForm<RosterChangeValues>();
  const today = dayjs().format('YYYY-MM-DD');
  const currentEmployees = (rosterRule.assignments ?? [])
    .filter((item) => !item.effectiveTo)
    .map((item) => ({
      value: item.employeeId ?? '',
      label: item.employeeName || item.employeeId || '',
    }));

  const submit = async (
    values: RosterChangeValues,
    action: 'assign' | 'remove',
  ) => {
    if (!values.employeeIds?.length) {
      message.warning(
        action === 'assign' ? '请选择要加入的员工' : '请选择要移除的员工',
      );
      return;
    }
    const change = {
      employeeIds: values.employeeIds,
      expectedVersion: String(rosterRule.version ?? 0),
      changeEffectiveDate: values.changeDate?.format('YYYY-MM-DD') ?? today,
    };
    try {
      if (action === 'assign') {
        await settlementServiceAssignCommissionRuleEmployees(
          { id: rosterRule.id ?? '' },
          { id: rosterRule.id ?? '', change },
        );
        message.success('员工已加入方案名单');
      } else {
        await settlementServiceRemoveCommissionRuleEmployees(
          { id: rosterRule.id ?? '' },
          { id: rosterRule.id ?? '', change },
        );
        message.success('员工已按变更生效日退出方案名单');
      }
      if (action === 'assign') {
        addForm.resetFields();
      } else {
        removeForm.resetFields();
      }
      onChanged();
    } catch (error: any) {
      onError(error, action === 'assign' ? '加入名单失败' : '移除名单失败');
    }
  };

  return (
    <div style={{ display: 'grid', gap: 16 }}>
      <Form
        form={addForm}
        layout="inline"
        onFinish={(values) => void submit(values, 'assign')}
      >
        <Form.Item name="employeeIds" label="加入员工">
          <Select
            mode="multiple"
            showSearch
            placeholder="搜索当前组织员工"
            style={{ minWidth: 260 }}
            options={employeeOptions}
            filterOption={false}
            onSearch={(value) => searchEmployees(value)}
            onDropdownVisibleChange={(visible) => {
              if (visible) searchEmployees();
            }}
          />
        </Form.Item>
        <Form.Item name="changeDate" label="生效日">
          <DatePicker
            disabledDate={(current: Dayjs) =>
              Boolean(current?.isBefore(dayjs().startOf('day')))
            }
          />
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit">
            加入名单
          </Button>
        </Form.Item>
      </Form>
      <Form
        form={removeForm}
        layout="inline"
        onFinish={(values) => void submit(values, 'remove')}
      >
        <Form.Item name="employeeIds" label="移除员工">
          <Select
            mode="multiple"
            placeholder="选择当前名单内员工"
            style={{ minWidth: 260 }}
            options={currentEmployees}
          />
        </Form.Item>
        <Form.Item name="changeDate" label="退出日">
          <DatePicker
            disabledDate={(current: Dayjs) =>
              Boolean(current?.isBefore(dayjs().startOf('day')))
            }
          />
        </Form.Item>
        <Form.Item>
          <Button danger htmlType="submit">
            移出名单
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
}
