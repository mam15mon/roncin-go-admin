import {
  SendOutlined,
  TableOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import { useAccess } from '@/app/access';
import { App, Button, Space, Tag, Typography } from 'antd';
import React, { useState } from 'react';
import { WorkbenchCommissionApplicationStatus } from '@/enums.generated';
import { workbenchServiceSubmitMyCommissionApplication } from '@/services/roncin/workbenchService';
import { applicationAmount, sumBaseAmounts } from './applicationDisplay';
import MyApplicationCandidatesDrawer from './MyApplicationCandidatesDrawer';
import MyApplicationHistoryDrawer from './MyApplicationHistoryDrawer';

const { Text } = Typography;

type Summary = API.WorkbenchApplicationSummary;
type Group = API.WorkbenchApplyMonthGroup;

type Props = {
  /** Overview 返回的本人月度申请摘要段；金额全部为组织本位币口径。 */
  summary: Summary;
  currency?: string;
  /** 提交成功后刷新 Overview；返回 Promise，解除 loading 前等待刷新完成。 */
  onOverviewRefresh: () => Promise<void>;
};

/**
 * 工作台月度申请面板：可申请（按归属月分组）/ 本月累计中 / 审批中 / 已批准。
 * - 只确认服务端计算结果，文案不涉及付款承诺类表述；
 * - 本月累计中不渲染任何申请入口（服务端同样拒绝当月提交）；
 * - 提交无业务参数，服务端以当前会话身份全量重算候选并固化快照。
 */
export default function MyApplicationPanel({
  summary,
  currency,
  onOverviewRefresh,
}: Props) {
  const { canOperateBusiness } = useAccess();
  const { message, modal } = App.useApp();
  const [submitting, setSubmitting] = useState(false);
  const [candidatesOpen, setCandidatesOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);

  const groups: Group[] = summary.applyGroups ?? [];
  const currencyText = currency || summary.baseCurrency || '-';
  const totalCount = groups.reduce(
    (acc, group) => acc + (group.commissionCount ?? 0),
    0,
  );
  const totalAmount = sumBaseAmounts(
    groups.map((group) => group.commissionAmount),
  );
  const accumulatingCount = summary.accumulatingCount ?? 0;
  const pendingReviewCount = summary.pendingReviewCount ?? 0;
  const approvedCount = summary.approvedCount ?? 0;
  const latest = summary.latestApplication;

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

  return (
    <div
      style={{
        marginTop: 16,
        paddingTop: 12,
        borderTop: '1px solid #f0f0f0',
      }}
      data-testid="application-summary-panel"
    >
      <Space size={8} style={{ marginBottom: 8 }}>
        <SendOutlined style={{ color: '#1677ff' }} />
        <Text strong style={{ fontSize: 13 }}>
          月度申请
        </Text>
        <Tag color="blue" style={{ fontSize: 12 }}>
          本位币：{currencyText}
        </Tag>
      </Space>

      <div
        style={{
          display: 'flex',
          gap: 24,
          flexWrap: 'wrap',
          alignItems: 'flex-start',
        }}
      >
        <div style={{ flex: 1, minWidth: 260 }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            可申请提成（截至上一自然月末）
          </Text>
          {groups.length === 0 ? (
            <div style={{ marginTop: 4 }}>
              <Text type="secondary" style={{ fontSize: 13 }}>
                暂无可申请提成；已提交的批次请在申请历史中查看。
              </Text>
            </div>
          ) : (
            <>
              <div style={{ marginTop: 4 }}>
                {groups.map((group) => (
                  <div
                    key={group.commissionMonth || ''}
                    style={{ fontSize: 13 }}
                  >
                    <Text>{group.commissionMonth || '-'}</Text>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {`  ${group.commissionCount ?? 0} 笔 · ${applicationAmount(group.commissionAmount)} ${currencyText}`}
                    </Text>
                  </div>
                ))}
              </div>
              <div style={{ marginTop: 4 }}>
                <Space size={6} align="baseline">
                  <Text strong style={{ fontSize: 16 }}>
                    {applicationAmount(totalAmount)}
                  </Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {`合计 ${totalCount} 笔 · ${currencyText}`}
                  </Text>
                </Space>
              </div>
            </>
          )}
        </div>

        <div style={{ flex: 1, minWidth: 220 }}>
          <Text type="secondary" style={{ fontSize: 13 }}>
            本月累计中
          </Text>
          <div style={{ marginTop: 4 }}>
            <Space size={6} align="baseline">
              <Text strong style={{ fontSize: 16 }}>
                {applicationAmount(summary.accumulatingAmount)}
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {`${accumulatingCount} 笔 · ${currencyText}`}
              </Text>
            </Space>
          </div>
          <Text type="secondary" style={{ fontSize: 12 }}>
            当前自然月尚未结束，提成持续累计；月末结束后可在下一次申请中提交。
          </Text>
        </div>

        <div style={{ flex: 1, minWidth: 220 }}>
          <Space size={8}>
            <Tag color="processing">{`审批中 ${pendingReviewCount} 张`}</Tag>
            <Tag color="success">{`已批准 ${approvedCount} 张`}</Tag>
          </Space>
          {latest?.applicationMonth ? (
            <div style={{ marginTop: 4 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {`最近申请：${latest.applicationMonth} · ${latest.commissionCount ?? 0} 笔`}
              </Text>
              {latest.status ===
              WorkbenchCommissionApplicationStatus.WORKBENCH_COMMISSION_APPLICATION_STATUS_REJECTED ? (
                <div>
                  <Text type="warning" style={{ fontSize: 12 }}>
                    最近一次申请已被驳回，请在申请历史中查看原因并重新提交。
                  </Text>
                </div>
              ) : null}
            </div>
          ) : null}
          <div style={{ marginTop: 8 }}>
            <Space size={8}>
              {canOperateBusiness && (
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
              )}
              <Button
                icon={<TableOutlined />}
                disabled={groups.length === 0}
                onClick={() => setCandidatesOpen(true)}
              >
                候选明细
              </Button>
              <Button
                icon={<UnorderedListOutlined />}
                onClick={() => setHistoryOpen(true)}
              >
                申请历史
              </Button>
            </Space>
          </div>
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
    </div>
  );
}
