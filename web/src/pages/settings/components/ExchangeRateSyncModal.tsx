import { SyncOutlined, WarningOutlined } from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  Typography,
} from 'antd';
import React, { useEffect, useState } from 'react';
import { ExchangeRateSyncTarget } from '@/enums.generated';
import {
  exchangeRateServiceFetchExchangeRates,
  exchangeRateServiceSyncExchangeRates,
} from '@/services/roncin/exchangeRateService';
import { trimDecimal } from '@/utils/format';

const { Text } = Typography;

export type ExchangeRateSyncTargetValue = 'CURRENT_WEEK' | 'NEXT_WEEK';

export interface ExchangeRateSyncDraftRow {
  fromCurrency: string;
  arRate: string;
  apRate: string;
  rate: string;
  conversionPath: string;
}

interface ExchangeRateSyncPreviewData {
  target?: number;
  baseCurrency?: string;
  effectiveFrom?: string;
  effectiveTo?: string;
  source?: string;
  fallbackUsed?: boolean;
  rows?: API.ExchangeRateSyncPreviewRow[];
}

const formatWeekRange = (from?: string, to?: string) => {
  if (!from || !to) return '';
  const parse = (value: string) => new Date(value);
  const format = (date: Date) =>
    `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(
      date.getDate(),
    ).padStart(2, '0')}`;
  return `${format(parse(from))} ~ ${format(parse(to))}`;
};

