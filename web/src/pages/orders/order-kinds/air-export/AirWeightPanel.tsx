import {
  type ActionType,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import type { FormInstance } from 'antd';
import {
  Alert,
  Button,
  Descriptions,
  InputNumber,
  Radio,
  Space,
  Spin,
} from 'antd';
import type { Rule } from 'antd/es/form';
import { useRef, useState } from 'react';
import {
  GuardedModalForm,
  ResponsiveProTable,
  SectionCard,
} from '@/components/ui';
import {
  airWeightServiceConfirmAirWeight,
  airWeightServiceListAirWeightConfirmations,
  airWeightServicePreviewAirWeight,
} from '@/services/roncin/airWeightService';
import { ensureListResponse, isWriteResponse } from '@/utils/api';
import { formatDate, trimDecimal } from '@/utils/format';

export default function AirWeightPanel({
  scope,
  targetId,
  canUpdate,
  onConfirmed,
}: {
  scope: 'ORDER' | 'MASTER';
  targetId: string;
  canUpdate: boolean;
  onConfirmed?: () => void | Promise<void>;
}) {
  const [source, setSource] = useState<'ACTUAL' | 'PLANNED'>('ACTUAL');
  const [divisor, setDivisor] = useState('6000');
  const [numerator, setNumerator] = useState(1);
  const [denominator, setDenominator] = useState(1);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const client = useQueryClient();
  const historyRef = useRef<ActionType | undefined>(undefined);
  const validRule =
    Number(divisor) > 0 &&
    Number.isFinite(Number(divisor)) &&
    Number.isSafeInteger(numerator) &&
    Number.isSafeInteger(denominator) &&
    numerator >= 0 &&
    denominator > 0 &&
    numerator <= denominator;
  const input = {
    scope,
    targetId,
    source,
    volumetricDivisor: divisor,
    bubbleShareNumerator: numerator,
    bubbleShareDenominator: denominator,
  };
  const preview = useQuery({
    queryKey: ['air-weight', 'preview', input],
    enabled: validRule,
    queryFn: async ({ signal }) => {
      const response = await airWeightServicePreviewAirWeight(input, {
        signal,
      });
      if (!response?.data) throw new Error('重量参考加载失败');
      return response.data;
    },
  });
  const data =
    !validRule || preview.isError || preview.isFetching
      ? undefined
      : preview.data;
  return (
    <SectionCard
      title={scope === 'ORDER' ? '本票计费重量' : '主单计费重量'}
      extra={
        <Button disabled={!validRule} onClick={() => void preview.refetch()}>
          重新核对
        </Button>
      }
    >
      <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        {!validRule && (
          <Alert
            type="warning"
            title="请填写正除数与有效分泡比例（0 ≤ 分子 ≤ 分母，分母为正整数）"
          />
        )}
        <Alert
          type="info"
          showIcon
          title="按明确货量来源计算参考，人工确认最终采用值"
          description="主单与本票分别计算；主单重量不会复制到每票费用。参数或货量变化会保留原确认及历史，并提示依据已变化。"
        />
        <Radio.Group
          aria-label="重量货量来源"
          value={source}
          onChange={(event) => setSource(event.target.value)}
          options={[
            { label: '实际货物明细', value: 'ACTUAL' },
            { label: '订单委托货量', value: 'PLANNED' },
          ]}
        />
        <Space wrap>
          <span>体积重除数（cm³/kg）</span>
          <InputNumber
            aria-label="体积重除数"
            stringMode
            min="0.001"
            value={divisor}
            onChange={(value) => setDivisor(value || '')}
          />
          <span>分泡比例</span>
          <InputNumber
            aria-label="分泡分子"
            min={0}
            precision={0}
            value={numerator}
            onChange={(value) => setNumerator(value ?? 0)}
          />
          <span>/</span>
          <InputNumber
            aria-label="分泡分母"
            min={1}
            precision={0}
            value={denominator}
            onChange={(value) => setDenominator(value ?? 1)}
          />
        </Space>
        {preview.isFetching && (
          <Spin description="正在核对输入与参考重量">
            <div style={{ minHeight: 60 }} />
          </Spin>
        )}
        {preview.isError && (
          <Alert
            type="error"
            title="重量参考加载失败"
            description="请重新核对后再确认，当前不能使用旧参考。"
          />
        )}
        {data && (
          <>
            {!data.available && (
              <Alert
                type="warning"
                title="暂不能计算参考"
                description={data.reason}
              />
            )}
            <Descriptions
              size="small"
              column={{ xs: 1, sm: 2, lg: 4 }}
              items={[
                {
                  key: 'gross',
                  label: '来源毛重 KG',
                  children: trimDecimal(data.grossWeightKg),
                },
                {
                  key: 'volume',
                  label: '来源体积 m³',
                  children: trimDecimal(data.volumeCbm),
                },
                {
                  key: 'volumetric',
                  label: '体积重 KG',
                  children: trimDecimal(data.volumetricWeightKg),
                },
                {
                  key: 'bubble',
                  label: '泡差 KG',
                  children: trimDecimal(data.bubbleDifferenceKg),
                },
                {
                  key: 'shared',
                  label: '分泡重量 KG',
                  children: trimDecimal(data.sharedBubbleWeightKg),
                },
                {
                  key: 'suggested',
                  label: '参考计费重 KG',
                  children: trimDecimal(data.suggestedChargeableWeightKg),
                },
                {
                  key: 'confirmed',
                  label: '已确认计费重 KG',
                  children: trimDecimal(
                    data.currentConfirmation?.confirmedWeightKg,
                  ),
                },
                {
                  key: 'at',
                  label: '确认时间',
                  children: formatDate(
                    data.currentConfirmation?.confirmedAt,
                    'datetime',
                  ),
                },
              ]}
            />
            {data.confirmationStale && (
              <Alert
                type="warning"
                showIcon
                title="确认依据已变化"
                description="原确认值和历史已保留；请核对新参考并重新确认后再带入新费用。"
              />
            )}
            {canUpdate && (
              <Button type="primary" onClick={() => setConfirmOpen(true)}>
                确认或调整计费重
              </Button>
            )}
          </>
        )}
        <ResponsiveProTable<API.AirWeightConfirmation>
          rowKey="id"
          actionRef={historyRef}
          search={false}
          options={false}
          size="small"
          headerTitle="确认历史"
          pagination={{ defaultPageSize: 10 }}
          request={async ({ current, pageSize }) => {
            const response = await airWeightServiceListAirWeightConfirmations({
              scope,
              targetId,
              page: current || 1,
              pageSize: pageSize || 10,
            });
            ensureListResponse(response, '重量历史加载失败');
            return {
              data: response?.data || [],
              total: response?.total || 0,
              success: true,
            };
          }}
          columns={[
            { title: '版本', dataIndex: 'version', width: 70 },
            {
              title: '货量来源',
              render: (_, item) =>
                item.source === 'PLANNED' ? '订单委托货量' : '实际货物明细',
            },
            {
              title: '确认重量（KG）',
              dataIndex: 'confirmedWeightKg',
              render: (_, item) => trimDecimal(item.confirmedWeightKg),
            },
            { title: '依据/调整原因', dataIndex: 'reason', ellipsis: true },
            {
              title: '确认时间',
              dataIndex: 'confirmedAt',
              render: (_, item) => formatDate(item.confirmedAt, 'datetime'),
              width: 180,
            },
          ]}
        />
      </Space>
      {confirmOpen && data && (
        <GuardedModalForm<{ confirmedWeightKg: string; reason?: string }>
          title="确认计费重量"
          open
          onOpenChange={setConfirmOpen}
          guardIdentity={`${scope}:${targetId}`}
          initialValues={{
            confirmedWeightKg:
              data.suggestedChargeableWeightKg ||
              data.currentConfirmation?.confirmedWeightKg,
          }}
          modalProps={{ destroyOnHidden: true }}
          onFinish={async (values) => {
            if (preview.isFetching || preview.isError || !data.inputFingerprint)
              return false;
            const response = await airWeightServiceConfirmAirWeight({
              ...input,
              inputFingerprint: data.inputFingerprint,
              expectedVersion: data.currentConfirmation?.version || '0',
              confirmedWeightKg: values.confirmedWeightKg,
              reason: values.reason || '',
            });
            if (!isWriteResponse(response, true)) return false;
            await client.invalidateQueries({ queryKey: ['air-weight'] });
            historyRef.current?.reload();
            await onConfirmed?.();
            setConfirmOpen(false);
            return true;
          }}
        >
          <Alert
            type="info"
            title={`本次参考：${trimDecimal(data.suggestedChargeableWeightKg)} KG`}
            description="最终值允许人工调整；不同于参考或无参考时必须填写依据。保存时服务端再次校验货量、成员和版本。"
            style={{ marginBottom: 16 }}
          />
          <ProFormText
            name="confirmedWeightKg"
            label="最终确认重量（KG）"
            rules={[
              { required: true, message: '请输入确认重量' },
              {
                pattern: /^\d+(\.\d{1,4})?$/,
                message: '请输入正数，最多四位小数',
              },
            ]}
          />
          <ProFormTextArea
            name="reason"
            label="确认依据 / 调整原因"
            rules={[
              ({ getFieldValue }: FormInstance) =>
                ({
                  validator: async (_, value) => {
                    if (
                      (!data.available ||
                        Number(getFieldValue('confirmedWeightKg')) !==
                          Number(data.suggestedChargeableWeightKg)) &&
                      !String(value || '').trim()
                    )
                      throw new Error('人工调整或无参考时请填写依据');
                  },
                }) as Rule,
            ]}
            fieldProps={{ maxLength: 500 }}
          />
        </GuardedModalForm>
      )}
    </SectionCard>
  );
}
