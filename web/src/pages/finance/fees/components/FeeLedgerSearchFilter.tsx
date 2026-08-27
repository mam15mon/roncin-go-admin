import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Divider,
  Form,
  Input,
  Row,
  Select,
  Space,
  Badge,
} from 'antd';
import {
  DownOutlined,
  ReloadOutlined,
  SearchOutlined,
  UpOutlined,
} from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import { QuickDateRangePicker } from '@/components/ui';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';

export interface FeeLedgerFilterParams {
  keyword?: string;
  direction?: string;
  financialProgress?: string;
  status?: string;
  settlementPartyId?: string;
  expenseDateRange?: [Dayjs, Dayjs];

  // 单据与往来
  customerId?: string;
  orderNo?: string;
  masterNo?: string;
  houseNo?: string;
  billNo?: string;
  feeName?: string;
  invoiceNo?: string;
  consignee?: string;
  shipper?: string;

  // 航次与人员
  businessType?: string;
  currency?: string;
  etdRange?: [Dayjs, Dayjs];
  etaRange?: [Dayjs, Dayjs];
  salesName?: string;
  operatorName?: string;
  csName?: string;
  vesselName?: string;
  voyageNo?: string;

  // 账期与审计节点
  invoiceDateRange?: [Dayjs, Dayjs];
  verificationDateRange?: [Dayjs, Dayjs];
  orderCreatedAtRange?: [Dayjs, Dayjs];
  billCreatedAtRange?: [Dayjs, Dayjs];

  // 合约风控与标签
  isReconciled?: string;
  isLocked?: string;
  contractNo?: string;
  feeCategory?: string;
  serviceType?: string;
  feeTags?: string;
  billTags?: string;
}

export interface FeeLedgerSearchFilterProps {
  onSearch: (values: FeeLedgerFilterParams) => void;
  onReset: () => void;
  loading?: boolean;
}

