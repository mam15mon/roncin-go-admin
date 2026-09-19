import {
  ClearOutlined,
  DownOutlined,
  SearchOutlined,
  UpOutlined,
} from '@ant-design/icons';
import type { SelectProps } from 'antd';
import {
  App,
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
import type { Dayjs } from 'dayjs';
import React, { useEffect, useRef, useState } from 'react';
import { standardDateRangePresets } from '../date-presets';
import type {
  OrderListFilterOptions,
  OrderListFilterParams,
  OrderPersonnelFilterOption,
  OrderSelectOption,
} from './types';

const { RangePicker } = DatePicker;

export interface OrderListSearchFilterProps {
  onSearch: (values: OrderListFilterParams) => void;
  onReset: () => void;
  options?: OrderListFilterOptions;
  loading?: boolean;
}

interface RemoteSelectProps extends Omit<SelectProps<string>, 'options'> {
  options?: OrderSelectOption[];
  loadOptions?: (keyword?: string) => Promise<OrderSelectOption[]>;
}

function RemoteSelect({
  options = [],
  loadOptions,
  ...props
}: RemoteSelectProps) {
  const { message } = App.useApp();
  const [remoteOptions, setRemoteOptions] =
    useState<OrderSelectOption[]>(options);
  const [loading, setLoading] = useState(false);
  const loaderRef = useRef(loadOptions);
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const requestRef = useRef(0);

  useEffect(() => {
    loaderRef.current = loadOptions;
  }, [loadOptions]);

  useEffect(() => {
    if (!loaderRef.current) setRemoteOptions(options);
  }, [options]);

  const requestOptions = async (keyword = '') => {
    if (!loaderRef.current) return;
    const requestID = ++requestRef.current;
    setLoading(true);
    try {
      const result = await loaderRef.current(keyword);
      if (requestID === requestRef.current) setRemoteOptions(result);
    } catch (error) {
      if (requestID === requestRef.current) {
        message.error(
          error instanceof Error ? error.message : '筛选候选项加载失败',
        );
      }
    } finally {
      if (requestID === requestRef.current) setLoading(false);
    }
  };

  useEffect(() => {
    void requestOptions();
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      requestRef.current += 1;
    };
  }, []);

  return (
    <Select<string>
      {...props}
      allowClear
      loading={loading}
      options={loadOptions ? remoteOptions : options}
      showSearch={{
        filterOption: loadOptions
          ? false
          : (input, option) =>
              String(option?.label ?? '')
                .toLowerCase()
                .includes(input.toLowerCase()),
        onSearch: loadOptions
          ? (keyword) => {
              if (timerRef.current) clearTimeout(timerRef.current);
              timerRef.current = setTimeout(
                () => void requestOptions(keyword),
                250,
              );
            }
          : undefined,
      }}
    />
  );
}

interface PersonnelFilterProps {
  label: string;
  userField: keyof OrderListFilterParams;
  loadUsers?: (keyword?: string) => Promise<OrderSelectOption[]>;
}

function PersonnelFilter({
  label,
  userField,
  loadUsers,
}: PersonnelFilterProps) {
  return (
    <Form.Item label={label} name={userField}>
      <RemoteSelect placeholder="选择员工" loadOptions={loadUsers} />
    </Form.Item>
  );
}

