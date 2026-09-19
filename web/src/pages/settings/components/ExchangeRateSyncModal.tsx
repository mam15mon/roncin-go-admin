import {
  HistoryOutlined,
  InfoCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { TableProps } from 'antd';
import {
  Alert,
  App,
  Button,
  Flex,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useEffect, useState } from 'react';
import { ExchangeRateSyncTarget } from '@/enums.generated';
import {
  exchangeRateServiceFetchExchangeRates,
  exchangeRateServiceSyncExchangeRates,
} from '@/services/roncin/exchangeRateService';
import { getErrorMessage } from '@/utils/errorMessage';
import { trimDecimal } from '@/utils/format';

const { Text } = Typography;

export type ExchangeRateSyncTargetValue = 'CURRENT_WEEK' | 'NEXT_WEEK';
export type ExchangeRatePrecision = '4' | '3' | '2' | 'raw';

export interface ExchangeRateSyncDraftRow {
  fromCurrency: string;
  arRate: string;
  apRate: string;
  rate: string;
  conversionPath: string;
}

export interface ExchangeRateMarkupItem {
  arOffset?: number;
  apOffset?: number;
  rateOffset?: number;
}

export type ExchangeRateMarkupMap = Record<string, ExchangeRateMarkupItem>;

/**
 * 预置贵司执行了 34 周的经典商业加点模板：
 * USD/GBP/EUR/CHF: +0.02, AUD/SGD: +0.015, HKD: +0.002, THB: +0.0005
 */
export const DEFAULT_COMPANY_MARKUPS: ExchangeRateMarkupMap = {
  USD: { apOffset: 0.02 },
  GBP: { apOffset: 0.02 },
  EUR: { apOffset: 0.02 },
  CHF: { apOffset: 0.02 },
  AUD: { apOffset: 0.015 },
  SGD: { apOffset: 0.015 },
  HKD: { apOffset: 0.002 },
  THB: { apOffset: 0.0005 },
};

const MARKUP_STORAGE_KEY = 'roncin_exchange_rate_last_markups';

export const loadLastMarkups = (): ExchangeRateMarkupMap => {
  try {
    const raw = localStorage.getItem(MARKUP_STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      if (parsed && typeof parsed === 'object') {
        return parsed;
      }
    }
  } catch {
    // 忽略异常
  }
  return DEFAULT_COMPANY_MARKUPS;
};

export const saveLastMarkups = (markups: ExchangeRateMarkupMap) => {
  try {
    localStorage.setItem(MARKUP_STORAGE_KEY, JSON.stringify(markups));
  } catch {
    // 忽略异常
  }
};

interface ExchangeRateSyncPreviewData {
  target?: number;
  baseCurrency?: string;
  effectiveFrom?: string;
  effectiveTo?: string;
  source?: string;
  fallbackUsed?: boolean;
  rows?: API.ExchangeRateSyncPreviewRow[];
}

const CURRENCY_CN_NAMES: Record<string, string> = {
  CNY: '人民币',
  USD: '美元',
  EUR: '欧元',
  HKD: '港币',
  GBP: '英镑',
  JPY: '日元',
  AUD: '澳元',
  CAD: '加元',
  SGD: '新加坡元',
  CHF: '瑞士法郎',
  NZD: '新西兰元',
  THB: '泰铢',
  KRW: '韩元',
  MYR: '林吉特',
  RUB: '卢布',
  AED: '迪拉姆',
  SAR: '里亚尔',
  BRL: '雷亚尔',
  INR: '印度卢比',
  PHP: '菲律宾比索',
  IDR: '印尼卢比',
  ZAR: '兰特',
  SEK: '瑞典克朗',
  NOK: '挪威克朗',
  DKK: '丹麦克朗',
  TRY: '土耳其里拉',
  HUF: '福林',
  PLN: '兹罗提',
  ILS: '谢克尔',
  EGP: '埃及镑',
};

const formatWeekRange = (from?: string, to?: string) => {
  if (!from || !to) return '';
  const parse = (value: string) => new Date(value);
  const format = (date: Date) =>
    `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(
      date.getDate(),
    ).padStart(2, '0')}`;
  return `${format(parse(from))} ~ ${format(parse(to))}`;
};

const formatRateValue = (val: string, prec: ExchangeRatePrecision): string => {
  if (!val) return '';
  if (prec === 'raw') {
    return trimDecimal(val);
  }
  const num = Number(val);
  if (!Number.isFinite(num)) return val;
  return num.toFixed(Number(prec));
};

interface ExchangeRateSyncModalProps {
  open: boolean;
  baseCurrency: string;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * ExchangeRateSyncModal 牌价一键同步弹窗：
 * - 本周/预设下周切换、实时抓取中行/新浪专线牌价预览；
 * - 精度选择器（4位中行标准/3位/2位/原始抓取），无损切换；
 * - 商业加点与微调记忆（自动记住上次各币种微调量，支持一键套用与还原）；
 * - 严格固定单元格高度与基线对齐，杜绝错位对不齐与 -0.0000 噪音；
 * - 实时计算买卖点差 (AR - AP)，倒挂鲜明警示；
 * - 财务终审「确认发布」；抓取失败显式引导手工录入。
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
  const [precision, setPrecision] = useState<ExchangeRatePrecision>('4');
  const [fetching, setFetching] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [preview, setPreview] = useState<
    ExchangeRateSyncPreviewData | undefined
  >();
  const [rawRows, setRawRows] = useState<ExchangeRateSyncDraftRow[]>([]);
  const [draftRows, setDraftRows] = useState<ExchangeRateSyncDraftRow[]>([]);
  const [fetchError, setFetchError] = useState<string>();

  useEffect(() => {
    if (!open) {
      setTarget('CURRENT_WEEK');
      setPrecision('4');
      setPreview(undefined);
      setRawRows([]);
      setDraftRows([]);
      setFetchError(undefined);
    }
  }, [open]);

  const applyPreview = (
    data: ExchangeRateSyncPreviewData,
    currentPrecision: ExchangeRatePrecision = precision,
  ) => {
    setPreview(data);
    const raw: ExchangeRateSyncDraftRow[] = (data.rows ?? []).map((row) => ({
      fromCurrency: row.fromCurrency ?? '',
      arRate: row.arRate ? trimDecimal(row.arRate) : '',
      apRate: row.apRate ? trimDecimal(row.apRate) : '',
      rate: row.rate ? trimDecimal(row.rate) : '',
      conversionPath: row.conversionPath ?? '',
    }));
    setRawRows(raw);
    setDraftRows(
      raw.map((row) => ({
        ...row,
        arRate: formatRateValue(row.arRate, currentPrecision),
        apRate: formatRateValue(row.apRate, currentPrecision),
        rate: formatRateValue(row.rate, currentPrecision),
      })),
    );
  };

  const handlePrecisionChange = (nextPrec: ExchangeRatePrecision) => {
    setPrecision(nextPrec);
    setDraftRows((currentDrafts) =>
      currentDrafts.map((draft) => {
        const raw = rawRows.find((r) => r.fromCurrency === draft.fromCurrency);
        const baseAR = raw?.arRate || draft.arRate;
        const baseAP = raw?.apRate || draft.apRate;
        const baseRate = raw?.rate || draft.rate;
        return {
          ...draft,
          arRate: formatRateValue(baseAR, nextPrec),
          apRate: formatRateValue(baseAP, nextPrec),
          rate: formatRateValue(baseRate, nextPrec),
        };
      }),
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
      applyPreview(response.data as ExchangeRateSyncPreviewData, precision);
    } catch (error) {
      setPreview(undefined);
      setRawRows([]);
      setDraftRows([]);
      setFetchError(getErrorMessage(error, '牌价抓取失败'));
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

  // 一键套用上次记住的微调加点（如 USD +0.02、EUR +0.02 等）
  const handleApplyLastMarkups = () => {
    const markups = loadLastMarkups();
    let appliedCount = 0;
    setDraftRows((rows) =>
      rows.map((row) => {
        const raw = rawRows.find((r) => r.fromCurrency === row.fromCurrency);
        const rule = markups[row.fromCurrency];
        if (!rule || !raw) return row;

        let nextAR = Number(raw.arRate);
        let nextAP = Number(raw.apRate);
        let nextRate = Number(raw.rate);

        if (rule.arOffset) {
          nextAR += rule.arOffset;
        }
        if (rule.apOffset) {
          nextAP += rule.apOffset;
        }
        if (rule.rateOffset) {
          nextRate += rule.rateOffset;
        }

        appliedCount += 1;
        return {
          ...row,
          arRate: formatRateValue(String(nextAR), precision),
          apRate: formatRateValue(String(nextAP), precision),
          rate: formatRateValue(String(nextRate), precision),
        };
      }),
    );
    message.success(
      `已套用 ${appliedCount} 个币种的上次微调加点（如 USD +0.02、EUR +0.02 等）`,
    );
  };

  // 一键还原为中行抓取的原始牌价
  const handleRestoreRaw = () => {
    setDraftRows(
      rawRows.map((row) => ({
        ...row,
        arRate: formatRateValue(row.arRate, precision),
        apRate: formatRateValue(row.apRate, precision),
        rate: formatRateValue(row.rate, precision),
      })),
    );
    message.info('已还原为中行原始抓取牌价');
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

      // 自动保存本次的微调加点量（对比 rawRows 计算差值），方便下次一键继承
      const currentMarkups: ExchangeRateMarkupMap = {};
      for (const draft of draftRows) {
        const raw = rawRows.find((r) => r.fromCurrency === draft.fromCurrency);
        if (!raw) continue;
        const arDiff = Number(draft.arRate) - Number(raw.arRate);
        const apDiff = Number(draft.apRate) - Number(raw.apRate);
        const rateDiff = Number(draft.rate) - Number(raw.rate);

        const hasAR = Math.abs(arDiff) >= 0.0001;
        const hasAP = Math.abs(apDiff) >= 0.0001;
        const hasRate = Math.abs(rateDiff) >= 0.0001;

        if (hasAR || hasAP || hasRate) {
          currentMarkups[draft.fromCurrency] = {
            arOffset: hasAR ? Number(arDiff.toFixed(6)) : 0,
            apOffset: hasAP ? Number(apDiff.toFixed(6)) : 0,
            rateOffset: hasRate ? Number(rateDiff.toFixed(6)) : 0,
          };
        }
      }
      if (Object.keys(currentMarkups).length > 0) {
        saveLastMarkups(currentMarkups);
      }

      message.success('周汇率已发布生效，本次微调加点已自动记忆');
      onSuccess();
      onClose();
    } catch (error) {
      message.error(getErrorMessage(error, '周汇率发布失败'));
    } finally {
      setPublishing(false);
    }
  };

  const weekRange = formatWeekRange(
    preview?.effectiveFrom,
    preview?.effectiveTo,
  );

  // 严格计算加点偏移量：阈值过滤舍入噪声，绝不出现 -0.0000 或 +0.0000
  const getMarkupOffset = (
    currentVal?: string,
    rawVal?: string,
  ): string | null => {
    if (!currentVal || !rawVal) return null;
    const curr = Number(currentVal);
    const raw = Number(rawVal);
    if (!Number.isFinite(curr) || !Number.isFinite(raw)) return null;
    const diff = curr - raw;
    if (Math.abs(diff) < 0.0001) return null;
    const p = precision === 'raw' ? 4 : Number(precision);
    const sign = diff > 0 ? `+${diff.toFixed(p)}` : diff.toFixed(p);
    if (sign === '+0.0000' || sign === '-0.0000' || Number(sign) === 0) {
      return null;
    }
    return sign;
  };

  const columns: TableProps<ExchangeRateSyncDraftRow>['columns'] = [
    {
      title: '币种',
      dataIndex: 'fromCurrency',
      width: 110,
      render: (currency: string) => {
        const name = CURRENCY_CN_NAMES[currency];
        return (
          <div
            style={{
              height: 56,
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
            }}
          >
            <Text strong style={{ fontSize: 14 }}>
              {currency}
            </Text>
            {name && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {name}
              </Text>
            )}
          </div>
        );
      },
    },
    {
      title: '应收汇率（卖出价）',
      dataIndex: 'arRate',
      width: 165,
      render: (_, row) => {
        const raw = rawRows.find((r) => r.fromCurrency === row.fromCurrency);
        const offsetInfo = getMarkupOffset(row.arRate, raw?.arRate);
        return (
          <div
            style={{
              height: 56,
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
            }}
          >
            <Input
              value={row.arRate}
              placeholder="现汇卖出价"
              onChange={(event) =>
                updateDraft(row.fromCurrency, 'arRate', event.target.value)
              }
            />
            <div
              style={{
                height: 20,
                display: 'flex',
                alignItems: 'center',
                overflow: 'hidden',
              }}
            >
              {offsetInfo ? (
                <Tag
                  color="blue"
                  style={{
                    fontSize: 11,
                    margin: 0,
                    lineHeight: '18px',
                    padding: '0 6px',
                  }}
                >
                  加点 {offsetInfo}
                </Tag>
              ) : (
                <Text type="secondary" style={{ fontSize: 11 }}>
                  原价: {raw?.arRate || '-'}
                </Text>
              )}
            </div>
          </div>
        );
      },
    },
    {
      title: '应付汇率（买入价）',
      dataIndex: 'apRate',
      width: 165,
      render: (_, row) => {
        const raw = rawRows.find((r) => r.fromCurrency === row.fromCurrency);
        const offsetInfo = getMarkupOffset(row.apRate, raw?.apRate);
        return (
          <div
            style={{
              height: 56,
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
            }}
          >
            <Input
              value={row.apRate}
              placeholder="现汇买入价"
              onChange={(event) =>
                updateDraft(row.fromCurrency, 'apRate', event.target.value)
              }
            />
            <div
              style={{
                height: 20,
                display: 'flex',
                alignItems: 'center',
                overflow: 'hidden',
              }}
            >
              {offsetInfo ? (
                <Tag
                  color="blue"
                  style={{
                    fontSize: 11,
                    margin: 0,
                    lineHeight: '18px',
                    padding: '0 6px',
                  }}
                >
                  加点 {offsetInfo}
                </Tag>
              ) : (
                <Text type="secondary" style={{ fontSize: 11 }}>
                  原价: {raw?.apRate || '-'}
                </Text>
              )}
            </div>
          </div>
        );
      },
    },
    {
      title: (
        <Tooltip title="应收汇率（卖出价）与应付汇率（买入价）之差。正数表示正常收益点差，负数表示买卖倒挂风险。">
          <Space size={2} align="center">
            <span>买卖点差</span>
            <InfoCircleOutlined style={{ fontSize: 12, color: '#8c8c8c' }} />
          </Space>
        </Tooltip>
      ),
      key: 'spread',
      width: 115,
      render: (_, row) => {
        const ar = Number(row.arRate);
        const ap = Number(row.apRate);
        const p = precision === 'raw' ? 4 : Number(precision);
        if (
          !Number.isFinite(ar) ||
          !Number.isFinite(ap) ||
          !row.arRate ||
          !row.apRate
        ) {
          return (
            <div
              style={{
                height: 56,
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <Text type="secondary">-</Text>
            </div>
          );
        }
        const diff = ar - ap;
        const formatted = diff.toFixed(p);
        return (
          <div
            style={{
              height: 56,
              display: 'flex',
              alignItems: 'center',
            }}
          >
            {diff > 0 ? (
              <Tag color="success" style={{ margin: 0 }}>
                +{formatted}
              </Tag>
            ) : diff < 0 ? (
              <Tag color="error" style={{ margin: 0 }}>
                倒挂 {formatted}
              </Tag>
            ) : (
              <Tag style={{ margin: 0 }}>0.{'0'.repeat(p)}</Tag>
            )}
          </div>
        );
      },
    },
    {
      title: '基准汇率（折算价）',
      dataIndex: 'rate',
      width: 145,
      render: (_, row) => {
        const raw = rawRows.find((r) => r.fromCurrency === row.fromCurrency);
        const offsetInfo = getMarkupOffset(row.rate, raw?.rate);
        return (
          <div
            style={{
              height: 56,
              display: 'flex',
              flexDirection: 'column',
              gap: 4,
            }}
          >
            <Input
              value={row.rate}
              placeholder="中行折算价"
              onChange={(event) =>
                updateDraft(row.fromCurrency, 'rate', event.target.value)
              }
            />
            <div
              style={{
                height: 20,
                display: 'flex',
                alignItems: 'center',
                overflow: 'hidden',
              }}
            >
              {offsetInfo ? (
                <Tag
                  color="blue"
                  style={{
                    fontSize: 11,
                    margin: 0,
                    lineHeight: '18px',
                    padding: '0 6px',
                  }}
                >
                  微调 {offsetInfo}
                </Tag>
              ) : (
                <Text type="secondary" style={{ fontSize: 11 }}>
                  原折算: {raw?.rate || '-'}
                </Text>
              )}
            </div>
          </div>
        );
      },
    },
    {
      title: '来源 / 换算路径',
      dataIndex: 'conversionPath',
      ellipsis: true,
      render: (text: string) => (
        <div
          style={{
            height: 56,
            display: 'flex',
            alignItems: 'center',
          }}
        >
          <Tooltip title={text} placement="topLeft">
            <span style={{ cursor: 'pointer' }}>{text}</span>
          </Tooltip>
        </div>
      ),
    },
  ];

  return (
    <Modal
      title={
        baseCurrency === 'CNY'
          ? '从中国银行同步周汇率'
          : `一键同步周汇率（本币 ${baseCurrency}）`
      }
      open={open}
      onCancel={onClose}
      width={1000}
      style={{ top: 28 }}
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
      <Flex vertical gap={12} style={{ width: '100%' }}>
        <Flex justify="space-between" align="center" wrap="wrap" gap={12}>
          <Space wrap align="center">
            <Text strong>目标生效周：</Text>
            <Tag
              color="blue"
              style={{ fontSize: 13, padding: '2px 8px', margin: 0 }}
            >
              本周（周一至周日）
            </Tag>
            <Button
              type="primary"
              ghost
              icon={<SyncOutlined />}
              loading={fetching}
              onClick={() => handleFetch()}
            >
              抓取牌价
            </Button>
          </Space>

          {draftRows.length > 0 && (
            <Space align="center" wrap>
              <Text strong>小数精度：</Text>
              <Radio.Group
                value={precision}
                onChange={(e) => handlePrecisionChange(e.target.value)}
                optionType="button"
                buttonStyle="solid"
                options={[
                  { label: '4 位（中行标准）', value: '4' },
                  { label: '3 位', value: '3' },
                  { label: '2 位（对账简化）', value: '2' },
                  { label: '原始抓取', value: 'raw' },
                ]}
              />
              <Button
                icon={<HistoryOutlined />}
                onClick={handleApplyLastMarkups}
                title="按上次发布的微调偏好（如 USD +0.02、EUR +0.02 等）自动计算加点"
              >
                套用上次微调
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={handleRestoreRaw}
                title="恢复为中行原始抓取牌价（清除所有加点微调）"
              >
                还原原价
              </Button>
            </Space>
          )}
        </Flex>

        {fetchError && (
          <Alert
            type="error"
            showIcon
            title="牌价抓取失败"
            description={`${fetchError}。请检查网络后重试，或使用「新建汇率」手工录入本周汇率，单据不会因此受阻。`}
          />
        )}

        {preview && (
          <>
            <Alert
              type={preview.fallbackUsed ? 'warning' : 'info'}
              showIcon={preview.fallbackUsed}
              icon={preview.fallbackUsed ? <WarningOutlined /> : undefined}
              title={`数据来源：${preview.source ?? '-'}（${weekRange}）`}
              description={
                preview.fallbackUsed
                  ? '主数据源暂不可用，已启用备选来源换算，请逐行核对后再发布。'
                  : '请核对各币种建议值。支持直接修改或「套用上次微调」加点，系统会自动记住本次微调幅度。确认无误后点击「确认发布」生效。'
              }
            />
            <Table<ExchangeRateSyncDraftRow>
              rowKey="fromCurrency"
              size="small"
              pagination={false}
              dataSource={draftRows}
              scroll={{ y: 400, x: 920 }}
              columns={columns}
            />
          </>
        )}
      </Flex>
    </Modal>
  );
}

export default ExchangeRateSyncModal;
