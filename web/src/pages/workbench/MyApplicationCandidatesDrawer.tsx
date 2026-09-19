import type { TableColumnsType } from 'antd';
import { Drawer, Select, Space, Table } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
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
  const [items, setItems] = useState<Candidate[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<CandidatesQuery>({
    page: 1,
    pageSize: DEFAULT_PAGE_SIZE,
  });
  const sequenceRef = useRef(0);

  useEffect(() => {
    if (!open) return;
    const sequence = ++sequenceRef.current;
    setLoading(true);
    workbenchServiceListMyApplicationCandidates({
      page: query.page,
      pageSize: query.pageSize,
      ...(query.commissionMonth
        ? { commissionMonth: query.commissionMonth }
        : {}),
    })
      .then((response) => {
        if (sequence !== sequenceRef.current) return;
        setItems(response.data ?? []);
        setTotal(Number(response.total ?? 0));
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；保留当前内容并停止加载。
      })
      .finally(() => {
        if (sequence === sequenceRef.current) setLoading(false);
      });
    return () => {
      sequenceRef.current += 1;
    };
  }, [open, query]);

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

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="可申请提成明细"
      width={920}
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
        <Table<Candidate>
          rowKey={(record) =>
            record.verificationId ||
            record.nettingId ||
            record.commissionDate ||
            ''
          }
          size="small"
          loading={loading}
          columns={columns}
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
      </Space>
    </Drawer>
  );
}
