import {
  ArrowUpOutlined,
  SendOutlined,
  TableOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { Alert, App, Button, Space, Steps, Tag, Typography } from 'antd';
import React, { useState } from 'react';
import { useAccess } from '@/app/access';
import { SectionCard } from '@/components/ui';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';
import { workbenchServiceSubmitMyCommissionApplication } from '@/services/roncin/workbenchService';
import { applicationAmount, sumBaseAmounts } from './applicationDisplay';
import { amountFont, workbenchAmount } from './display';
import MyApplicationCandidatesDrawer from './MyApplicationCandidatesDrawer';
import MyApplicationHistoryDrawer from './MyApplicationHistoryDrawer';

const { Text, Paragraph } = Typography;

type Summary = API.WorkbenchApplicationSummary;
type Group = API.WorkbenchApplyMonthGroup;
type Estimated = API.WorkbenchEstimatedOpportunity;

/** 阶段推导输入：摘要缺省时各计数按 0 处理。 */
type StageSummary = Pick<Summary, 'pendingReviewCount' | 'applyGroups'>;

/** 申请状态流当前阶段：0 本月累计中 / 1 可申请 / 2 财务审批。 */
export type ApplicationStage = { step: 0 | 1 | 2 };

/**
 * 阶段推导（纯函数）：审批中优先于可申请——有在途申请时用户动作是等待/
 * 查看历史，不是再次提交，与服务端「同月唯一申请」约束一致。
 */
export function deriveStage(summary?: StageSummary): ApplicationStage {
  if ((summary?.pendingReviewCount ?? 0) > 0) return { step: 2 };
  if ((summary?.applyGroups?.length ?? 0) > 0) return { step: 1 };
  return { step: 0 };
}

type Props = {
  /** Overview 返回的本人月度申请摘要段；缺省时各计数按 0 展示，金额为本位币口径。 */
  summary?: Summary;
  /** Overview 提成摘要中的预计可计提段，作为本卡前瞻脚注。 */
  estimated?: Estimated;
  currency?: string;
  /** 提交成功后刷新 Overview；返回 Promise，解除 loading 前等待刷新完成。 */
  onOverviewRefresh: () => Promise<void>;
};

/** Steps 阶段金额：14px、加粗、等宽数字。 */
const stepAmountStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 600,
  ...amountFont,
};

/**
 * 工作台月度申请卡：三阶段状态流（本月累计中 → 可申请 → 财务审批）+
 * 动作区 + 预计可计提前瞻脚注。
 * - 只确认服务端计算结果，文案不涉及付款承诺类表述；
 * - 本月累计中不渲染任何申请入口（服务端同样拒绝当月提交）；
 * - 提交无业务参数，服务端以当前会话身份全量重算候选并固化快照。
 */
