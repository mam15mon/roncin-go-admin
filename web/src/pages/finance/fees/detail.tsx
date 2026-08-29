import { history, useParams } from '@umijs/max';
import {
  App,
  Button,
  Col,
  Descriptions,
  Input,
  Row,
  Spin,
  Table,
  Tag,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  EllipsisTooltip,
  PageHeaderShell,
  SectionCard,
} from '@/components/ui';
import { orderFeeServiceListFees } from '@/services/roncin/orderFeeService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';

const businessLabels: Record<number, string> = {
  1: '海运出口',
  2: '海运进口',
  3: '空运出口',
  4: '空运进口',
  5: '陆运',
  6: '铁路',
};

const businessRoutes: Record<number, string> = {
  1: 'sea-export',
  2: 'sea-import',
  3: 'air-export',
  4: 'air-import',
};

const statusLabels: Record<number, { text: string; color: string }> = {
  1: { text: '账单未建立', color: 'gold' },
  2: { text: '未核销未开票', color: 'orange' },
  3: { text: '已进账单', color: 'blue' },
  4: { text: '已作废', color: 'default' },
};

export default function FinanceFeeDetailPage() {
  const params = useParams<{ orderId: string }>();
  const orderId = params.orderId || '';
  const { message } = App.useApp();

  const [loading, setLoading] = useState(true);
  const [order, setOrder] = useState<API.Order>();
  const [customerName, setCustomerName] = useState<string>('');
  const [fees, setFees] = useState<API.OrderFee[]>([]);

  // 加载订单与费用数据
  const loadData = async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [orderRes, feesRes] = await Promise.all([
        orderServiceGetOrder({ id: orderId }),
        orderFeeServiceListFees({ orderId }),
      ]);
      const ord = orderRes.data;
      setOrder(ord);
      setFees(feesRes.data || []);

      if (ord?.customerId) {
        partnerServiceGetPartner({
          id: ord.customerId,
        }, { skipErrorHandler: true })
          .then((res) => {
            if (res.data) {
              setCustomerName(
                res.data.legalName || res.data.code || ord.customerId || '',
              );
            }
          })
          .catch(() => {
            setCustomerName('');
            message.warning('客户信息加载失败');
          });
      }
    } catch (err: any) {
      message.error(err.message || '加载费用详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [orderId]);

  // 1. 拆分 4 类费用（应收/应付）
  const receivableFees = useMemo(
    () => fees.filter((f) => Number(f.direction) === 1),
    [fees],
  );
  const specialReceivableFees = useMemo<API.OrderFee[]>(
    () => [], // 专项应收扩展
    [],
  );
  const payableFees = useMemo(
    () => fees.filter((f) => Number(f.direction) === 2),
    [fees],
  );
  const specialPayableFees = useMemo<API.OrderFee[]>(
    () => [], // 专项应付扩展
    [],
  );

  // 2. 多币种单票合计统计
  const statistics = useMemo(() => {
    let usdRec = 0;
    let usdPay = 0;
    let cnyRec = 0;
    let cnyPay = 0;

    let baseRec = 0;
    let basePay = 0;
    let baseRecNet = 0;
    let basePayNet = 0;

    fees.forEach((f) => {
      const amt = Number(f.totalAmount || 0);
      const baseAmt = Number(f.baseCurrencyAmount || 0);
      const taxRate = Number(f.taxRate || 0);
      const baseNetAmt =
        taxRate > 0 ? baseAmt / (1 + taxRate / 100) : baseAmt;

      if (Number(f.direction) === 1) {
        if (f.currency === 'USD') usdRec += amt;
        if (f.currency === 'CNY') cnyRec += amt;
        baseRec += baseAmt;
        baseRecNet += baseNetAmt;
      } else {
        if (f.currency === 'USD') usdPay += amt;
        if (f.currency === 'CNY') cnyPay += amt;
        basePay += baseAmt;
        basePayNet += baseNetAmt;
      }
    });

    const usdProfit = usdRec - usdPay;
    const cnyProfit = cnyRec - cnyPay;
    const baseProfit = baseRec - basePay;
    const baseProfitNet = baseRecNet - basePayNet;

    const taxMarginRate =
      baseRec > 0 ? `${((baseProfit / baseRec) * 100).toFixed(2)}%` : '0%';
    const netMarginRate =
      baseRecNet > 0
        ? `${((baseProfitNet / baseRecNet) * 100).toFixed(2)}%`
        : '0%';

    return {
      usdRec: usdRec.toFixed(2),
      usdPay: usdPay.toFixed(2),
      usdProfit: usdProfit.toFixed(2),
      cnyRec: cnyRec.toFixed(2),
      cnyPay: cnyPay.toFixed(2),
      cnyProfit: cnyProfit.toFixed(2),
      baseRec: baseRec.toFixed(2),
      basePay: basePay.toFixed(2),
      baseProfit: baseProfit.toFixed(2),
      taxMarginRate,
      baseRecNet: baseRecNet.toFixed(2),
      basePayNet: basePayNet.toFixed(2),
      baseProfitNet: baseProfitNet.toFixed(2),
      netMarginRate,
    };
  }, [fees]);

  // 计算表格底部汇总字符串（如 "513.75 USD + 1640.00 CNY"）
  const formatTableSummary = (list: API.OrderFee[]) => {
    const curMap: Record<string, number> = {};
    list.forEach((f) => {
      const cur = f.currency || 'CNY';
      curMap[cur] = (curMap[cur] || 0) + Number(f.totalAmount || 0);
    });
    const parts = Object.entries(curMap).map(
      ([cur, val]) => `${val.toFixed(2)} ${cur}`,
    );
    return parts.length > 0 ? parts.join(' + ') : '0.00 CNY';
  };

  // 标准明细表格列
  const feeColumns = [
    {
      title: '费用名称',
      dataIndex: 'feeName',
      key: 'feeName',
      width: 140,
      render: (val: any) => <strong style={{ color: '#262626' }}>{val}</strong>,
    },
    {
      title: '币种',
      dataIndex: 'currency',
      key: 'currency',
      width: 75,
      align: 'center' as const,
      render: (val: any) => <Tag style={{ margin: 0 }}>{val}</Tag>,
    },
    {
      title: '数量',
      dataIndex: 'quantity',
      key: 'quantity',
      width: 75,
      align: 'right' as const,
    },
    {
      title: '单价',
      dataIndex: 'unitPrice',
      key: 'unitPrice',
      width: 95,
      align: 'right' as const,
    },
    {
      title: '总价',
      dataIndex: 'totalAmount',
      key: 'totalAmount',
      width: 110,
      align: 'right' as const,
      render: (val: any, row: API.OrderFee) => (
        <strong
          style={{
            color: Number(row.direction) === 1 ? '#1677ff' : '#fa8c16',
          }}
        >
          {val}
        </strong>
      ),
    },
    {
      title: '汇率',
      dataIndex: 'exchangeRate',
      key: 'exchangeRate',
      width: 85,
      align: 'right' as const,
    },
    {
      title: '税率%',
      dataIndex: 'taxRate',
      key: 'taxRate',
      width: 75,
      align: 'right' as const,
      render: (val: any) => (val ? `${Number(val)}%` : '0%'),
    },
    {
      title: '税金',
      dataIndex: 'taxAmount',
      key: 'taxAmount',
      width: 90,
      align: 'right' as const,
    },
    {
      title: '不含税总价',
      dataIndex: 'netAmount',
      key: 'netAmount',
      width: 110,
      align: 'right' as const,
    },
    {
      title: '结算单位',
      dataIndex: 'settlementPartyName',
      key: 'settlementPartyName',
      width: 200,
      render: (val: any) => (
        <EllipsisTooltip maxWidth={190}>{val || '-'}</EllipsisTooltip>
      ),
    },
    {
      title: '费用状态',
      dataIndex: 'status',
      key: 'status',
      width: 110,
      render: (val: any) => {
        const st = statusLabels[Number(val) || 1] || {
          text: '账单未建立',
          color: 'default',
        };
        return (
          <Tag color={st.color} style={{ margin: 0 }}>
            {st.text}
          </Tag>
        );
      },
    },
    {
      title: '备注',
      dataIndex: 'note',
      key: 'note',
      width: 140,
      render: (val: any) => (
        <EllipsisTooltip maxWidth={130}>{val || '-'}</EllipsisTooltip>
      ),
    },
  ];

  // 渲染单张明细表格
  const renderFeeTable = (
    title: string,
    dataSource: API.OrderFee[],
    emptyText: string,
  ) => (
    <SectionCard title={title} style={{ marginBottom: 16 }}>
      <Table
        rowKey="id"
        size="small"
        bordered
        pagination={false}
        columns={feeColumns}
        dataSource={dataSource}
        locale={{ emptyText }}
        summary={() => (
          <Table.Summary fixed>
            <Table.Summary.Row style={{ background: '#fafafa', fontWeight: 600 }}>
              <Table.Summary.Cell index={0} colSpan={4}>
                总计金额(含税)
              </Table.Summary.Cell>
              <Table.Summary.Cell index={4} colSpan={8}>
                <span style={{ color: '#1677ff' }}>
                  {formatTableSummary(dataSource)}
                </span>
              </Table.Summary.Cell>
            </Table.Summary.Row>
          </Table.Summary>
        )}
      />
    </SectionCard>
  );

  const route = businessRoutes[order?.businessType || 0];

  return (
    <Spin spinning={loading}>
      <PageHeaderShell
        title={`订单 ${order?.orderNo || orderId} 费用详情`}
        subTitle="费用明细与单票多币种收支核算"
        breadcrumbs={[
          {
            label: '费用明细',
            onClick: () => history.push('/finance/fees'),
          },
          { label: '费用详情' },
        ]}
        extra={
          <div style={{ display: 'flex', gap: 8 }}>
            <Button key="refresh" onClick={loadData}>
              刷新数据
            </Button>
            {route && (
              <Button
                key="to-order"
                type="primary"
                onClick={() => history.push(`/orders/${route}/${orderId}`)}
              >
                查看业务订单
              </Button>
            )}
          </div>
        }
      />

      <div style={{ padding: '0 24px 32px' }}>
        {/* 1. 基础信息卡片 */}
        <SectionCard title="基础信息" style={{ marginBottom: 16 }}>
          <Descriptions
            size="small"
            bordered
            column={{ xs: 1, sm: 2, md: 3, lg: 5, xl: 5 }}
            styles={{
              label: {
                background: '#fafafa',
                width: '90px',
                fontSize: 12,
                fontWeight: 500,
              },
              content: { background: '#fff', fontSize: 12 },
            }}
          >
            <Descriptions.Item label="订单编号">
              {route ? (
                <a
                  style={{ fontWeight: 600 }}
                  onClick={() => history.push(`/orders/${route}/${orderId}`)}
                >
                  {order?.orderNo || orderId}
                </a>
              ) : (
                order?.orderNo || orderId
              )}
            </Descriptions.Item>
            <Descriptions.Item label="委托单位">
              <EllipsisTooltip maxWidth={160}>
                {customerName || order?.customerId || '-'}
              </EllipsisTooltip>
            </Descriptions.Item>
            <Descriptions.Item label="业务类型">
              {businessLabels[order?.businessType || 0] || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="服务类型">订舱</Descriptions.Item>
            <Descriptions.Item label="主单号">
              {order?.goodsDescription || '-'}
            </Descriptions.Item>

            <Descriptions.Item label="航班/船名">
              {order?.vesselVoyage || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="箱型箱量">
              {order?.totalPackages ? `${order.totalPackages} ${order.totalPackageUnit || '件'}` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="起运港">
              {order?.originLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目的港">
              {order?.destinationLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="航空/船公司">
              {order?.carrierId || '-'}
            </Descriptions.Item>

            <Descriptions.Item label="ETD/班期">
              {order?.etd || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="订舱代理">
              {order?.bookingAgentId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="业务编号">
              {order?.customerReferenceNo || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="下单时间">
              {order?.orderDate || order?.createdAt || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="合同号">
              {order?.contractNo || '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 2. 单票合计看板 */}
        <SectionCard title="单票合计" style={{ marginBottom: 16 }}>
          <div
            style={{
              background: '#fafafa',
              padding: 12,
              borderRadius: 6,
              border: '1px solid #f0f0f0',
            }}
          >
            <Row gutter={[12, 10]} align="middle">
              {/* 美金合计 */}
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>美金 应收(含税)</span>}
                  value={statistics.usdRec}
                  readOnly
                  style={{ fontWeight: 500 }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>美金 应付(含税)</span>}
                  value={statistics.usdPay}
                  readOnly
                  style={{ fontWeight: 500 }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>美金 毛利(含税)</span>}
                  value={statistics.usdProfit}
                  readOnly
                  style={{
                    fontWeight: 600,
                    color:
                      Number(statistics.usdProfit) >= 0 ? '#52c41a' : '#ff4d4f',
                  }}
                />
              </Col>
              <Col span={6} />

              {/* 人民币合计 */}
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>人民币 应收(含税)</span>}
                  value={statistics.cnyRec}
                  readOnly
                  style={{ fontWeight: 500 }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>人民币 应付(含税)</span>}
                  value={statistics.cnyPay}
                  readOnly
                  style={{ fontWeight: 500 }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>人民币 毛利(含税)</span>}
                  value={statistics.cnyProfit}
                  readOnly
                  style={{
                    fontWeight: 600,
                    color:
                      Number(statistics.cnyProfit) >= 0 ? '#52c41a' : '#ff4d4f',
                  }}
                />
              </Col>
              <Col span={6} />

              {/* 本币含税 */}
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 应收(含税)</span>}
                  value={statistics.baseRec}
                  readOnly
                  style={{ fontWeight: 600, color: '#1677ff' }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 应付(含税)</span>}
                  value={statistics.basePay}
                  readOnly
                  style={{ fontWeight: 600, color: '#fa8c16' }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 毛利(含税)</span>}
                  value={statistics.baseProfit}
                  readOnly
                  style={{
                    fontWeight: 700,
                    color:
                      Number(statistics.baseProfit) >= 0 ? '#52c41a' : '#ff4d4f',
                  }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>含税毛利率</span>}
                  value={statistics.taxMarginRate}
                  readOnly
                  style={{ fontWeight: 600 }}
                />
              </Col>

              {/* 本币不含税 */}
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 应收(不含税)</span>}
                  value={statistics.baseRecNet}
                  readOnly
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 应付(不含税)</span>}
                  value={statistics.basePayNet}
                  readOnly
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>本币 毛利(不含税)</span>}
                  value={statistics.baseProfitNet}
                  readOnly
                  style={{
                    fontWeight: 600,
                    color:
                      Number(statistics.baseProfitNet) >= 0
                        ? '#52c41a'
                        : '#ff4d4f',
                  }}
                />
              </Col>
              <Col span={6}>
                <Input
                  prefix={<span style={{ color: '#8c8c8c', fontSize: 12, marginRight: 6 }}>不含税毛利率</span>}
                  value={statistics.netMarginRate}
                  readOnly
                />
              </Col>
            </Row>
          </div>
        </SectionCard>

        {/* 3. 4 大类费用表格 */}
        {renderFeeTable('应收明细', receivableFees, '暂无应收明细')}
        {renderFeeTable(
          '专项应收明细',
          specialReceivableFees,
          '暂无专项应收明细',
        )}
        {renderFeeTable('应付明细', payableFees, '暂无应付明细')}
        {renderFeeTable(
          '专项应付明细',
          specialPayableFees,
          '暂无专项应付明细',
        )}
      </div>
    </Spin>
  );
}
