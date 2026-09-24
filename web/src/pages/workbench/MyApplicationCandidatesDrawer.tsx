import { keepPreviousData, useQuery } from '@tanstack/react-query';
import type { TableColumnsType } from 'antd';
import { Drawer, Select, Space, Table } from 'antd';
import React, { useState } from 'react';
import { useColumnSettings } from '@/components/ui/column-settings';
import { workbenchServiceListMyApplicationCandidates } from '@/services/roncin/workbenchService';
import { formatDate } from '@/utils/format';
import {
  amountWithCurrency,
  workbenchCalculationBasisText,
  workbenchPersonnelRoleText,
  workbenchSourceNo,
} from './display';

type Candidate = API.WorkbenchApplicationCandidate;

type CandidatesQuery = {
  page: number;
  pageSize: number;
  commissionMonth?: string;
};

const DEFAULT_PAGE_SIZE = 20;

/** 服务端状态域前缀：可申请提成候选下钻查询的统一 key 前缀。 */
const CANDIDATES_QUERY_BASE = ['workbench', 'application-candidates'] as const;

type Props = {
  open: boolean;
  baseCurrency?: string;
  /** 归属月过滤候选项：来自 Overview 可申请分组，服务端仍按归属月过滤分页。 */
  monthOptions: { label: string; value: string }[];
  onClose: () => void;
};

/**
 * 可申请提成候选明细下钻抽屉：服务端分页 + 归属月过滤。
 * 候选由服务端按现有计提口径全量解析，页面只读，不提供任何提交或挑选能力。
 */
/** 候选投影只有归属日期；归属月仅取 YYYY-MM 前缀用于展示。 */
const monthOf = (record: Candidate) =>
  record.commissionDate?.slice(0, 7) || '-';

export default function MyApplicationCandidatesDrawer({
  open,
  baseCurrency,
  monthOptions,
  onClose,
}: Props) {
  const [query, setQuery] = useState<CandidatesQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });

  const { data, isFetching } = useQuery({
    queryKey: [
      ...CANDIDATES_QUERY_BASE,
      {
        page: query.page,
        pageSize: query.pageSize,
        commissionMonth: query.commissionMonth,
      },
    ],
    queryFn: () =>
      workbenchServiceListMyApplicationCandidates({
        page: query.page,
        pageSize: query.pageSize,
        ...(query.commissionMonth
          ? { commissionMonth: query.commissionMonth }
          : {}),
      }),
    enabled: open,
    // 旧行为为空 catch 静默，仅靠请求层 notification；显式声明避免全局 message 补充弹错
    meta: { silent: true },
    // 翻页与切换归属月期间保留当前内容，与既有手写层行为一致。
    placeholderData: keepPreviousData,
  });

  const items = data?.data ?? [];
  const total = Number(data?.total ?? 0);

  const currency = baseCurrency || undefined;

  const columns: TableColumnsType<Candidate> = [
    {
      title: '归属日期',
      dataIndex: 'commissionDate',
      width: 110,
      render: (_, record) => formatDate(record.commissionDate, 'date'),
    },
    {
      title: '归属月',
      dataIndex: 'commissionMonth',
      width: 90,
      render: (_, record) => monthOf(record),
    },
    {
      title: '来源单号',
      dataIndex: 'sourceNo',
      width: 150,
      render: (_, record) => workbenchSourceNo(record),
    },
    {
      title: '人员身份',
      dataIndex: 'personnelRole',
      width: 100,
      render: (_, record) => workbenchPersonnelRoleText(record.personnelRole),
    },
    {
      title: '提成方案',
      dataIndex: 'ruleName',
      width: 160,
      render: (_, record) => record.ruleName || '-',
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      width: 110,
      render: (_, record) =>
        workbenchCalculationBasisText(record.calculationBasis),
    },
    {
      title: '提成金额',
      dataIndex: 'commissionAmount',
      width: 140,
      align: 'right',
      render: (_, record) =>
        amountWithCurrency(
          record.commissionAmount,
          record.baseCurrency || currency,
        ),
    },
  ];

  const columnSettings = useColumnSettings<TableColumnsType<Candidate>[number]>({
    tableKey: 'workbench:application-candidates',
    columns,
  });

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="可申请提成明细"
      size={920}
      destroyOnHidden
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <Space size={8}>
            <span style={{ fontSize: 13 }}>归属月</span>
            <Select
              allowClear
              placeholder="全部可申请月份"
              style={{ width: 160 }}
              value={query.commissionMonth}
              options={monthOptions}
              onChange={(value) =>
                setQuery((prev) => ({
                  ...prev,
                  page: 1,
                  commissionMonth: value,
                }))
              }
            />
          </Space>
          <span style={{ fontSize: 12, color: '#64748b' }}>
            展示截至上一自然月末、尚未进入任何申请的合格提成。
          </span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
          {columnSettings.entry}
        </div>
        <Table<Candidate>
          rowKey={(record) =>
            record.verificationId ||
            record.nettingId ||
            record.commissionDate ||
            ''
          }
          size="small"
          loading={isFetching}
          columns={columnSettings.columns}
          dataSource={items}
          pagination={{
            current: query.page,
            pageSize: query.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (count) => `共 ${count} 条`,
            onChange: (page, pageSize) =>
              setQuery((prev) => ({ ...prev, page, pageSize })),
          }}
        />
        {columnSettings.modal}
      </Space>
    </Drawer>
  );
}