export const FeeLedgerSearchFilter: React.FC<FeeLedgerSearchFilterProps> = ({
  onSearch,
  onReset,
  loading = false,
}) => {
  const [form] = Form.useForm<FeeLedgerFilterParams>();
  const [collapsed, setCollapsed] = useState<boolean>(true);
  const [partnerOptions, setPartnerOptions] = useState<
    { label: string; value: string }[]
  >([]);

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
    handlePartnerSearch();
  }, []);

  const handleFinish = (values: FeeLedgerFilterParams) => {
    onSearch(values);
  };

  const handleResetClick = () => {
    form.resetFields();
    onReset();
  };

  // 统计已填写的扩展筛选条件数
  const formValues = Form.useWatch([], form);
  const activeExtraCount = useMemo(() => {
    if (!formValues) return 0;
    let count = 0;
    const extraKeys: (keyof FeeLedgerFilterParams)[] = [
      'customerId',
      'orderNo',
      'masterNo',
      'houseNo',
      'billNo',
      'feeName',
      'invoiceNo',
      'consignee',
      'shipper',
      'businessType',
      'currency',
      'etdRange',
      'etaRange',
      'salesName',
      'operatorName',
      'csName',
      'vesselName',
      'voyageNo',
      'invoiceDateRange',
      'verificationDateRange',
      'orderCreatedAtRange',
      'billCreatedAtRange',
      'isReconciled',
      'isLocked',
      'contractNo',
      'feeCategory',
      'serviceType',
      'feeTags',
      'billTags',
    ];
    extraKeys.forEach((k) => {
      const val = formValues[k];
      if (val !== undefined && val !== null && val !== '' && val !== 'ALL') {
        count++;
      }
    });
    return count;
  }, [formValues]);

  return (
    <Card
      size="small"
      style={{
        marginBottom: 12,
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
      }}
      bodyStyle={{ padding: '12px 16px 8px' }}
    >
      <Form
        form={form}
        layout="horizontal"
        onFinish={handleFinish}
        initialValues={{}}
      >
        {/* ================= 第一行：首屏核心 6 项与主操作按钮 ================= */}
        <Row gutter={[12, 8]} align="middle">
          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item
              name="keyword"
              label="综合搜索"
              labelCol={{ flex: '80px' }}
              style={{ marginBottom: 8 }}
            >
              <Input
                allowClear
                placeholder="订单/主单/单位/账单号"
                onPressEnter={form.submit}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12} md={6} lg={3}>
            <Form.Item
              name="direction"
              label="属性"
              labelCol={{ flex: '50px' }}
              style={{ marginBottom: 8 }}
            >
              <Select
                allowClear
                placeholder="全部属性"
                options={[
                  { label: '应收', value: 'RECEIVABLE' },
                  { label: '应付', value: 'PAYABLE' },
                ]}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item
              name="financialProgress"
              label="财务进度"
              labelCol={{ flex: '80px' }}
              style={{ marginBottom: 8 }}
            >
              <Select
                allowClear
                placeholder="全部财务进度"
                options={[
                  { label: '账单未建立', value: 'UNBILLED' },
                  { label: '未核销未开票', value: 'UNVERIFIED_UNINVOICED' },
                  { label: '已开票未核销', value: 'INVOICED_UNVERIFIED' },
                  { label: '已开票部分核销', value: 'INVOICED_PARTIALLY_VERIFIED' },
                  { label: '部分核销未开票', value: 'PARTIALLY_VERIFIED_UNINVOICED' },
                  { label: '已核销未开票', value: 'VERIFIED_UNINVOICED' },
                  { label: '已完成', value: 'COMPLETED' },
                ]}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12} md={6} lg={3}>
            <Form.Item
              name="status"
              label="状态"
              labelCol={{ flex: '50px' }}
              style={{ marginBottom: 8 }}
            >
              <Select
                allowClear
                placeholder="全部状态"
                options={[
                  { label: '草稿', value: 'DRAFT' },
                  { label: '已确认', value: 'CONFIRMED' },
                  { label: '已开账', value: 'BILLED' },
                  { label: '已作废', value: 'CANCELLED' },
                ]}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item
              name="settlementPartyId"
              label="结算单位"
              labelCol={{ flex: '80px' }}
              style={{ marginBottom: 8 }}
            >
              <Select
                showSearch
                allowClear
                placeholder="输入名称/全拼搜索"
                options={partnerOptions}
                onSearch={handlePartnerSearch}
                filterOption={false}
              />
            </Form.Item>
          </Col>

          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item
              name="expenseDateRange"
              label="费用时间"
              labelCol={{ flex: '80px' }}
              style={{ marginBottom: 8 }}
            >
              <QuickDateRangePicker placeholder={['开始', '结束']} />
            </Form.Item>
          </Col>

          <Col xs={24} sm={24} md={12} lg={2} style={{ textAlign: 'right' }}>
            <Space style={{ marginBottom: 8 }}>
              <Button onClick={handleResetClick} icon={<ReloadOutlined />}>
                重置
              </Button>
              <Button
                type="primary"
                htmlType="submit"
                onClick={() => handleFinish(form.getFieldsValue())}
                icon={<SearchOutlined />}
                loading={loading}
              >
                查询
              </Button>
              <Button
                type="link"
                style={{ padding: 0 }}
                onClick={() => setCollapsed(!collapsed)}
              >
                {collapsed ? (
                  <>
                    展开
                    {activeExtraCount > 0 && (
                      <Badge
                        count={activeExtraCount}
                        size="small"
                        style={{ marginLeft: 4, backgroundColor: '#1677ff' }}
                      />
                    )}
                    <DownOutlined style={{ fontSize: 11, marginLeft: 4 }} />
                  </>
                ) : (
                  <>
                    收起
                    <UpOutlined style={{ fontSize: 11, marginLeft: 4 }} />
                  </>
                )}
              </Button>
            </Space>
          </Col>
        </Row>

        {/* ================= 展开后呈现的 33 项全维专业筛选项 ================= */}
        {!collapsed && (
          <div style={{ marginTop: 8 }}>
            <Divider style={{ margin: '8px 0 12px' }} />

            {/* 第 1 组：单据与往来实体 */}
            <div style={{ marginBottom: 6 }}>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: '#1677ff',
                  marginBottom: 8,
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <span
                  style={{
                    display: 'inline-block',
                    width: 3,
                    height: 12,
                    background: '#1677ff',
                    marginRight: 6,
                    borderRadius: 2,
                  }}
                />
                单据编号与往来实体
              </div>
              <Row gutter={[12, 4]}>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="customerId"
                    label="委托单位"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Select
                      showSearch
                      allowClear
                      placeholder="输入名称/全拼搜索"
                      options={partnerOptions}
                      onSearch={handlePartnerSearch}
                      filterOption={false}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="orderNo"
                    label="订单编号"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入订单编号" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="masterNo"
                    label="主提单号"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="海运MBL/空运AWB" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="houseNo"
                    label="分提单号"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入分单号 HBL" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="billNo"
                    label="账单编号"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入账单编号" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="feeName"
                    label="费用科目"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="如海运费/报关费" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="invoiceNo"
                    label="发票号码"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入发票号" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="consignee"
                    label="收货人"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入收货人抬头" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="shipper"
                    label="发货人"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入发货人抬头" />
                  </Form.Item>
                </Col>
              </Row>
            </div>

            {/* 第 2 组：航次船期与人员 */}
            <div style={{ marginBottom: 6 }}>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: '#1677ff',
                  marginBottom: 8,
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <span
                  style={{
                    display: 'inline-block',
                    width: 3,
                    height: 12,
                    background: '#1677ff',
                    marginRight: 6,
                    borderRadius: 2,
                  }}
                />
                航次船期与业务责任人
              </div>
              <Row gutter={[12, 4]}>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="businessType"
                    label="业务类型"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Select
                      allowClear
                      placeholder="全部类型"
                      options={[
                        { label: '海运出口', value: 'SE' },
                        { label: '海运进口', value: 'SI' },
                        { label: '空运出口', value: 'AE' },
                        { label: '空运进口', value: 'AI' },
                        { label: '铁运出口', value: 'RE' },
                        { label: '散货拼箱', value: 'LCL' },
                        { label: '其他业务', value: 'OTHER' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="currency"
                    label="计价币种"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Select
                      allowClear
                      placeholder="全部币种"
                      options={[
                        { label: 'CNY', value: 'CNY' },
                        { label: 'USD', value: 'USD' },
                        { label: 'EUR', value: 'EUR' },
                        { label: 'HKD', value: 'HKD' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="etdRange"
                    label="离港ETD"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="etaRange"
                    label="到港ETA"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="salesName"
                    label="业务人员"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入业务员姓名" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="operatorName"
                    label="操作人员"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入操作员姓名" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="csName"
                    label="客服人员"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入客服姓名" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="vesselName"
                    label="船名"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入船名" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="voyageNo"
                    label="航次"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入航次" />
                  </Form.Item>
                </Col>
              </Row>
            </div>

            {/* 第 3 组：账期、审计与风控 */}
            <div>
              <div
                style={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: '#1677ff',
                  marginBottom: 8,
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <span
                  style={{
                    display: 'inline-block',
                    width: 3,
                    height: 12,
                    background: '#1677ff',
                    marginRight: 6,
                    borderRadius: 2,
                  }}
                />
                账期审计、合约与风控标记
              </div>
              <Row gutter={[12, 4]}>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="invoiceDateRange"
                    label="开票时间"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="verificationDateRange"
                    label="核销时间"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="orderCreatedAtRange"
                    label="接单时间"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="billCreatedAtRange"
                    label="开账时间"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <QuickDateRangePicker placeholder={['开始', '结束']} />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="isReconciled"
                    label="是否对账"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Select
                      allowClear
                      placeholder="全部"
                      options={[
                        { label: '已对账', value: 'YES' },
                        { label: '未对账', value: 'NO' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="isLocked"
                    label="财务锁单"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Select
                      allowClear
                      placeholder="全部"
                      options={[
                        { label: '已锁单', value: 'YES' },
                        { label: '未锁单', value: 'NO' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="contractNo"
                    label="合约协议"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入合约协议号" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="feeTags"
                    label="费用标签"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入费用标签" />
                  </Form.Item>
                </Col>
                <Col xs={24} sm={12} md={6} lg={4}>
                  <Form.Item
                    name="billTags"
                    label="账单标签"
                    labelCol={{ flex: '80px' }}
                    style={{ marginBottom: 6 }}
                  >
                    <Input allowClear placeholder="输入账单标签" />
                  </Form.Item>
                </Col>
              </Row>
            </div>
          </div>
        )}
      </Form>
    </Card>
  );
};

export default FeeLedgerSearchFilter;