interface ExchangeRateSyncModalProps {
  open: boolean;
  baseCurrency: string;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * ExchangeRateSyncModal 牌价一键同步弹窗：本周/预设下周切换、抓取预览、
 * 财务微调与终审「确认发布」；抓取失败显式引导手工录入（非静默兜底）。
 */
export function ExchangeRateSyncModal({
  open,
  baseCurrency,
  onClose,
  onSuccess,
}: ExchangeRateSyncModalProps) {
  const { message } = App.useApp();
  const [target, setTarget] =
    useState<ExchangeRateSyncTargetValue>('CURRENT_WEEK');
  const [fetching, setFetching] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [preview, setPreview] = useState<
    ExchangeRateSyncPreviewData | undefined
  >();
  const [draftRows, setDraftRows] = useState<ExchangeRateSyncDraftRow[]>([]);
  const [fetchError, setFetchError] = useState<string>();

  useEffect(() => {
    if (!open) {
      setTarget('CURRENT_WEEK');
      setPreview(undefined);
      setDraftRows([]);
      setFetchError(undefined);
    }
  }, [open]);

  const applyPreview = (data: ExchangeRateSyncPreviewData) => {
    setPreview(data);
    const seed = (value?: string) => (value ? trimDecimal(value) : '');
    setDraftRows(
      (data.rows ?? []).map((row) => ({
        fromCurrency: row.fromCurrency ?? '',
        arRate: seed(row.arRate),
        apRate: seed(row.apRate),
        rate: seed(row.rate),
        conversionPath: row.conversionPath ?? '',
      })),
    );
  };

  const handleFetch = async (nextTarget?: ExchangeRateSyncTargetValue) => {
    const effectiveTarget = nextTarget ?? target;
    if (nextTarget) {
      setTarget(nextTarget);
    }
    setFetching(true);
    setFetchError(undefined);
    try {
      const response = await exchangeRateServiceFetchExchangeRates({
        target:
          effectiveTarget === 'NEXT_WEEK'
            ? ExchangeRateSyncTarget.EXCHANGE_RATE_SYNC_TARGET_NEXT_WEEK
            : ExchangeRateSyncTarget.EXCHANGE_RATE_SYNC_TARGET_CURRENT_WEEK,
      });
      if (!response.data) {
        throw new Error('抓取结果为空');
      }
      applyPreview(response.data as ExchangeRateSyncPreviewData);
    } catch (error: any) {
      setPreview(undefined);
      setDraftRows([]);
      setFetchError(error?.message || '牌价抓取失败');
    } finally {
      setFetching(false);
    }
  };

  const updateDraft = (
    fromCurrency: string,
    field: keyof ExchangeRateSyncDraftRow,
    value: string,
  ) => {
    setDraftRows((rows) =>
      rows.map((row) =>
        row.fromCurrency === fromCurrency ? { ...row, [field]: value } : row,
      ),
    );
  };

  const handlePublish = async () => {
    if (draftRows.length === 0) return;
    setPublishing(true);
    try {
      await exchangeRateServiceSyncExchangeRates({
        target:
          target === 'NEXT_WEEK'
            ? ExchangeRateSyncTarget.EXCHANGE_RATE_SYNC_TARGET_NEXT_WEEK
            : ExchangeRateSyncTarget.EXCHANGE_RATE_SYNC_TARGET_CURRENT_WEEK,
        rows: draftRows.map((row) => ({
          fromCurrency: row.fromCurrency,
          arRate: row.arRate,
          apRate: row.apRate,
          rate: row.rate,
        })),
      });
      message.success('周汇率已发布生效');
      onSuccess();
      onClose();
    } catch (error: any) {
      message.error(error?.message || '周汇率发布失败');
    } finally {
      setPublishing(false);
    }
  };

  const weekRange = formatWeekRange(
    preview?.effectiveFrom,
    preview?.effectiveTo,
  );

  return (
    <Modal
      title={
        baseCurrency === 'CNY'
          ? '从中国银行同步周汇率'
          : `一键同步周汇率（本币 ${baseCurrency}）`
      }
      open={open}
      onCancel={onClose}
      width={860}
      destroyOnHidden
      footer={[
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="publish"
          type="primary"
          loading={publishing}
          disabled={draftRows.length === 0}
          onClick={handlePublish}
        >
          确认发布
        </Button>,
      ]}
    >
      <Space direction="vertical" style={{ width: '100%' }} size={12}>
        <Space wrap>
          <Text>目标生效周：</Text>
          <Radio.Group
            value={target}
            onChange={(event) => handleFetch(event.target.value)}
            optionType="button"
            buttonStyle="solid"
            options={[
              { label: '本周（周一至周日）', value: 'CURRENT_WEEK' },
              { label: '预设下周（下周一至下周日）', value: 'NEXT_WEEK' },
            ]}
          />
          <Button
            type="primary"
            icon={<SyncOutlined />}
            loading={fetching}
            onClick={() => handleFetch()}
          >
            抓取牌价
          </Button>
        </Space>

        {fetchError && (
          <Alert
            type="error"
            showIcon
            message="牌价抓取失败"
            description={`${fetchError}。请检查网络后重试，或使用「新建汇率」手工录入本周汇率，单据不会因此受阻。`}
          />
        )}

        {preview && (
          <>
            <Alert
              type={preview.fallbackUsed ? 'warning' : 'info'}
              showIcon={preview.fallbackUsed}
              icon={preview.fallbackUsed ? <WarningOutlined /> : undefined}
              message={`数据来源：${preview.source ?? '-'}（${weekRange}）`}
              description={
                preview.fallbackUsed
                  ? '主数据源暂不可用，已启用备选来源换算，请逐行核对后再发布。'
                  : '请核对各币种建议值，可直接修改实现商业加点或报价取整，点击「确认发布」完成财务终审。同周重复发布将覆盖更新本周配置。'
              }
            />
            <Table<ExchangeRateSyncDraftRow>
              rowKey="fromCurrency"
              size="small"
              pagination={false}
              dataSource={draftRows}
              columns={[
                {
                  title: '币种',
                  dataIndex: 'fromCurrency',
                  width: 90,
                },
                {
                  title: '应收汇率（现汇卖出价）',
                  dataIndex: 'arRate',
                  width: 170,
                  render: (_, row) => (
                    <Input
                      value={row.arRate}
                      onChange={(event) =>
                        updateDraft(
                          row.fromCurrency,
                          'arRate',
                          event.target.value,
                        )
                      }
                    />
                  ),
                },
                {
                  title: '应付汇率（现汇买入价）',
                  dataIndex: 'apRate',
                  width: 170,
                  render: (_, row) => (
                    <Input
                      value={row.apRate}
                      onChange={(event) =>
                        updateDraft(
                          row.fromCurrency,
                          'apRate',
                          event.target.value,
                        )
                      }
                    />
                  ),
                },
                {
                  title: '基准汇率',
                  dataIndex: 'rate',
                  width: 140,
                  render: (_, row) => (
                    <Input
                      value={row.rate}
                      onChange={(event) =>
                        updateDraft(
                          row.fromCurrency,
                          'rate',
                          event.target.value,
                        )
                      }
                    />
                  ),
                },
                {
                  title: '来源 / 换算路径',
                  dataIndex: 'conversionPath',
                  ellipsis: true,
                },
              ]}
            />
          </>
        )}
      </Space>
    </Modal>
  );
}

export default ExchangeRateSyncModal;
