import { PageContainer } from '@ant-design/pro-components';
import { Button, Descriptions, Result, Spin, Tag } from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router';
import { SectionCard } from '@/components/ui';
import { FinanceCommissionStatus } from '@/enums.generated';
import { settlementServiceGetMyFeeSupplementAdjustmentSource } from '@/services/roncin/settlementService';
import { formatDate } from '@/utils/format';
import { commissionDecreaseStatusMeta, decimalText } from './types';

/**
 * 员工本人专属冲减来源落地页：只展示后端白名单字段，不提供任何管理动作；
 * 404/无权限统一按不存在处理，不泄露他人记录事实。
 */
export default function MySupplementSourcePage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const requestSequenceRef = useRef(0);
  const [loading, setLoading] = useState(Boolean(id));
  const [notFound, setNotFound] = useState(!id);
  const [source, setSource] = useState<API.MyFeeSupplementAdjustmentSource>();

  useEffect(() => {
    const requestSequence = ++requestSequenceRef.current;
    if (!id) {
      setNotFound(true);
      setLoading(false);
      return;
    }
    setLoading(true);
    setNotFound(false);
    setSource(undefined);
    settlementServiceGetMyFeeSupplementAdjustmentSource(
      { id },
      { skipErrorHandler: true },
    )
      .then((response) => {
        if (requestSequence !== requestSequenceRef.current) return;
        setSource(response.data);
      })
      .catch(() => {
        // 他人调整、参数非法或已删除统一按不存在处理。
        if (requestSequence !== requestSequenceRef.current) return;
        setNotFound(true);
      })
      .finally(() => {
        if (requestSequence === requestSequenceRef.current) {
          setLoading(false);
        }
      });
  }, [id]);

  const status = source?.status;
  const statusMeta =
    commissionDecreaseStatusMeta[
      status ?? FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_DRAFT
    ];

  return (
    <PageContainer
      title="我的提成调整来源"
      subTitle="费用变更或收付款变化产生的提成调整；发放、补付与扣回由财务执行"
      style={{ minHeight: '100vh', backgroundColor: '#f5f7fa' }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <Spin size="large" />
        </div>
      ) : notFound || !source ? (
        <Result
          status="404"
          title="记录不存在"
          subTitle="未找到对应的提成调整，或该记录与您无关。"
          extra={
            <Button type="primary" href="/welcome">
              返回工作台
            </Button>
          }
        />
      ) : (
        <>
          <SectionCard title="提成调整" style={{ marginBottom: 16 }}>
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 2, md: 3, lg: 4, xl: 4 }}
              bordered
            >
              <Descriptions.Item label="调整编号">
                {source.adjustmentNo || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusMeta?.color}>
                  {status ===
                    FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_PAID &&
                  source.direction === 'INCREASE'
                    ? '已补付'
                    : statusMeta?.text || '-'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="调整金额">
                <strong
                  style={{
                    color:
                      source.direction === 'INCREASE' ? '#389e0d' : '#d4380d',
                  }}
                >
                  {`${source.direction === 'INCREASE' ? '+' : '-'}${decimalText(source.suggestedAmount)} ${source.baseCurrency || ''}`}
                </strong>
              </Descriptions.Item>
              <Descriptions.Item label="生成时间">
                {formatDate(source.createdAt, 'minute')}
              </Descriptions.Item>
              <Descriptions.Item label="订单号">
                {source.orderNo || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="调整来源">
                {{
                  LOCKED_FEE_SUPPLEMENT: '费用补录',
                  FEE_CORRECTION: '费用更正',
                  ECONOMIC_REALLOCATION: '收付款重算',
                }[source.sourceType || ''] || '-'}{' '}
                {source.sourceNo || ''}
              </Descriptions.Item>
              <Descriptions.Item label="原提成号">
                {source.commissionNo || '-'}
              </Descriptions.Item>
            </Descriptions>
          </SectionCard>

          {source.feeName && (
            <SectionCard title="相关费用摘要" style={{ marginBottom: 16 }}>
              <Descriptions
                size="small"
                column={{ xs: 1, sm: 2, md: 3, lg: 4, xl: 4 }}
                bordered
              >
                <Descriptions.Item label="费用名称">
                  {source.feeName || source.feeCode || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="费用金额">
                  {`${decimalText(source.feeTotalAmount)} ${source.feeCurrency || ''}`}
                </Descriptions.Item>
                <Descriptions.Item label="本位币金额">
                  {`${decimalText(source.feeBaseCurrencyAmount)} ${source.feeBaseCurrency || ''}`}
                </Descriptions.Item>
              </Descriptions>
            </SectionCard>
          )}

          <SectionCard title="调整原因" style={{ marginBottom: 16 }}>
            {source.supplementReason || '-'}
          </SectionCard>
        </>
      )}
    </PageContainer>
  );
}
