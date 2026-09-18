import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDatePicker,
  ProFormDateRangePicker,
  ProFormDigit,
  ProFormSelect,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
  ProTable,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  DatePicker,
  Drawer,
  Form,
  Modal,
  Select,
  Table,
  Tag,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { FinanceOrganizationPurpose } from '@/enums.generated';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceAssignCommissionRuleEmployees,
  settlementServiceCopyCommissionRule,
  settlementServiceCreateCommissionRule,
  settlementServiceListCommissionEmployees,
  settlementServiceListCommissionRules,
  settlementServiceListFinanceOrganizationOptions,
  settlementServiceRemoveCommissionRuleEmployees,
  settlementServiceUpdateCommissionRule,
} from '@/services/roncin/settlementService';
import { toTableRequest, unwrapList } from '@/utils/api';
import { formatDate } from '@/utils/format';
import {
  calculationBasisMeta,
  calculationBasisText,
  decimalText,
  getBusinessReason,
  personnelRoleMeta,
  personnelRoleText,
  type RuleValues,
} from '../types';

type CommissionRulesDrawerProps = {
  open: boolean;
  onClose: () => void;
  canManage: boolean;
};

type EmployeeOption = { label: string; value: string };

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

/** CopyValues 复制为新方案表单值。 */
type CopyValues = {
  name: string;
  personnelRole: 'SALES' | 'OPERATOR' | 'CUSTOMER_SERVICE';
  calculationBasis: 'REALIZED_PROFIT' | 'REALIZED_REVENUE';
  ratePercent: number;
  effectiveFrom?: string;
  effectiveTo?: string;
  employeeIds?: string[];
  note?: string;
};

const toEmployeeOptions = (
  response: API.ListCommissionEmployeesResponse,
): EmployeeOption[] =>
  unwrapList(response).flatMap((item) =>
    item.id
      ? [
          {
            value: item.id,
            label: item.displayName ?? item.id,
          },
        ]
      : [],
  );

