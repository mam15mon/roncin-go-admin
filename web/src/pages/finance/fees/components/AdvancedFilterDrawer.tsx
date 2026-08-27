import React, { useEffect, useState } from 'react';
import {
  Drawer,
  Form,
  Row,
  Col,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Divider,
} from 'antd';
import {
  FilterOutlined,
  ReloadOutlined,
  CheckOutlined,
  ClearOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { standardDateRangePresets } from '@/components/ui';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

const { RangePicker } = DatePicker;

export interface AdvancedFeeFilterValues {
  // === 第一组 ===
  expenseDateRange?: [dayjs.Dayjs, dayjs.Dayjs];
  direction?: string;
  settlementPartyId?: string;
  financialProgress?: string;
  invoiceDateRange?: [dayjs.Dayjs, dayjs.Dayjs];
  verificationDateRange?: [dayjs.Dayjs, dayjs.Dayjs];

  // === 第二组 ===
  orderOrMasterNo?: string;
  businessType?: string;
  customerId?: string;
  feeName?: string;
  orderCreatedAtRange?: [dayjs.Dayjs, dayjs.Dayjs];
  billCreatedAtRange?: [dayjs.Dayjs, dayjs.Dayjs];

  // === 第三组 ===
  etdRange?: [dayjs.Dayjs, dayjs.Dayjs];
  etaRange?: [dayjs.Dayjs, dayjs.Dayjs];
  currency?: string;
  salesName?: string;
  csName?: string;
  operatorName?: string;

  // === 第四组 ===
  isReconciled?: string;
  invoiceNo?: string;
  feeSource?: string;
  consignee?: string;
  isLocked?: string;
  shipper?: string;

  // === 第五组 ===
  vesselName?: string;
  voyageNo?: string;
  contractNo?: string;
  feeCategory?: string;
  serviceType?: string;
  billTags?: string;

  // === 第六组 ===
  matchedServiceType?: string;
  feeTags?: string;
  businessCode1?: string;
}

export const DEFAULT_FEE_FILTER_VALUES: AdvancedFeeFilterValues = {
  expenseDateRange: [dayjs('2026-02-26'), dayjs('2026-08-26')],
  direction: undefined,
  settlementPartyId: undefined,
  financialProgress: undefined,
  businessType: undefined,
  feeCategory: 'ALL',
};

interface AdvancedFilterDrawerProps {
  open: boolean;
  onClose: () => void;
  initialValues: AdvancedFeeFilterValues;
  onApply: (values: AdvancedFeeFilterValues) => void;
}

export function AdvancedFilterDrawer({
  open,
  onClose,
  initialValues,
  onApply,
}: AdvancedFilterDrawerProps) {
  const [form] = Form.useForm<AdvancedFeeFilterValues>();
  const [partnerOptions, setPartnerOptions] = useState<
    { label: string; value: string }[]
  >([]);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        ...DEFAULT_FEE_FILTER_VALUES,
        ...initialValues,
      });
    }
  }, [open, initialValues, form]);

  const handleReset = () => {
    form.resetFields();
    form.setFieldsValue(DEFAULT_FEE_FILTER_VALUES);
  };

  const handleClearAll = () => {
    form.resetFields();
  };

  const handleSubmit = () => {
    const values = form.getFieldsValue();
    onApply(values);
    onClose();
  };

  // 异步加载往来单位选项
  const handlePartnerSearch = async (keyword?: string) => {
    try {
      const res = await partnerServiceListPartners({
        page: 1,
        pageSize: 200,
        keyword,
      });
      const opts = (res.data || []).map((item) => ({
        label: item.legalName || item.code || item.id || '',
        value: item.id || '',
      }));
      setPartnerOptions(opts);
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    if (open && partnerOptions.length === 0) {
      handlePartnerSearch();
    }
  }, [open]);

  return (
    <Drawer
      open={open}
      onClose={onClose}
      width={780}
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <FilterOutlined style={{ color: '#1677ff', fontSize: 16 }} />
          <span style={{ fontWeight: 600, fontSize: 15 }}>
            集运费用明细 — 33 项全维高级业务筛选
          </span>
        </div>
      }
      extra={
        <Space size={8}>
          <Button icon={<ClearOutlined />} onClick={handleClearAll} size="small">
            清空所有
          </Button>
          <Button icon={<ReloadOutlined />} onClick={handleReset} size="small">
            恢复默认
          </Button>
        </Space>
      }
      footer={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '4px 0',
          }}
        >
          <span style={{ fontSize: 12, color: '#8c8c8c' }}>
            💡 支持任意维度单项或组合交叉穿透对账
          </span>
          <Space>
            <Button onClick={onClose}>取消</Button>
            <Button type="primary" icon={<CheckOutlined />} onClick={handleSubmit}>
              确定并执行筛选
            </Button>
          </Space>
        </div>
      }
    >
      <Form form={form} layout="vertical" size="small">
        {/* ================= 第一组：费用时间与收付账期 ================= */}
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第一组：费用时间与收付账期
          </div>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                label="费用时间"
                name="expenseDateRange"
                tooltip="费用实际发生的业务归属时间"
              >
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="费用属性" name="direction">
                <Select
                  allowClear
                  placeholder="请选择属性"
                  options={[
                    { label: '应收 (RECEIVABLE)', value: 'RECEIVABLE' },
                    { label: '应付 (PAYABLE)', value: 'PAYABLE' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="费用状态" name="financialProgress">
                <Select
                  allowClear
                  placeholder="请选择状态"
                  options={[
                    { label: '账单未建立', value: 'UNBILLED' },
                    { label: '未核销未开票', value: 'UNVERIFIED_UNINVOICED' },
                    { label: '已开票未核销', value: 'INVOICED_UNVERIFIED' },
                    { label: '已开票部分核销', value: 'INVOICED_PARTIALLY_VERIFIED' },
                    { label: '部分核销未开票', value: 'PARTIALLY_VERIFIED_UNINVOICED' },
                    { label: '已核销未开票', value: 'VERIFIED_UNINVOICED' },
                    { label: '已完成 (已开票已核销)', value: 'COMPLETED' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="结算单位" name="settlementPartyId">
                <Select
                  showSearch
                  allowClear
                  placeholder="输入名称/全拼/首字母搜索结算单位"
                  options={partnerOptions}
                  onSearch={handlePartnerSearch}
                  filterOption={false}
                />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="开票时间" name="invoiceDateRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="核销时间" name="verificationDateRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
          </Row>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* ================= 第二组：单据与业务实体 ================= */}
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第二组：单据与业务实体
          </div>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="订单/主单号/加拼主单号" name="orderOrMasterNo">
                <Input allowClear placeholder="输入订单号或主分提单号" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="业务类型" name="businessType">
                <Select
                  allowClear
                  placeholder="全部业务类型"
                  options={[
                    { label: '全部业务类型', value: '' },
                    { label: '海运出口 (SE)', value: 'SE' },
                    { label: '海运进口 (SI)', value: 'SI' },
                    { label: '空运出口 (AE)', value: 'AE' },
                    { label: '空运进口 (AI)', value: 'AI' },
                    { label: '陆运 (LAND)', value: 'LAND' },
                    { label: '铁路 (RAIL)', value: 'RAIL' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="委托单位" name="customerId">
                <Select
                  showSearch
                  allowClear
                  placeholder="输入名称/全拼搜索委托单位"
                  options={partnerOptions}
                  onSearch={handlePartnerSearch}
                  filterOption={false}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="费用名称" name="feeName">
                <Input allowClear placeholder="输入海运费/报关费/港杂费等" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="订单创建时间" name="orderCreatedAtRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="账单创建时间" name="billCreatedAtRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
          </Row>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* ================= 第三组：航次船期与人员 ================= */}
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第三组：航次船期与人员
          </div>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="ETD（预计离港时间）" name="etdRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="ETA（预计到港时间）" name="etaRange">
                <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} placeholder={['开始', '结束']} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="计价币种" name="currency">
                <Select
                  allowClear
                  placeholder="请选择币种"
                  options={[
                    { label: 'CNY 人民币', value: 'CNY' },
                    { label: 'USD 美元', value: 'USD' },
                    { label: 'EUR 欧元', value: 'EUR' },
                    { label: 'HKD 港币', value: 'HKD' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="业务人员" name="salesName">
                <Input allowClear placeholder="输入业务员姓名或部门" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="客服人员" name="csName">
                <Input allowClear placeholder="输入客服姓名或部门" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="操作人员" name="operatorName">
                <Input allowClear placeholder="输入操作员姓名或部门" />
              </Form.Item>
            </Col>
          </Row>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* ================= 第四组：对账与风控 ================= */}
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第四组：对账与风控
          </div>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="是否对账" name="isReconciled">
                <Select
                  allowClear
                  placeholder="请选择对账状态"
                  options={[
                    { label: '已对账', value: 'YES' },
                    { label: '未对账', value: 'NO' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="发票号" name="invoiceNo">
                <Input allowClear placeholder="输入完整或部分发票凭证号" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="费用来源" name="feeSource">
                <Select
                  allowClear
                  placeholder="请选择费用来源"
                  options={[
                    { label: '手动录入', value: 'MANUAL' },
                    { label: '自动计费', value: 'AUTO' },
                    { label: '简云同步', value: 'SYNC' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="收货人简称" name="consignee">
                <Input allowClear placeholder="输入收货人简称" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="单条费用锁定" name="isLocked">
                <Select
                  allowClear
                  placeholder="请选择锁定状态"
                  options={[
                    { label: '已锁定 (已封账)', value: 'LOCKED' },
                    { label: '未锁定 (可编辑)', value: 'UNLOCKED' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="发货人简称" name="shipper">
                <Input allowClear placeholder="输入发货人简称" />
              </Form.Item>
            </Col>
          </Row>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* ================= 第五组：航运工具与合约 ================= */}
        <div style={{ marginBottom: 18 }}>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第五组：航运工具与合约
          </div>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="船名" name="vesselName">
                <Input allowClear placeholder="输入船名英文或中文" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="航次/航班号/班列号" name="voyageNo">
                <Input allowClear placeholder="输入航次或航班代码" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="合约号" name="contractNo">
                <Input allowClear placeholder="输入船东/协议合约号" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="费用类别" name="feeCategory">
                <Select
                  allowClear
                  placeholder="全部类别"
                  options={[
                    { label: '全部类别', value: 'ALL' },
                    { label: '海运运费', value: 'OCEAN_FREIGHT' },
                    { label: '港口杂费', value: 'PORT_FEE' },
                    { label: '堆存堆场费', value: 'STORAGE_FEE' },
                    { label: '报关报检费', value: 'CUSTOMS_FEE' },
                    { label: '拖车陆运费', value: 'TRUCKING_FEE' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="服务类型" name="serviceType">
                <Select
                  allowClear
                  placeholder="请选择服务类型"
                  options={[
                    { label: 'CY-CY 场到场', value: 'CY-CY' },
                    { label: 'CFS-CFS 拼箱站到站', value: 'CFS-CFS' },
                    { label: 'DOOR-DOOR 门到门', value: 'DOOR-DOOR' },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="账单标签" name="billTags">
                <Input allowClear placeholder="模糊(或) 匹配账单标签" />
              </Form.Item>
            </Col>
          </Row>
        </div>

        <Divider style={{ margin: '12px 0' }} />

        {/* ================= 第六组：拓展属性与标签 ================= */}
        <div>
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              fontWeight: 600,
              fontSize: 13,
              color: '#1677ff',
              marginBottom: 10,
            }}
          >
            <span
              style={{
                display: 'inline-block',
                width: 3,
                height: 13,
                background: '#1677ff',
                marginRight: 6,
                borderRadius: 2,
              }}
            />
            第六组：拓展属性与标签
          </div>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="对应服务类型" name="matchedServiceType">
                <Input allowClear placeholder="输入对应服务类型" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="费用标签" name="feeTags">
                <Input allowClear placeholder="模糊(或) 匹配费用标签" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="业务代码一" name="businessCode1">
                <Input allowClear placeholder="输入业务归属代码" />
              </Form.Item>
            </Col>
          </Row>
        </div>
      </Form>
    </Drawer>
  );
}
