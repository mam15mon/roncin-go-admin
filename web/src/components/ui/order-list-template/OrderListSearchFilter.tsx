import {
  ClearOutlined,
  DownOutlined,
  SaveOutlined,
  SearchOutlined,
  SettingOutlined,
  UpOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Row,
  Select,
  Space,
} from 'antd';
import React, { useState } from 'react';
import { standardDateRangePresets } from '../date-presets';
import type { OrderListFilterParams } from './types';

const { RangePicker } = DatePicker;

export interface OrderListSearchFilterProps {
  onSearch: (values: OrderListFilterParams) => void;
  onReset: () => void;
  options?: {
    ports?: { label: string; value: string }[];
    airports?: { label: string; value: string }[];
    shippingLines?: { label: string; value: string }[];
    airlines?: { label: string; value: string }[];
    partners?: { label: string; value: string }[];
    users?: { label: string; value: string }[];
    departments?: { label: string; value: string }[];
  };
  loading?: boolean;
}

export function OrderListSearchFilter({
  onSearch,
  onReset,
  options = {},
  loading = false,
}: OrderListSearchFilterProps) {
  const [form] = Form.useForm();
  const [collapsed, setCollapsed] = useState(true);

  const handleFinish = (rawValues: any) => {
    const formatRange = (range?: any[]) =>
      range?.[0] && range[1]
        ? [range[0].format('YYYY-MM-DD'), range[1].format('YYYY-MM-DD')] as [string, string]
        : undefined;

    const values: OrderListFilterParams = {
      keyword: rawValues.keyword?.trim() || undefined,
      customerId: rawValues.customerId || undefined,
      shippingLineId: rawValues.shippingLineId || undefined,
      originLocationId: rawValues.originLocationId || undefined,
      destinationLocationId: rawValues.destinationLocationId || undefined,
      consignee: rawValues.consignee?.trim() || undefined,
      shipper: rawValues.shipper?.trim() || undefined,

      createdAtRange: formatRange(rawValues.createdAtRange),
      etaRange: formatRange(rawValues.etaRange),
      etdRange: formatRange(rawValues.etdRange),
      lockedAtRange: formatRange(rawValues.lockedAtRange),
      statusTimeRange: formatRange(rawValues.statusTimeRange),

      operatorId: rawValues.operatorId || undefined,
      salesId: rawValues.salesId || undefined,
      customerServiceId: rawValues.customerServiceId || undefined,
      creatorId: rawValues.creatorId || undefined,

      stage: rawValues.stage || undefined,
      shareStatus: rawValues.shareStatus || undefined,
      isLocked: rawValues.isLocked || undefined,
      tags: rawValues.tags || undefined,
    };

    onSearch(values);
  };

  const handleReset = () => {
    form.resetFields();
    onReset();
  };

  return (
    <Card
      bordered={false}
      style={{
        borderRadius: 8,
        border: '1px solid #f0f0f0',
        backgroundColor: '#ffffff',
        marginBottom: 12,
      }}
      styles={{ body: { padding: '14px 16px 8px' } }}
    >
      <Form form={form} layout="vertical" onFinish={handleFinish}>
        {/* 第一行：最常用高频搜索字段 */}
        <Row gutter={[12, 0]}>
          <Col xs={24} sm={12} md={6} lg={5}>
            <Form.Item name="keyword" label="单号/主单号/加拼主单号">
              <Input
                placeholder="输入订单号/提单号/柜号"
                allowClear
                prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} md={6} lg={5}>
            <Form.Item name="customerId" label="委托单位">
              <Select
                showSearch
                allowClear
                placeholder="选择/搜索委托客户"
                options={options.partners}
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item name="originLocationId" label="起运港 (POL)">
              <Select
                showSearch
                allowClear
                placeholder="选择起运港"
                options={options.ports || options.airports}
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} md={6} lg={4}>
            <Form.Item name="destinationLocationId" label="目的港 (POD)">
              <Select
                showSearch
                allowClear
                placeholder="选择目的港"
                options={options.ports || options.airports}
                filterOption={(input, option) =>
                  (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
                }
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={24} md={12} lg={6}>
            <Form.Item name="etdRange" label="ETD (预计离港日期)">
              <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        {/* 展开的更多专业筛选字段 */}
        {!collapsed && (
          <>
            {/* 时间与节点类 */}
            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="createdAtRange" label="创建时间">
                  <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="etaRange" label="ETA (预计到达日期)">
                  <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="lockedAtRange" label="订单锁定时间">
                  <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="statusTimeRange" label="订单状态时间">
                  <RangePicker presets={standardDateRangePresets} style={{ width: '100%' }} />
                </Form.Item>
              </Col>
            </Row>

            {/* 业务实体与单证类 */}
            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="shippingLineId" label="船公司/航空公司">
                  <Select
                    showSearch
                    allowClear
                    placeholder="选择船司或航司"
                    options={options.shippingLines || options.airlines}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="shipper" label="发货人简称">
                  <Input placeholder="输入发货人简称" allowClear />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="consignee" label="收货人简称">
                  <Input placeholder="输入收货人简称" allowClear />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="stage" label="业务进程">
                  <Select
                    allowClear
                    placeholder="全部进程"
                    options={[
                      { label: '全部进程', value: 'all' },
                      { label: '未退关 (进行中)', value: 'unreturned' },
                      { label: '已完结 (已归档)', value: 'completed' },
                      { label: '已退关 (已撤单)', value: 'returned' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            {/* 人员与组织架构类 */}
            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="operatorId" label="操作人员">
                  <Select
                    showSearch
                    allowClear
                    placeholder="选择操作责任人"
                    options={options.users}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="salesId" label="业务人员 (销售)">
                  <Select
                    showSearch
                    allowClear
                    placeholder="选择业务人员"
                    options={options.users}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="customerServiceId" label="客服人员">
                  <Select
                    showSearch
                    allowClear
                    placeholder="选择客服人员"
                    options={options.users}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="creatorId" label="订单创建人员">
                  <Select
                    showSearch
                    allowClear
                    placeholder="选择创建人"
                    options={options.users}
                  />
                </Form.Item>
              </Col>
            </Row>

            {/* 状态与标记类 */}
            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="shareStatus" label="分享状态">
                  <Select
                    allowClear
                    placeholder="全部状态"
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '已分享', value: 'shared' },
                      { label: '未分享', value: 'unshared' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Form.Item name="isLocked" label="是否锁定">
                  <Select
                    allowClear
                    placeholder="全部"
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '已锁定', value: 'locked' },
                      { label: '未锁定', value: 'unlocked' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={24} md={12}>
                <Form.Item name="tags" label="订单标签">
                  <Select
                    mode="tags"
                    allowClear
                    placeholder="输入或选择标签（如：VIP、高货值、电商快船）"
                    options={[
                      { label: 'VIP 客户', value: 'VIP' },
                      { label: '高货值', value: 'HIGH_VALUE' },
                      { label: '危险品 DG', value: 'DG' },
                      { label: '快船专线', value: 'EXPRESS' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 筛选操作控制栏 */}
        <Row justify="space-between" align="middle" style={{ marginTop: 4, marginBottom: 4 }}>
          <Col>
            <Space size="middle">
              <Button
                type="link"
                size="small"
                icon={<SettingOutlined />}
                style={{ paddingLeft: 0 }}
                onClick={() => {}}
              >
                查询设置
              </Button>
              <Button
                type="link"
                size="small"
                icon={<SaveOutlined />}
                onClick={() => {}}
              >
                设为默认
              </Button>
            </Space>
          </Col>
          <Col>
            <Space size="small">
              <Button icon={<ClearOutlined />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                htmlType="submit"
                loading={loading}
              >
                查询
              </Button>
              <Button
                type="link"
                onClick={() => setCollapsed(!collapsed)}
                icon={collapsed ? <DownOutlined /> : <UpOutlined />}
              >
                {collapsed ? '展开更多筛选' : '收起筛选'}
              </Button>
            </Space>
          </Col>
        </Row>
      </Form>
    </Card>
  );
}

export default OrderListSearchFilter;