export function OrderListSearchFilter({
  onSearch,
  onReset,
  options = {},
  loading = false,
}: OrderListSearchFilterProps) {
  const [form] = Form.useForm();
  const [collapsed, setCollapsed] = useState(true);
  const personnelCacheRef = useRef(
    new Map<string, Promise<OrderPersonnelFilterOption[]>>(),
  );

  const personnelLoader = options.loadPersonnel;
  const loadPersonnelUsers = personnelLoader
    ? async (keyword = '') => {
        let request = personnelCacheRef.current.get(keyword);
        if (!request) {
          request = personnelLoader(keyword);
          personnelCacheRef.current.set(keyword, request);
        }
        const result = await request;
        const seen = new Set<string>();
        return result
          .filter((item) => {
            if (seen.has(item.userId)) return false;
            seen.add(item.userId);
            return true;
          })
          .map((item) => ({ label: item.displayName, value: item.userId }));
      }
    : undefined;

  // biome-ignore lint/suspicious/noExplicitAny: 模板泛型默认/边界保持消费方零改动的宽松度；收紧需模板泛型化改造（后续任务）
  const handleFinish = (rawValues: Record<string, any>) => {
    const formatRange = (range?: [Dayjs, Dayjs]) =>
      range?.[0] && range[1]
        ? ([range[0].format('YYYY-MM-DD'), range[1].format('YYYY-MM-DD')] as [
            string,
            string,
          ])
        : undefined;
    const tagIds = (rawValues.tagIds as string[] | undefined)?.filter(Boolean);
    const numberKeyword = rawValues.numberKeyword?.trim() || undefined;

    onSearch({
      numberType: numberKeyword ? rawValues.numberType || 'order' : undefined,
      numberKeyword,
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
      stage: rawValues.stage === 'all' ? undefined : rawValues.stage,
      shareStatus:
        rawValues.shareStatus === 'all' ? undefined : rawValues.shareStatus,
      isLocked: rawValues.isLocked === 'all' ? undefined : rawValues.isLocked,
      tagIds: tagIds?.length ? tagIds : undefined,
    });
  };

  const filterRows = [
    ['操作人员', 'operatorId'],
    ['业务人员', 'salesId'],
    ['客服人员', 'customerServiceId'],
    ['订单创建人员', 'creatorId'],
  ] as const;

  return (
    <Card
      variant="borderless"
      style={{ borderRadius: 8, border: '1px solid #f0f0f0', marginBottom: 12 }}
      styles={{ body: { padding: '14px 16px 8px' } }}
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleFinish}
        initialValues={{
          numberType: 'order',
          shareStatus: 'all',
          isLocked: 'all',
        }}
      >
        <Row gutter={[12, 0]}>
          <Col xs={24} sm={12} lg={6}>
            <Form.Item label="复合单号">
              <Space.Compact block>
                <Form.Item name="numberType" noStyle>
                  <Select
                    style={{ width: 112 }}
                    options={[
                      { label: '订单号', value: 'order' },
                      { label: '主单号', value: 'master' },
                      { label: '加拼主单号', value: 'consolidated_master' },
                      { label: '客户业务号', value: 'customer_reference' },
                      { label: 'Booking No', value: 'booking' },
                    ]}
                  />
                </Form.Item>
                <Form.Item name="numberKeyword" noStyle>
                  <Input
                    allowClear
                    prefix={<SearchOutlined />}
                    placeholder="输入单号"
                  />
                </Form.Item>
              </Space.Compact>
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} lg={5}>
            <Form.Item name="customerId" label="委托单位">
              <RemoteSelect
                placeholder="输入代码、名称或别名检索"
                options={options.partners}
                loadOptions={options.loadPartners}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Form.Item name="originLocationId" label="起运港">
              <RemoteSelect
                placeholder="输入港口代码或名称"
                options={options.ports || options.airports}
                loadOptions={options.loadPorts}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} lg={4}>
            <Form.Item name="destinationLocationId" label="目的港">
              <RemoteSelect
                placeholder="输入港口代码或名称"
                options={options.ports || options.airports}
                loadOptions={options.loadPorts}
              />
            </Form.Item>
          </Col>
          <Col xs={24} sm={12} lg={5}>
            <Form.Item name="etdRange" label="ETD（预计开船时间）">
              <RangePicker
                presets={standardDateRangePresets}
                style={{ width: '100%' }}
              />
            </Form.Item>
          </Col>
        </Row>

        {!collapsed && (
          <>
            <Row gutter={[12, 0]}>
              {[
                ['创建时间', 'createdAtRange'],
                ['ETA（预计到港时间）', 'etaRange'],
                ['订单状态时间', 'statusTimeRange'],
                ['订单锁定时间', 'lockedAtRange'],
              ].map(([label, name]) => (
                <Col xs={24} sm={12} lg={6} key={name}>
                  <Form.Item name={name} label={label}>
                    <RangePicker
                      presets={standardDateRangePresets}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                </Col>
              ))}
            </Row>

            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} lg={6}>
                <Form.Item name="shippingLineId" label="船公司">
                  <RemoteSelect
                    placeholder="输入船公司代码或名称"
                    options={options.shippingLines}
                    loadOptions={options.loadCarriers}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Form.Item name="shipper" label="发货人简称">
                  <Input placeholder="输入发货人简称" allowClear />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Form.Item name="consignee" label="收货人简称">
                  <Input placeholder="输入收货人简称" allowClear />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Form.Item name="stage" label="进程">
                  <Select
                    allowClear
                    placeholder="全部进程"
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '未退关', value: 'unreturned' },
                      { label: '已退关', value: 'returned' },
                      { label: '已完结', value: 'completed' },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[12, 0]}>
              {filterRows.map(([label, userField]) => (
                <Col xs={24} sm={12} lg={6} key={userField}>
                  <PersonnelFilter
                    label={label}
                    userField={userField}
                    loadUsers={loadPersonnelUsers}
                  />
                </Col>
              ))}
            </Row>

            <Row gutter={[12, 0]}>
              <Col xs={24} sm={12} lg={4}>
                <Form.Item name="isLocked" label="是否锁定">
                  <Select
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '是', value: 'locked' },
                      { label: '否', value: 'unlocked' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} lg={4}>
                <Form.Item name="shareStatus" label="分享状态">
                  <Select
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '已分享', value: 'shared' },
                      { label: '未分享', value: 'unshared' },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col xs={24} sm={12} lg={11}>
                <Form.Item name="tagIds" label="订单标签">
                  <Select
                    mode="multiple"
                    allowClear
                    showSearch={{ optionFilterProp: 'label' }}
                    placeholder="选择标签筛选订单"
                    options={options.tags}
                  />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        <Row
          justify="end"
          align="middle"
          style={{ marginTop: 4, marginBottom: 4 }}
        >
          <Col>
            <Space size="small">
              <Button
                icon={<ClearOutlined />}
                onClick={() => {
                  form.resetFields();
                  onReset();
                }}
              >
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
                onClick={() => setCollapsed((value) => !value)}
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