export default function CommissionRulesDrawer({
  open,
  onClose,
  canManage,
}: CommissionRulesDrawerProps) {
  const { message, modal } = App.useApp();
  const ruleActionRef = useRef<ActionType | undefined>(undefined);
  const ruleFormRef = useRef<ProFormInstance | undefined>(undefined);
  const [ruleFormOpen, setRuleFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<API.FinanceCommissionRule>();
  const [organizationOptions, setOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [readOrganizationOptions, setReadOrganizationOptions] = useState<
    API.FinanceOrganizationOption[]
  >([]);
  const [organizationId, setOrganizationId] = useState<string>();
  // 员工候选（新建表单 / 名单 / 复制共用，按各自组织刷新）。
  const [employeeOptions, setEmployeeOptions] = useState<EmployeeOption[]>([]);
  // 名单管理：当前操作的方案。
  const [rosterRule, setRosterRule] = useState<API.FinanceCommissionRule>();
  // 复制为新方案：源方案。
  const [copySourceRule, setCopySourceRule] =
    useState<API.FinanceCommissionRule>();
  // 列表「适用员工」过滤：远程搜索候选与选中值分离。
  const [employeeFilterOptions, setEmployeeFilterOptions] = useState<
    EmployeeOption[]
  >([]);
  const [employeeFilter, setEmployeeFilter] = useState<string>();

  useEffect(() => {
    if (!open || !canManage) return;
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_COMMISSION_MANAGE,
    })
      .then((response) => {
        if (!cancelled) setOrganizationOptions(response.data ?? []);
      })
      .catch(() => {
        if (!cancelled) message.warning('提成方案可管理公司候选加载失败');
      });
    return () => {
      cancelled = true;
    };
  }, [canManage, message, open]);
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    void settlementServiceListFinanceOrganizationOptions({
      purpose:
        FinanceOrganizationPurpose.FINANCE_ORGANIZATION_PURPOSE_COMMISSION_READ,
    })
      .then((response) => {
        if (!cancelled) setReadOrganizationOptions(response.data ?? []);
      })
      .catch(() => {
        if (!cancelled) message.warning('提成方案公司候选加载失败');
      });
    return () => {
      cancelled = true;
    };
  }, [message, open]);

  /** 指定组织内员工远程搜索：服务端关键字分页，pageSize 不超过 200。 */
  const searchEmployees = useCallback(
    async (
      keyword: string,
      targetOrganizationId: string | undefined,
      apply: (options: EmployeeOption[]) => void,
    ) => {
      try {
        const response = await settlementServiceListCommissionEmployees({
          page: 1,
          pageSize: 200,
          ...(targetOrganizationId
            ? { organizationId: targetOrganizationId }
            : {}),
          ...(keyword ? { keyword } : {}),
        });
        apply(toEmployeeOptions(response));
      } catch {
        apply([]);
      }
    },
    [],
  );

  const searchEmployeeFilter = useCallback(async (keyword = '') => {
    try {
      const response = await settlementServiceListCommissionEmployees({
        page: 1,
        pageSize: 200,
        ...(keyword ? { keyword } : {}),
      });
      setEmployeeFilterOptions(toEmployeeOptions(response));
    } catch {
      setEmployeeFilterOptions([]);
    }
  }, []);

  const openRoster = (record: API.FinanceCommissionRule) => {
    setRosterRule(record);
    setEmployeeOptions([]);
    void searchEmployees('', record.organizationId, setEmployeeOptions);
  };

  const handleRuleError = (error: any, fallback: string) => {
    const reason = getBusinessReason(error);
    if (reason === financeErrorReasons.FINANCE_COMMISSION_RULE_CONFLICT) {
      modal.warning({
        title: '方案已被其他人更新或名称冲突',
        content: '请关闭后重试；若为名称冲突请更换方案名称。',
      });
      return;
    }
    if (reason === 'FINANCE_COMMISSION_RULE_RETROACTIVE') {
      message.error('名单与区间变更只能选择当天或未来的生效日期');
      return;
    }
    if (reason === 'FINANCE_COMMISSION_RULE_INTERVAL_OVERLAP') {
      message.error('同一员工同一身份在重叠期间只能属于一个启用方案');
      return;
    }
    if (reason === 'FINANCE_COMMISSION_RULE_EMPLOYEE_INVALID') {
      message.error('所选员工不是当前组织的有效成员');
      return;
    }
    if (reason === 'FINANCE_COMMISSION_RULE_LOCKED_PARAMS') {
      message.error('方案已生效，核心参数不可修改，请复制为新方案');
      return;
    }
    message.error(error.message || fallback);
  };

  const ruleColumns: ProColumns<API.FinanceCommissionRule>[] = [
    {
      title: '所属公司',
      dataIndex: 'organizationName',
      width: 140,
      search: false,
      renderText: (value) => value || '-',
    },
    {
      title: '方案名称',
      dataIndex: 'name',
      copyable: true,
      render: (_, record) => (
        <span>
          {record.name}
          {record.legacyReadonly ? (
            <Tag style={{ marginLeft: 6 }} color="warning">
              历史只读
            </Tag>
          ) : null}
        </span>
      ),
    },
    {
      title: '适用角色',
      dataIndex: 'personnelRole',
      valueType: 'select',
      width: 100,
      valueEnum: personnelRoleMeta,
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      valueType: 'select',
      width: 110,
      valueEnum: calculationBasisMeta,
      renderText: (value) => calculationBasisText(value),
    },
    {
      title: '提成比例',
      dataIndex: 'ratePercent',
      align: 'right',
      width: 90,
      search: false,
      renderText: (value) => `${decimalText(value)}%`,
    },
    {
      title: '生效区间',
      key: 'effectiveRange',
      width: 170,
      search: false,
      render: (_, record) =>
        record.effectiveFrom || record.effectiveTo ? (
          <span>
            {record.effectiveFrom || '-'} ~ {record.effectiveTo || '-'}
          </span>
        ) : (
          '永久有效'
        ),
    },
    {
      title: '适用员工',
      dataIndex: 'activeEmployeeCount',
      width: 90,
      search: false,
      render: (_, record) => (
        <a
          onClick={() => {
            if (record.legacyReadonly) {
              message.info('迁移前历史旧规则没有员工分配，只能复制为新方案');
              return;
            }
            openRoster(record);
          }}
        >
          {record.activeEmployeeCount ?? 0} 人
        </a>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      valueType: 'select',
      width: 90,
      valueEnum: {
        true: { text: '已启用', status: 'Success' },
        false: { text: '已停用', status: 'Default' },
      },
      render: (_, record) => (
        <Tag color={record.enabled ? 'success' : 'default'}>
          {record.enabled ? '已启用' : '已停用'}
        </Tag>
      ),
    },
    {
      title: '版本',
      dataIndex: 'version',
      width: 60,
      search: false,
      renderText: (value) => `v${value || 0}`,
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 150,
      render: (_, record) =>
        canManage ? (
          <span>
            <a
              key="copy"
              onClick={() => {
                setCopySourceRule(record);
                setEmployeeOptions([]);
                void searchEmployees(
                  '',
                  record.organizationId,
                  setEmployeeOptions,
                );
              }}
            >
              复制为新方案
            </a>
            {!record.legacyReadonly && (
              <a
                key="edit"
                style={{ marginLeft: 8 }}
                onClick={() => {
                  setEditingRule(record);
                  setRuleFormOpen(true);
                }}
              >
                编辑
              </a>
            )}
          </span>
        ) : null,
    },
  ];

  const editingReached =
    Boolean(editingRule?.enabled) &&
    Boolean(editingRule?.effectiveFrom) &&
    (editingRule?.effectiveFrom ?? '') <= dayjs().format('YYYY-MM-DD');

  return (
    <>
      <Drawer title="提成方案" size={1080} open={open} onClose={onClose}>
        <div style={{ marginBottom: 12, display: 'flex', gap: 8 }}>
          <Select
            allowClear
            placeholder="所属公司"
            style={{ minWidth: 220 }}
            value={organizationId}
            options={readOrganizationOptions.map((item) => ({
              value: item.id,
              label: item.name ?? item.code ?? item.id,
            }))}
            onChange={(value) => {
              setOrganizationId(value);
              ruleActionRef.current?.reload();
            }}
          />
          <Select
            allowClear
            showSearch
            placeholder="适用员工"
            style={{ minWidth: 200 }}
            value={employeeFilter}
            options={employeeFilterOptions}
            filterOption={false}
            onSearch={(value) => void searchEmployeeFilter(value)}
            onDropdownVisibleChange={(visible) => {
              if (visible) void searchEmployeeFilter();
            }}
            onChange={(value) => {
              setEmployeeFilter(value);
              ruleActionRef.current?.reload();
            }}
          />
        </div>
        <ProTable<API.FinanceCommissionRule>
          actionRef={ruleActionRef}
          rowKey="id"
          columns={ruleColumns}
          bordered
          size="small"
          search={{ defaultCollapsed: false }}
          toolBarRender={() =>
            canManage
              ? [
                  <Button
                    key="new-rule"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingRule(undefined);
                      setEmployeeOptions([]);
                      setRuleFormOpen(true);
                    }}
                  >
                    新建方案
                  </Button>,
                ]
              : []
          }
          request={async (params) => {
            const response = await settlementServiceListCommissionRules({
              page: params.current ?? 1,
              pageSize: params.pageSize ?? 20,
              keyword: params.name,
              personnelRole: params.personnelRole,
              enabled: params.enabled,
              organizationId,
              ...(employeeFilter ? { employeeId: employeeFilter } : {}),
            });
            return toTableRequest(response);
          }}
        />
      </Drawer>

      <ModalForm<RuleValues>
        formRef={ruleFormRef}
        key={editingRule?.id || 'new-rule'}
        title={editingRule ? '编辑提成方案' : '新建提成方案'}
        open={ruleFormOpen}
        width={620}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setRuleFormOpen(false),
        }}
        initialValues={{
          name: editingRule?.name,
          personnelRole: editingRule?.personnelRole || 'SALES',
          calculationBasis: editingRule?.calculationBasis || 'REALIZED_PROFIT',
          ratePercent: editingRule
            ? Number(editingRule.ratePercent)
            : undefined,
          effectiveRange:
            editingRule?.effectiveFrom && editingRule?.effectiveTo
              ? [
                  dayjs(editingRule.effectiveFrom),
                  dayjs(editingRule.effectiveTo),
                ]
              : undefined,
          enabled: editingRule?.enabled ?? true,
          note: editingRule?.note,
          organizationId: editingRule?.organizationId,
        }}
        onFinish={async (values) => {
          // 方案核心参数：名单变更只走名单管理入口，不随本表单提交。
          const rule = {
            name: values.name,
            personnelRole: values.personnelRole,
            calculationBasis: values.calculationBasis,
            ratePercent: String(values.ratePercent),
            effectiveFrom: values.effectiveRange?.[0]?.format('YYYY-MM-DD'),
            effectiveTo: values.effectiveRange?.[1]?.format('YYYY-MM-DD'),
            enabled: values.enabled,
            note: values.note,
          };
          try {
            if (editingRule?.id && editingRule.version) {
              await settlementServiceUpdateCommissionRule(
                { id: editingRule.id },
                {
                  id: editingRule.id,
                  expectedVersion: editingRule.version,
                  rule,
                },
              );
              message.success('提成方案已更新');
            } else {
              const formOrganizationId = ruleFormRef.current?.getFieldValue(
                'organizationId',
              ) as string | undefined;
              if (!formOrganizationId) {
                message.error('请选择所属公司');
                return false;
              }
              if (values.enabled && !(values.employeeIds ?? []).length) {
                message.error('启用方案前必须至少分配一名员工');
                return false;
              }
              await settlementServiceCreateCommissionRule({
                organizationId: formOrganizationId,
                rule: { ...rule, employeeIds: values.employeeIds ?? [] },
              });
              message.success('提成方案已创建');
            }
            setRuleFormOpen(false);
            ruleActionRef.current?.reload();
            return true;
          } catch (error: any) {
            handleRuleError(error, '保存方案失败');
            return false;
          }
        }}
      >
        {editingRule?.legacyReadonly && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            title="迁移前历史旧规则为只读"
            description="该规则没有任何员工分配且已停用，不能编辑或重新启用；请复制为新方案并重新分配员工。"
          />
        )}
        {!editingRule && (
          <ProFormSearchableSelect
            name="organizationId"
            label="所属公司"
            rules={[{ required: true, message: '请选择所属公司' }]}
            options={organizationOptions.map((item) => ({
              value: item.id ?? '',
              label: item.name ?? item.code ?? item.id ?? '',
            }))}
            fieldProps={{
              onChange: (value: string) => {
                // 组织变化后清空员工候选并按新组织重载。
                setEmployeeOptions([]);
                if (value) {
                  void searchEmployees('', value, setEmployeeOptions);
                }
              },
            }}
          />
        )}
        <ProFormText
          name="name"
          label="方案名称"
          disabled={Boolean(editingRule?.legacyReadonly)}
          rules={[{ required: true, message: '请输入方案名称' }]}
        />
        <ProFormSearchableSelect
          name="personnelRole"
          label="提成人员身份"
          disabled={Boolean(editingRule)}
          rules={[{ required: true, message: '请选择人员身份' }]}
          options={[
            { value: 'SALES', label: '业务人员' },
            { value: 'OPERATOR', label: '操作人员' },
            { value: 'CUSTOMER_SERVICE', label: '客服人员' },
          ]}
          extra="人员身份表示适用员工按哪一种订单提成归属参与计算；方案生效后不可修改。"
        />
        <ProFormSearchableSelect
          name="calculationBasis"
          label="计提口径"
          disabled={Boolean(editingRule && editingReached)}
          rules={[{ required: true, message: '请选择计提口径' }]}
          options={[
            {
              value: 'REALIZED_PROFIT',
              label: '已实现毛利（推荐，按收入配比分摊成本）',
            },
            {
              value: 'REALIZED_REVENUE',
              label: '已实现收入（按来源单分摊收入全额）',
            },
          ]}
        />
        <ProFormDigit
          name="ratePercent"
          label="提成比例（%）"
          min={0.0001}
          max={100}
          fieldProps={{ precision: 4 }}
          disabled={Boolean(editingRule && editingReached)}
          rules={[{ required: true, message: '请输入提成比例' }]}
        />
        <ProFormDateRangePicker
          name="effectiveRange"
          label="生效区间"
          fieldProps={{
            disabled: [Boolean(editingRule && editingReached), false],
          }}
          extra={
            editingRule && editingReached
              ? '方案已生效：起始日不可修改，终止日只能选择当天或未来日期。'
              : '新启用方案的起始日必须是当天或未来日期。'
          }
        />
        <ProFormSwitch
          name="enabled"
          label="启用"
          disabled={Boolean(editingRule && editingReached)}
        />
        <ProFormTextArea
          name="note"
          label="方案说明"
          fieldProps={{ maxLength: 500 }}
          disabled={Boolean(editingRule?.legacyReadonly)}
        />
        {!editingRule && (
          <ProFormSelect
            name="employeeIds"
            label="适用员工"
            extra="同一批员工共享方案的身份、口径、比例与有效期；个别员工比例不同请另建方案。启用方案必须至少分配一名员工。"
            fieldProps={{
              mode: 'multiple',
              filterOption: false,
              onSearch: (value: string) => {
                void searchEmployees(
                  value,
                  ruleFormRef.current?.getFieldValue('organizationId') as
                    | string
                    | undefined,
                  setEmployeeOptions,
                );
              },
              onDropdownVisibleChange: (visible: boolean) => {
                if (visible) {
                  void searchEmployees(
                    '',
                    ruleFormRef.current?.getFieldValue('organizationId') as
                      | string
                      | undefined,
                    setEmployeeOptions,
                  );
                }
              },
            }}
            options={employeeOptions}
            placeholder="搜索当前组织员工"
          />
        )}
      </ModalForm>

      <Modal
        title={`名单管理 - ${rosterRule?.name ?? ''}`}
        open={Boolean(rosterRule)}
        width={760}
        destroyOnHidden
        footer={null}
        onCancel={() => setRosterRule(undefined)}
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
                  setEmployeeOptions,
                );
              }}
              onChanged={() => {
                ruleActionRef.current?.reload();
                setRosterRule(undefined);
              }}
              onError={handleRuleError}
            />
          </>
        )}
      </Modal>

      <ModalForm<CopyValues>
        key={copySourceRule?.id || 'copy-rule'}
        title="复制为新方案"
        open={Boolean(copySourceRule)}
        width={620}
        modalProps={{
          destroyOnHidden: true,
          onCancel: () => setCopySourceRule(undefined),
        }}
        initialValues={{
          name: copySourceRule ? `${copySourceRule.name}-新方案` : undefined,
          personnelRole:
            (copySourceRule?.personnelRole as CopyValues['personnelRole']) ||
            'SALES',
          calculationBasis:
            (copySourceRule?.calculationBasis as CopyValues['calculationBasis']) ||
            'REALIZED_PROFIT',
          ratePercent: copySourceRule
            ? Number(copySourceRule.ratePercent)
            : undefined,
          note: copySourceRule?.note,
          employeeIds: (copySourceRule?.assignments ?? [])
            .filter(
              (item) =>
                !item.effectiveTo ||
                item.effectiveTo >= dayjs().format('YYYY-MM-DD'),
            )
            .map((item) => item.employeeId ?? ''),
        }}
        onFinish={async (values) => {
          if (!copySourceRule?.id) return false;
          const effectiveFrom = values.effectiveFrom
            ? dayjs(values.effectiveFrom).format('YYYY-MM-DD')
            : '';
          if (!effectiveFrom) {
            message.error('请选择新方案生效起始日（当天或未来）');
            return false;
          }
          if (!(values.employeeIds ?? []).length) {
            message.error('新方案必须至少分配一名员工');
            return false;
          }
          try {
            await settlementServiceCopyCommissionRule(
              { id: copySourceRule.id },
              {
                id: copySourceRule.id,
                name: values.name,
                personnelRole: values.personnelRole,
                calculationBasis: values.calculationBasis,
                ratePercent: String(values.ratePercent),
                effectiveFrom,
                effectiveTo: values.effectiveTo
                  ? dayjs(values.effectiveTo).format('YYYY-MM-DD')
                  : undefined,
                employeeIds: values.employeeIds ?? [],
                note: values.note,
              },
            );
            message.success('已复制为新方案，原方案终止日已自动衔接');
            setCopySourceRule(undefined);
            ruleActionRef.current?.reload();
            return true;
          } catch (error: any) {
            handleRuleError(error, '复制方案失败');
            return false;
          }
        }}
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="原方案将在新方案生效日前一日自然结束"
          description="新方案必须以当天或未来日期生效；原方案的历史计算依据保持不变，来源日期在原方案期间内的提成继续按原方案计算。"
        />
        <ProFormText
          name="name"
          label="新方案名称"
          rules={[{ required: true, message: '请输入新方案名称' }]}
        />
        <ProFormSearchableSelect
          name="personnelRole"
          label="提成人员身份"
          rules={[{ required: true, message: '请选择人员身份' }]}
          options={[
            { value: 'SALES', label: '业务人员' },
            { value: 'OPERATOR', label: '操作人员' },
            { value: 'CUSTOMER_SERVICE', label: '客服人员' },
          ]}
        />
        <ProFormSearchableSelect
          name="calculationBasis"
          label="计提口径"
          rules={[{ required: true, message: '请选择计提口径' }]}
          options={[
            { value: 'REALIZED_PROFIT', label: '已实现毛利（推荐）' },
            { value: 'REALIZED_REVENUE', label: '已实现收入' },
          ]}
        />
        <ProFormDigit
          name="ratePercent"
          label="提成比例（%）"
          min={0.0001}
          max={100}
          fieldProps={{ precision: 4 }}
          rules={[{ required: true, message: '请输入提成比例' }]}
        />
        <ProFormDatePicker
          name="effectiveFrom"
          label="新方案生效起始日"
          fieldProps={{
            disabledDate: (current: Dayjs) =>
              Boolean(current?.isBefore(dayjs().startOf('day'))),
          }}
          rules={[{ required: true, message: '请选择生效起始日' }]}
          extra="必须为当天或未来日期。"
        />
        <ProFormDatePicker
          name="effectiveTo"
          label="新方案生效终止日（可选）"
        />
        <ProFormSelect
          name="employeeIds"
          label="适用员工"
          extra="默认带出原方案当前/未来名单，可调整。"
          fieldProps={{
            mode: 'multiple',
            filterOption: false,
            onSearch: (value: string) => {
              void searchEmployees(
                value,
                copySourceRule?.organizationId,
                setEmployeeOptions,
              );
            },
            onDropdownVisibleChange: (visible: boolean) => {
              if (visible) {
                void searchEmployees(
                  '',
                  copySourceRule?.organizationId,
                  setEmployeeOptions,
                );
              }
            },
          }}
          options={employeeOptions}
          placeholder="搜索当前组织员工"
        />
        <ProFormTextArea
          name="note"
          label="方案说明"
          fieldProps={{ maxLength: 500 }}
        />
      </ModalForm>
    </>
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
