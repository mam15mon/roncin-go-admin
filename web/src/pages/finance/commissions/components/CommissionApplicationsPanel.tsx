import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@/app/access';
import { App, DatePicker, Space } from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { SearchFilterTemplate } from '@/components/ui';
import { FinanceCommissionApplicationStatus } from '@/enums.generated';
import { getRequestErrorStatus } from '@/requestErrorConfig';
import {
  settlementServiceApproveCommissionApplication,
  settlementServiceListCommissionApplications,
  settlementServiceListCommissionEmployees,
  settlementServiceRejectCommissionApplication,
} from '@/services/roncin/settlementService';
import { toTableRequest } from '@/utils/api';
import { confirmWithReason } from '@/utils/confirmWithReason';
import { decimalText } from '../types';
import {
  APPLICATION_STATUS_FILTER_OPTIONS,
  applicationStatusTag,
} from './applicationStatus';
import CommissionApplicationDetailDrawer from './CommissionApplicationDetailDrawer';

const { MonthPicker } = DatePicker;

type Application = API.FinanceCommissionApplication;

type ApplicationFilterValues = {
  employeeId?: string;
  status?: number;
  /** MonthPicker 的表单值：选择后为 Dayjs，提交前统一格式化为 YYYY-MM。 */
  applicationMonth?: Dayjs | string;
};

/** 已提交过滤条件：MonthPicker 值已格式化为 YYYY-MM 字符串，不再含 Dayjs。 */
type CommittedApplicationFilters = {
  employeeId?: string;
  status?: number;
  applicationMonth?: string;
};

/** 版本冲突 / 并发处理：409 统一按「已被处理，请刷新」提示。 */
export function isApplicationConflictError(error: unknown): boolean {
  return getRequestErrorStatus(error) === 409;
}

/**
 * 月度提成申请批次面板：每行一名员工的一张申请批次，支持状态/提交月/员工过滤。
 * 首期只提供整单批准与整单驳回（驳回原因必填），不提供任何部分批准、
 * 剔除明细或拆分入口；批准只确认提成计算结果，不代表任何付款行为。
 */
