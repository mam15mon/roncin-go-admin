import type { TableColumnsType } from 'antd';
import { Button, Descriptions, Drawer, Space, Spin, Table } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { DRAWER_SIZE } from '@/components/ui';
import { FinanceCommissionApplicationStatus } from '@/enums.generated';
import { settlementServiceGetCommissionApplication } from '@/services/roncin/settlementService';
import { formatDate } from '@/utils/format';
import { calculationBasisText, decimalText, personnelRoleText } from '../types';
import { applicationStatusTag } from './applicationStatus';

type Application = API.FinanceCommissionApplication;
type Line = API.FinanceCommissionApplicationLine;

type Props = {
  open: boolean;
  applicationId?: string;
  canManage: boolean;
  /** 变化时重新拉取详情（如整批决策冲突后刷新最新状态与版本）。 */
  refreshToken: number;
  onClose: () => void;
  onApprove: (record: Application) => void;
  onReject: (record: Application) => void;
};

/**
 * 申请批次明细下钻抽屉：申请头审计字段 + 明细快照（来源、人员身份、
 * 方案快照与金额）。只读下钻；整单批准/驳回动作由父级面板编排，
 * 页面不提供任何部分批准、剔除明细或拆分入口。
 */
export default function CommissionApplicationDetailDrawer({
  open,
  applicationId,
  canManage,
  refreshToken,
  onClose,
  onApprove,
  onReject,
}: Props) {
  const [detail, setDetail] =
    useState<API.FinanceCommissionApplicationDetail>();
  const [loading, setLoading] = useState(false);
  const sequenceRef = useRef(0);

  useEffect(() => {
    if (!open || !applicationId) return;
    const sequence = ++sequenceRef.current;
    setLoading(true);
    settlementServiceGetCommissionApplication({ id: applicationId })
      .then((response) => {
        if (sequence !== sequenceRef.current) return;
        setDetail(response.data);
      })
      .catch(() => {
        // 失败由统一请求错误处理提示；保留当前内容。
      })
      .finally(() => {
        if (sequence === sequenceRef.current) setLoading(false);
      });
    return () => {
      sequenceRef.current += 1;
    };
  }, [open, applicationId, refreshToken]);

  const application = detail?.application;
  const canDecide =
    canManage &&
    application?.status ===
      FinanceCommissionApplicationStatus.FINANCE_COMMISSION_APPLICATION_STATUS_PENDING_REVIEW;

  const columns: TableColumnsType<Line> = [
    {
      title: '提成单号',
      dataIndex: 'commissionNo',
      width: 180,
      render: (_, record) => record.commissionNo || '-',
    },
    {
      title: '归属日期',
      dataIndex: 'commissionDate',
      width: 100,
      render: (_, record) => record.commissionDate || '-',
    },
    {
      title: '来源单号',
      dataIndex: 'sourceNo',
      width: 150,
      render: (_, record) => record.verificationNo || record.nettingNo || '-',
    },
    {
      title: '人员身份',
      dataIndex: 'personnelRole',
      width: 100,
      render: (_, record) => personnelRoleText(record.personnelRole),
    },
    {
      title: '提成方案',
      dataIndex: 'ruleName',
      width: 160,
      render: (_, record) => (
        <Space size={4}>
          <span>{record.ruleName || '-'}</span>
          {record.ruleVersion ? (
            <span style={{ fontSize: 11, color: '#64748b' }}>
              {`v${record.ruleVersion}`}
            </span>
          ) : null}
        </Space>
      ),
    },
    {
      title: '计提口径',
      dataIndex: 'calculationBasis',
      width: 100,
      render: (_, record) => calculationBasisText(record.calculationBasis),
    },
    {
      title: '原币金额',
      dataIndex: 'commissionAmount',
      width: 130,
      align: 'right',
      render: (_, record) =>
        `${decimalText(record.commissionAmount)} ${record.baseCurrency || ''}`,
    },
    {
      title: 'CNY 金额',
      dataIndex: 'cnyCommissionAmount',
      width: 130,
      align: 'right',
      render: (_, record) => `${decimalText(record.cnyCommissionAmount)} CNY`,
    },
  ];

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title="申请批次明细"
      size={DRAWER_SIZE.LG}
      destroyOnHidden
      footer={
        canDecide ? (
          <Space
            size={8}
            style={{ display: 'flex', justifyContent: 'flex-end' }}
          >
            <Button
              type="primary"
              onClick={() => application && onApprove(application)}
            >
              整单批准
            </Button>
            <Button danger onClick={() => application && onReject(application)}>
              整单驳回
            </Button>
          </Space>
        ) : null
      }
    >
      <Spin spinning={loading}>
        <Space orientation="vertical" size={12} style={{ width: '100%' }}>
          <Space size={8}>
            <span style={{ fontSize: 14, fontWeight: 600 }}>
              {application?.employeeName || '-'}
            </span>
            {applicationStatusTag(application?.status)}
          </Space>
          <Descriptions
            size="small"
            column={3}
            bordered
            items={[
              {
                key: 'month',
                label: '申请月份',
                children: application?.applicationMonth || '-',
              },
              {
                key: 'coverageTo',
                label: '覆盖截止日',
                children: application?.coverageTo || '-',
              },
              {
                key: 'version',
                label: '当前版本',
                children: application?.version ?? '-',
              },
              {
                key: 'count',
                label: '笔数',
                children: application?.commissionCount ?? 0,
              },
              {
                key: 'amount',
                label: '本位币总额',
                children: `${decimalText(application?.totalCommissionAmount)} ${application?.baseCurrency || ''}`,
              },
              {
                key: 'cny',
                label: 'CNY 总额',
                children: `${decimalText(application?.totalCnyCommissionAmount)} CNY`,
              },
              {
                key: 'submitted',
                label: '提交人 / 时间',
                children: `${application?.submittedBy || '-'} · ${formatDate(application?.submittedAt)}`,
              },
              {
                key: 'decided',
                label: '决策人 / 时间',
                children: `${application?.decidedBy || '-'} · ${formatDate(application?.decidedAt)}`,
              },
              {
                key: 'reason',
                label: '驳回原因',
                children: application?.decisionReason || '-',
              },
            ]}
          />
          <Table<Line>
            rowKey={(record) => record.id || record.commissionId || ''}
            size="small"
            columns={columns}
            dataSource={detail?.lines ?? []}
            pagination={false}
            scroll={{ x: 1000 }}
          />
        </Space>
      </Spin>
    </Drawer>
  );
}
