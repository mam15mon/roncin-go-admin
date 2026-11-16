import { PlusOutlined } from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { App, Button, Drawer, Select, Tag } from 'antd';
import dayjs from 'dayjs';
import React, { useMemo, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { useAccess } from '@/app/access';
import { DRAWER_SIZE, ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { defaultSelectFilterOption } from '@/components/ui/searchable-select/utils';
import { financeErrorReasons } from '@/errorReasons.generated';
import {
  settlementServiceCopyCommissionRule,
  settlementServiceCreateCommissionRule,
  settlementServiceListCommissionEmployees,
  settlementServiceListCommissionRules,
  settlementServiceUpdateCommissionRule,
} from '@/services/roncin/settlementService';
import {
  ensureListResponse,
  isWriteResponse,
  toTableRequest,
} from '@/utils/api';
import { getErrorMessage } from '@/utils/errorMessage';
import { trimDecimal } from '@/utils/format';
import {
  calculationBasisMeta,
  calculationBasisText,
  getBusinessReason,
  personnelRoleMeta,
  personnelRoleText,
  type RuleValues,
} from '../types';
import CommissionRuleCopyModal, {
  type CopyValues,
} from './CommissionRuleCopyModal';
import CommissionRuleFormModal from './CommissionRuleFormModal';
import CommissionRuleRosterModal from './CommissionRuleRosterModal';
import { type EmployeeOption, toEmployeeOptions } from './commissionRuleShared';

type CommissionRulesDrawerProps = {
  open: boolean;
  onClose: () => void;
  canManage: boolean;
};

export default function CommissionRulesDrawer({
  open,
  onClose,
  canManage,
}: CommissionRulesDrawerProps) {
  const { message, modal } = App.useApp();
  const access = useAccess();
  const { initialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  const organizationId = currentUser?.currentOrganization?.id;
  const hasCompanyScope = Boolean(
    currentUser?.id && access.canOperateOrganization(organizationId),
  );
  const canRead = hasCompanyScope && access.canReadFinanceCommissions;
  const canManageCurrent =
    hasCompanyScope && canManage && access.canConfigureFinanceCommissions;
  const ruleActionRef = useRef<ActionType | undefined>(undefined);
  const ruleFormRef = useRef<ProFormInstance | undefined>(undefined);
  const [ruleFormOpen, setRuleFormOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<API.FinanceCommissionRule>();
  // 员工候选（新建表单 / 名单 / 复制共用，固定当前公司）。
  const [employeeKeyword, setEmployeeKeyword] = useState('');
  const [selectedEmployeeIds, setSelectedEmployeeIds] = useState<string[]>([]);
  const queryClient = useQueryClient();
  // 名单管理：当前操作的方案。
  const [rosterRule, setRosterRule] = useState<API.FinanceCommissionRule>();
  // 复制为新方案：源方案。
  const [copySourceRule, setCopySourceRule] =
    useState<API.FinanceCommissionRule>();
  // 列表「适用员工」过滤：远程搜索候选与选中值分离。
  const [employeeFilterKeyword, setEmployeeFilterKeyword] = useState('');
  const [filterEmployeesOpen, setFilterEmployeesOpen] = useState(false);
  const [employeeFilter, setEmployeeFilter] = useState<string>();
  const employeeQuery = useQuery({
    queryKey: [
      'finance',
      'commission-rule-employees',
      {
        organizationId,
        keyword: employeeKeyword,
        active:
          open &&
          canRead &&
          Boolean(ruleFormOpen || rosterRule || copySourceRule),
      },
    ],
    enabled:
      open && canRead && Boolean(ruleFormOpen || rosterRule || copySourceRule),
    meta: { errorMessage: '员工候选加载失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceListCommissionEmployees(
        {
          page: 1,
          pageSize: 200,
          organizationId,
          keyword: employeeKeyword || undefined,
        },
        { signal, skipErrorHandler: true },
      );
      if (!response || response.success === false)
        throw new Error('员工候选加载失败');
      return toEmployeeOptions(
        ensureListResponse(response, '员工候选加载失败'),
      );
    },
  });
  const filterEmployeeQuery = useQuery({
    queryKey: [
      'finance',
      'commission-rule-employees',
      {
        organizationId,
        keyword: employeeFilterKeyword,
        active: open && canRead && filterEmployeesOpen,
      },
    ],
    enabled: open && canRead && filterEmployeesOpen,
    meta: { errorMessage: '员工候选加载失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceListCommissionEmployees(
        {
          page: 1,
          pageSize: 200,
          organizationId,
          keyword: employeeFilterKeyword || undefined,
        },
        { signal, skipErrorHandler: true },
      );
      if (!response || response.success === false)
        throw new Error('员工候选加载失败');
      return toEmployeeOptions(
        ensureListResponse(response, '员工候选加载失败'),
      );
    },
  });
  const cachedEmployees = queryClient
    .getQueriesData<EmployeeOption[]>({
      queryKey: ['finance', 'commission-rule-employees'],
      predicate: (query) =>
        (query.queryKey[2] as { organizationId?: string } | undefined)
          ?.organizationId === organizationId,
    })
    .flatMap(([, options]) => options ?? []);
  const employeeOptions = useMemo(() => {
    const selected = new Set(selectedEmployeeIds);
    const preserved = cachedEmployees.filter((option) =>
      selected.has(option.value),
    );
    const assignments = [
      ...(copySourceRule?.assignments ?? []),
      ...(rosterRule?.assignments ?? []),
    ].flatMap((item) =>
      item.employeeId
        ? [
            {
              value: item.employeeId,
              label: item.employeeName || item.employeeId,
            },
          ]
        : [],
    );
    return [
      ...new Map(
        [...preserved, ...assignments, ...(employeeQuery.data ?? [])].map(
          (option) => [option.value, option],
        ),
      ).values(),
    ];
  }, [
    cachedEmployees,
    copySourceRule?.assignments,
    employeeQuery.data,
    rosterRule?.assignments,
    selectedEmployeeIds,
  ]);
  const employeeFilterOptions = [
    ...new Map(
      [
        ...cachedEmployees.filter((option) => option.value === employeeFilter),
        ...(filterEmployeeQuery.data ?? []),
      ].map((option) => [option.value, option]),
    ).values(),
  ];
  const searchEmployees = (keyword = '') => setEmployeeKeyword(keyword.trim());
  const searchEmployeeFilter = (keyword = '') => {
    setFilterEmployeesOpen(true);
    setEmployeeFilterKeyword(keyword.trim());
  };

  const requireCurrentRule = (record: API.FinanceCommissionRule) => {
    if (
      !hasCompanyScope ||
      !access.canOperateOrganization(record.organizationId)
    ) {
      message.warning('只能操作当前公司工作区的提成方案');
      return false;
    }
    return true;
  };

  const openRoster = (record: API.FinanceCommissionRule) => {
    if (!canRead || !requireCurrentRule(record)) return;
    setRosterRule(record);
    setEmployeeKeyword('');
    setSelectedEmployeeIds([]);
    searchEmployees();
  };

  const handleRuleError = (error: unknown, fallback: string) => {
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
    message.error(getErrorMessage(error, fallback));
  };

  // 方案核心参数：名单变更只走名单管理入口，不随本表单提交。
  const handleRuleSubmit = async (values: RuleValues) => {
    if (!canManageCurrent || !organizationId) {
      message.warning('请切换到有方案维护权限的公司工作区');
      return false;
    }
    if (editingRule && !requireCurrentRule(editingRule)) return false;
    try {
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
      if (editingRule?.id && editingRule.version) {
        const writeResponse1 = await settlementServiceUpdateCommissionRule(
          { id: editingRule.id },
          {
            id: editingRule.id,
            expectedVersion: editingRule.version,
            rule,
          },
        );
        if (!isWriteResponse(writeResponse1, true)) return false;
        message.success('提成方案已更新');
      } else {
        if (values.enabled && !(values.employeeIds ?? []).length) {
          message.error('启用方案前必须至少分配一名员工');
          return false;
        }
        const writeResponse2 = await settlementServiceCreateCommissionRule({
          organizationId,
          rule: { ...rule, employeeIds: values.employeeIds ?? [] },
        });
        if (!isWriteResponse(writeResponse2, true)) return false;
        message.success('提成方案已创建');
      }
      setRuleFormOpen(false);
      ruleActionRef.current?.reload();
      return true;
    } catch (error) {
      handleRuleError(error, '保存方案失败');
      return false;
    }
  };

  const handleCopySubmit = async (values: CopyValues) => {
    if (
      !copySourceRule?.id ||
      !canManageCurrent ||
      !requireCurrentRule(copySourceRule)
    )
      return false;
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
      const writeResponse3 = await settlementServiceCopyCommissionRule(
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
      if (!isWriteResponse(writeResponse3, true)) return false;
      message.success('已复制为新方案，原方案终止日已自动衔接');
      setCopySourceRule(undefined);
      ruleActionRef.current?.reload();
      return true;
    } catch (error) {
      handleRuleError(error, '复制方案失败');
      return false;
    }
  };

  const ruleColumns: ProColumns<API.FinanceCommissionRule>[] = [
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
      fieldProps: { filterOption: defaultSelectFilterOption },
      width: 100,
      valueEnum: personnelRoleMeta,
      renderText: (value) => personnelRoleText(value),
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      valueType: 'select',
      fieldProps: { filterOption: defaultSelectFilterOption },
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
      renderText: (value) => `${trimDecimal(value)}%`,
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
      fieldProps: { filterOption: defaultSelectFilterOption },
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
        canManageCurrent ? (
          <span>
            <a
              key="copy"
              onClick={() => {
                if (!requireCurrentRule(record)) return;
                setCopySourceRule(record);
                setEmployeeKeyword('');
                setSelectedEmployeeIds([]);
                searchEmployees();
              }}
            >
              复制为新方案
            </a>
            {!record.legacyReadonly && (
              <a
                key="edit"
                style={{ marginLeft: 8 }}
                onClick={() => {
                  if (!requireCurrentRule(record)) return;
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

  const ruleColumnSettings = useColumnSettings<
    ProColumns<API.FinanceCommissionRule>
  >({
    tableKey: 'finance:commission-rules',
    columns: ruleColumns,
  });

  return (
    <>
      <Drawer
        title="提成方案"
        size={DRAWER_SIZE.LG}
        open={open}
        onClose={onClose}
      >
        <div style={{ marginBottom: 12, display: 'flex', gap: 8 }}>
          <Select
            allowClear
            showSearch={{
              filterOption: false,
              onSearch: (value) => void searchEmployeeFilter(value),
            }}
            placeholder="适用员工"
            style={{ minWidth: 200 }}
            value={employeeFilter}
            options={employeeFilterOptions}
            onOpenChange={(visible) => {
              setFilterEmployeesOpen(visible);
              if (visible) searchEmployeeFilter();
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
          columns={ruleColumnSettings.columns}
          bordered
          size="small"
          search={{ defaultCollapsed: false }}
          options={{ reload: true, density: true, setting: false }}
          toolBarRender={() => [
            ruleColumnSettings.entry,
            ...(canManageCurrent
              ? [
                  <Button
                    key="new-rule"
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingRule(undefined);
                      setEmployeeKeyword('');
                      setSelectedEmployeeIds([]);
                      setRuleFormOpen(true);
                      searchEmployees();
                    }}
                  >
                    新建方案
                  </Button>,
                ]
              : []),
          ]}
          request={async (params) => {
            if (!canRead || !open || !organizationId)
              return { data: [], success: false, total: 0 };
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

      <CommissionRuleFormModal
        open={ruleFormOpen}
        editingRule={editingRule}
        formRef={ruleFormRef}
        employeeOptions={employeeOptions}
        searchEmployees={searchEmployees}
        onEmployeeSelectionChange={setSelectedEmployeeIds}
        onCancel={() => setRuleFormOpen(false)}
        onFinish={handleRuleSubmit}
      />

      <CommissionRuleRosterModal
        rosterRule={rosterRule}
        employeeOptions={employeeOptions}
        searchEmployees={searchEmployees}
        onEmployeeSelectionChange={setSelectedEmployeeIds}
        onCancel={() => setRosterRule(undefined)}
        onChanged={() => {
          ruleActionRef.current?.reload();
          setRosterRule(undefined);
        }}
        onError={handleRuleError}
      />

      <CommissionRuleCopyModal
        copySourceRule={copySourceRule}
        employeeOptions={employeeOptions}
        searchEmployees={searchEmployees}
        onEmployeeSelectionChange={setSelectedEmployeeIds}
        onCancel={() => setCopySourceRule(undefined)}
        onFinish={handleCopySubmit}
      />
    </>
  );
}