export default function CommissionApplicationsPanel() {
  const access = useAccess();
  const { message, modal } = App.useApp();
  const actionRef = useRef<ActionType | undefined>(undefined);
  const committedFiltersRef = useRef<CommittedApplicationFilters>({});
  const [employeeOptions, setEmployeeOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const [detailId, setDetailId] = useState<string>();
  const [detailRefreshToken, setDetailRefreshToken] = useState(0);

  const canManage = access.canManageFinanceCommissions === true;

  useEffect(() => {
    // 员工过滤候选项沿用提成台账先例：单页 200 条服务端员工选项。
    settlementServiceListCommissionEmployees({ page: 1, pageSize: 200 })
      .then((response) => {
        setEmployeeOptions(
          (response.data ?? [])
            .filter((item) => item.id)
            .map((item) => ({
              label: item.displayName || (item.id as string),
              value: item.id as string,
            })),
        );
      })
      .catch(() => setEmployeeOptions([]));
  }, []);

  const reload = () => actionRef.current?.reload();

  const refreshDetail = () => setDetailRefreshToken((token) => token + 1);

  const handleDecisionError = (error: unknown, fallback: string) => {
    if (isApplicationConflictError(error)) {
      message.warning('该申请已被处理，请刷新后查看最新状态');
      reload();
      refreshDetail();
      return;
    }
    message.error((error as Error).message || fallback);
  };

  const approve = (record: Application) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    modal.confirm({
      title: `批准 ${record.employeeName || '员工'} ${record.applicationMonth || ''} 月度申请？`,
      content: (
        <div>
          <p>
            {`整单共 ${record.commissionCount ?? 0} 笔、本位币合计 ${decimalText(record.totalCommissionAmount)} ${record.baseCurrency || ''}；批准将整批确认申请内全部提成，不支持部分批准。`}
          </p>
          <p style={{ color: '#faad14' }}>
            任一明细有问题时应整单驳回；批准只确认提成计算结果。
          </p>
        </div>
      ),
      okText: '整单批准',
      onOk: async () => {
        try {
          await settlementServiceApproveCommissionApplication(
            { id },
            { id, expectedVersion: version },
          );
          message.success('申请已整单批准');
          reload();
          setDetailId(undefined);
        } catch (error: unknown) {
          handleDecisionError(error, '批准失败');
        }
      },
    });
  };

  const reject = (record: Application) => {
    if (!record.id || !record.version) return;
    const id = record.id;
    const version = record.version;
    confirmWithReason(
      { modal, message },
      `驳回 ${record.employeeName || '员工'} ${record.applicationMonth || ''} 月度申请？`,
      async (reason) => {
        try {
          await settlementServiceRejectCommissionApplication(
            { id },
            { id, expectedVersion: version, reason },
          );
          message.success('申请已整单驳回，员工可在原申请上重新提交');
          reload();
          setDetailId(undefined);
        } catch (error: unknown) {
          handleDecisionError(error, '驳回失败');
        }
      },
      {
        danger: true,
        placeholder: '请输入驳回原因（必填）',
        requiredMessage: '请输入驳回原因',
      },
    );
  };

  const columns: ProColumns<Application>[] = [
    {
      title: '员工',
      dataIndex: 'employeeName',
      width: 110,
      renderText: (value) => value || '-',
    },
    {
      title: '申请月份',
      dataIndex: 'applicationMonth',
      width: 100,
      renderText: (value) => value || '-',
    },
    {
      title: '覆盖截止日',
      dataIndex: 'coverageTo',
      width: 110,
      renderText: (value) => value || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, record) => applicationStatusTag(record.status),
    },
    {
      title: '笔数',
      dataIndex: 'commissionCount',
      width: 70,
      align: 'right',
      renderText: (value) => value ?? 0,
    },
    {
      title: '本位币总额',
      dataIndex: 'totalCommissionAmount',
      width: 150,
      align: 'right',
      render: (_, record) => (
        <strong>
          {`${decimalText(record.totalCommissionAmount)} ${record.baseCurrency || ''}`}
        </strong>
      ),
    },
    {
      title: '提交时间',
      dataIndex: 'submittedAt',
      width: 150,
      renderText: (value) =>
        value ? value.slice(0, 16).replace('T', ' ') : '-',
    },
    {
      title: '操作',
      valueType: 'option',
      fixed: 'right',
      width: 170,
      render: (_, record) => {
        const actions: React.ReactNode[] = [
          <a key="detail" onClick={() => setDetailId(record.id)}>
            明细
          </a>,
        ];
        if (
          canManage &&
          access.canOperateOrganization(record.organizationId) &&
          record.status ===
            FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW
        ) {
          actions.push(
            <a key="approve" onClick={() => approve(record)}>
              整单批准
            </a>,
            <a key="reject" onClick={() => reject(record)}>
              整单驳回
            </a>,
          );
        }
        return actions;
      },
    },
  ];

  return (
    <>
      <SearchFilterTemplate<ApplicationFilterValues>
        layout="grid"
        collapsible={false}
        colSpan={6}
        items={[
          {
            name: 'status',
            label: '状态',
            type: 'select',
            placeholder: '待审批（默认）',
            span: 6,
            options: APPLICATION_STATUS_FILTER_OPTIONS,
          },
          {
            name: 'applicationMonth',
            label: '提交月份',
            type: 'custom',
            span: 6,
            render: () => (
              <MonthPicker
                allowClear
                placeholder="选择提交月份"
                style={{ width: '100%' }}
              />
            ),
          },
          {
            name: 'employeeId',
            label: '员工',
            type: 'searchable-select',
            placeholder: '全部员工',
            span: 6,
            options: employeeOptions,
          },
        ]}
        extraRight={
          <Space size={8}>
            <a onClick={reload}>刷新</a>
          </Space>
        }
        onSearch={(values) => {
          // MonthPicker 表单值是 Dayjs：服务端 application_month 只接受
          // YYYY-MM 字符串，提交前统一格式化（与资金流水日期提交同一范式）。
          const rawMonth = values.applicationMonth;
          committedFiltersRef.current = {
            status:
              values.status === undefined || values.status === null
                ? undefined
                : Number(values.status),
            applicationMonth: rawMonth
              ? dayjs(rawMonth).format('YYYY-MM')
              : undefined,
            employeeId: values.employeeId || undefined,
          };
          reload();
        }}
        onReset={() => {
          committedFiltersRef.current = {};
          reload();
        }}
      />
      <ProTable<Application>
        headerTitle="月度提成申请批次"
        actionRef={actionRef}
        rowKey="id"
        columns={columns}
        size="small"
        search={false}
        toolBarRender={false}
        cardProps={{
          style: { borderRadius: 8, border: '1px solid #f0f0f0' },
        }}
        request={async (params) => {
          const filters = committedFiltersRef.current;
          const response = await settlementServiceListCommissionApplications({
            page: params.current ?? 1,
            pageSize: params.pageSize ?? 20,
            // 默认聚焦待审批队列；显式选择状态后按选择过滤。
            status:
              filters.status ??
              FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW,
            applicationMonth: filters.applicationMonth,
            employeeId: filters.employeeId,
          });
          return toTableRequest(response);
        }}
      />
      <CommissionApplicationDetailDrawer
        open={detailId !== undefined}
        applicationId={detailId}
        canManage={canManage}
        refreshToken={detailRefreshToken}
        onClose={() => setDetailId(undefined)}
        onApprove={approve}
        onReject={reject}
      />
    </>
  );
}