export default function MyApplicationCard({
  summary,
  estimated,
  currency,
  onOverviewRefresh,
}: Props) {
  const { canOperateBusiness } = useAccess();
  const { message, modal } = App.useApp();
  const [submitting, setSubmitting] = useState(false);
  const [candidatesOpen, setCandidatesOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);

  const groups: Group[] = summary?.applyGroups ?? [];
  const currencyText = currency || summary?.baseCurrency || '-';
  const totalCount = groups.reduce(
    (acc, group) => acc + (group.commissionCount ?? 0),
    0,
  );
  const totalAmount = sumBaseAmounts(
    groups.map((group) => group.commissionAmount),
  );
  const accumulatingCount = summary?.accumulatingCount ?? 0;
  const pendingReviewCount = summary?.pendingReviewCount ?? 0;
  const approvedCount = summary?.approvedCount ?? 0;
  const latest = summary?.latestApplication;
  const opportunityCount = estimated?.opportunityCount ?? 0;
  const stage = deriveStage(summary);

  const latestRejected =
    Boolean(latest?.applicationMonth) &&
    latest?.status ===
      WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED;

  const submitApplication = () => {
    if (!canOperateBusiness || groups.length === 0 || submitting) return;
    const monthLabels = groups
      .map((group) => group.commissionMonth || '-')
      .join('、');
    modal.confirm({
      title: '确认提交月度提成申请？',
      content: (
        <div>
          <p>
            {`本次申请将覆盖截至上一自然月末的全部合格提成：归属月份 ${monthLabels}，共 ${totalCount} 笔，合计 ${applicationAmount(totalAmount)} ${currencyText}。`}
          </p>
          <p style={{ color: '#64748b' }}>
            覆盖截止日为提交月份的上一自然月末；提交后新满足条件的提成将顺延至下一次申请，
            不会追加进本申请。申请提交后进入财务整批审批，结果请在申请历史中查看。
          </p>
        </div>
      ),
      okText: '确认申请',
      cancelText: '再想想',
      onOk: async () => {
        setSubmitting(true);
        try {
          await workbenchServiceSubmitMyCommissionApplication({});
          message.success('申请已提交，等待财务审批');
          // 等待 Overview 刷新完成再解除 loading，避免旧摘要误导重复提交。
          await onOverviewRefresh();
        } catch (error: unknown) {
          message.error((error as Error).message || '提交申请失败');
        } finally {
          setSubmitting(false);
        }
      },
    });
  };

  const stepsItems: Array<{ title: string; content: React.ReactNode }> = [
    {
      title: '本月累计中',
      content: (
        <div>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
            <Text style={stepAmountStyle}>
              {applicationAmount(summary?.accumulatingAmount)}
            </Text>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {`${accumulatingCount} 笔 · ${currencyText}`}
            </Text>
          </div>
          <Text type="secondary" style={{ fontSize: 12 }}>
            当前自然月尚未结束，提成持续累计；月末结束后可在下一次申请中提交。
          </Text>
        </div>
      ),
    },
    {
      title: '可申请（截至上一自然月末）',
      content:
        groups.length === 0 ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            暂无可申请提成；已提交的批次请在申请历史中查看。
          </Text>
        ) : (
          <div>
            {groups.map((group) => (
              <div key={group.commissionMonth || ''} style={{ fontSize: 12 }}>
                <Text>{group.commissionMonth || '-'}</Text>
                <Text type="secondary">
                  {`  ${group.commissionCount ?? 0} 笔 · ${applicationAmount(group.commissionAmount)} ${currencyText}`}
                </Text>
              </div>
            ))}
            <div
              style={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 6,
                marginTop: 2,
              }}
            >
              <Text style={stepAmountStyle}>
                {applicationAmount(totalAmount)}
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {`合计 ${totalCount} 笔 · ${currencyText}`}
              </Text>
            </div>
          </div>
        ),
    },
    {
      title: '财务审批',
      content: (
        <div>
          <Space size={8} wrap>
            <Tag color="processing">{`审批中 ${pendingReviewCount} 张`}</Tag>
            <Tag color="success">{`已批准 ${approvedCount} 张`}</Tag>
          </Space>
          {latest?.applicationMonth ? (
            <div style={{ marginTop: 4 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {`最近申请：${latest.applicationMonth} · ${latest.commissionCount ?? 0} 笔`}
              </Text>
            </div>
          ) : null}
        </div>
      ),
    },
  ];

  return (
    <SectionCard
      title={
        <>
          月度申请
          <Tag color="blue" style={{ fontSize: 12 }}>
            本位币：{currencyText}
          </Tag>
        </>
      }
      extra={
        <Button
          icon={<UnorderedListOutlined />}
          onClick={() => setHistoryOpen(true)}
        >
          申请历史
        </Button>
      }
      style={{ marginBottom: 0 }}
    >
      <Steps size="small" current={stage.step} items={stepsItems} />

      {/* 动作区：紧跟 Steps 下方右侧；去申请仅对可办理业务的用户开放。 */}
      <div
        style={{
          marginTop: 12,
          display: 'flex',
          justifyContent: 'flex-end',
        }}
      >
        <Space size={8}>
          <Button
            icon={<TableOutlined />}
            disabled={groups.length === 0}
            onClick={() => setCandidatesOpen(true)}
          >
            候选明细
          </Button>
          {canOperateBusiness ? (
            <Button
              type="primary"
              icon={<SendOutlined />}
              disabled={groups.length === 0 || submitting}
              loading={submitting}
              onClick={submitApplication}
              data-testid="apply-submit-button"
            >
              去申请
            </Button>
          ) : null}
        </Space>
      </div>

      {latestRejected ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginTop: 12 }}
          title="最近一次申请已被驳回，请在申请历史中查看原因并重新提交。"
        />
      ) : null}

      {/* 预计可计提前瞻脚注：预计口径警示与部分统计声明必须保留。 */}
      <div
        style={{
          marginTop: 16,
          paddingTop: 12,
          borderTop: '1px solid #f0f0f0',
          display: 'flex',
          justifyContent: 'space-between',
          gap: 12,
          flexWrap: 'wrap',
        }}
      >
        <div style={{ flex: 1, minWidth: 220 }}>
          <Space size={8}>
            <ArrowUpOutlined style={{ color: '#1677ff' }} />
            <Text strong style={{ fontSize: 13 }}>
              预计可计提
            </Text>
            <Tag color="orange" style={{ fontSize: 12 }}>
              预计，非应发承诺
            </Tag>
          </Space>
          {estimated && opportunityCount > 0 ? (
            <Paragraph style={{ marginTop: 4, marginBottom: 0, fontSize: 13 }}>
              <Space size={6} align="baseline">
                <Text strong style={{ fontSize: 18, ...amountFont }}>
                  {workbenchAmount(estimated.estimatedAmount)}
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {`${currencyText} · ${opportunityCount} 笔机会${
                    estimated.hasMore === true
                      ? '（候选超出扫描上限，金额仅为部分统计）'
                      : ''
                  }`}
                </Text>
              </Space>
            </Paragraph>
          ) : (
            <Paragraph style={{ marginTop: 4, marginBottom: 0, fontSize: 13 }}>
              <Space size={6} align="baseline">
                <Text type="secondary">暂无法估算</Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  尚无可匹配的有效方案与人员归属，或计提来源尚未满足计提条件
                </Text>
              </Space>
            </Paragraph>
          )}
        </div>
      </div>

      {candidatesOpen ? (
        <MyApplicationCandidatesDrawer
          open
          baseCurrency={currencyText}
          monthOptions={groups
            .filter((group) => group.commissionMonth)
            .map((group) => ({
              label: group.commissionMonth as string,
              value: group.commissionMonth as string,
            }))}
          onClose={() => setCandidatesOpen(false)}
        />
      ) : null}
      {historyOpen ? (
        <MyApplicationHistoryDrawer
          open
          baseCurrency={currencyText}
          onClose={() => setHistoryOpen(false)}
          onResubmitted={onOverviewRefresh}
        />
      ) : null}
    </SectionCard>
  );
}
