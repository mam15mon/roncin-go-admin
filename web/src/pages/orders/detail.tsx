import {
  DollarOutlined,
  FileDoneOutlined,
  LockOutlined,
  PaperClipOutlined,
  ReloadOutlined,
  UserOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { history, useAccess, useParams } from '@umijs/max';
import {
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Empty,
  Row,
  Space,
  Spin,
  Statistic,
  Table,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { PageHeaderShell, SectionCard } from '@/components/ui';
import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderAttachmentServiceListAttachments } from '@/services/roncin/orderAttachmentService';
import { orderCargoItemServiceListCargoItems } from '@/services/roncin/orderCargoItemService';
import { orderContainerServiceListContainers } from '@/services/roncin/orderContainerService';
import { orderMilestoneServiceListMilestones } from '@/services/roncin/orderMilestoneService';
import { orderPersonnelServiceListPersonnel } from '@/services/roncin/orderPersonnelService';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import { orderShippingDocumentServiceListShippingDocuments } from '@/services/roncin/orderShippingDocumentService';
import { partnerServiceListPartners } from '@/services/roncin/partnerService';
import AbnormalCasePanel, {
  type AbnormalCasePanelRef,
} from './abnormal-case-panel';
import {
  MASTER_DATA_KINDS,
  isMasterDataKind,
  orderPersonnelRoleValueEnum,
  parseOrderKind,
  paymentTermOptions,
  tradeTermOptions,
} from './common';
import OrderFeePanel, { type OrderFeePanelRef } from './order-fee-panel';
import ReleasePodPanel, { type ReleasePodPanelRef } from './release-pod-panel';

const { Text, Title } = Typography;

export default function OrderDetailPage() {
  const params = useParams<{ kind: string; id: string }>();
  const { message } = App.useApp();
  const access = useAccess();

  const kind = params.kind || 'sea-export';
  const orderId = params.id;
  const config = parseOrderKind(kind) || {
    kind: 'sea-export',
    title: '海运出口订单',
    businessType: 1,
    category: 'sea',
  };

  const [loading, setLoading] = useState(true);
  const [order, setOrder] = useState<API.Order>();
  const [milestones, setMilestones] = useState<API.OrderMilestone[]>([]);
  const [shippingDocs, setShippingDocs] = useState<API.OrderShippingDocument[]>([]);
  const [containers, setContainers] = useState<API.OrderContainer[]>([]);
  const [cargoItems, setCargoItems] = useState<API.OrderCargoItem[]>([]);
  const [personnel, setPersonnel] = useState<API.OrderPersonnel[]>([]);
  const [attachments, setAttachments] = useState<API.OrderAttachment[]>([]);

  const [masterOptions, setMasterOptions] = useState<API.MasterDataItem[]>([]);
  const [ports, setPorts] = useState<API.Port[]>([]);
  const [airports, setAirports] = useState<API.Airport[]>([]);
  const [partners, setPartners] = useState<API.Partner[]>([]);

  const releasePodPanelRef = useRef<ReleasePodPanelRef | null>(null);
  const abnormalCasePanelRef = useRef<AbnormalCasePanelRef | null>(null);
  const orderFeePanelRef = useRef<OrderFeePanelRef | null>(null);

  // 1. 加载订单完整数据
  const loadOrderDetail = async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const [
        orderRes,
        milestonesRes,
        docsRes,
        containersRes,
        cargoRes,
        personnelRes,
        attachmentsRes,
        masterOptionsRes,
        portsRes,
        airportsRes,
        partnersRes,
      ] = await Promise.all([
        orderServiceGetOrder({ id: orderId }),
        orderMilestoneServiceListMilestones({ orderId }).catch(() => ({ data: [] })),
        orderShippingDocumentServiceListShippingDocuments({ orderId }).catch(() => ({ data: [] })),
        orderContainerServiceListContainers({ orderId }).catch(() => ({ data: [] })),
        orderCargoItemServiceListCargoItems({ orderId }).catch(() => ({ data: [] })),
        orderPersonnelServiceListPersonnel({ orderId }).catch(() => ({ data: [] })),
        orderAttachmentServiceListAttachments({ orderId }).catch(() => ({ data: [] })),
        masterDataServiceListOptions().catch(() => ({ data: [] })),
        masterDataServiceListPorts({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
        masterDataServiceListAirports({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
        partnerServiceListPartners({ page: 1, pageSize: 100 }).catch(() => ({ data: [] })),
      ]);

      setOrder(orderRes.data);
      setMilestones(milestonesRes.data ?? []);
      setShippingDocs(docsRes.data ?? []);
      setContainers(containersRes.data ?? []);
      setCargoItems(cargoRes.data ?? []);
      setPersonnel(personnelRes.data ?? []);
      setAttachments(attachmentsRes.data ?? []);
      setMasterOptions(masterOptionsRes.data ?? []);
      setPorts(portsRes.data ?? []);
      setAirports(airportsRes.data ?? []);
      setPartners(partnersRes.data ?? []);
    } catch (error: any) {
      message.error(error.message || '加载订单详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadOrderDetail();
  }, [orderId]);

  // 主数据字典映射
  const partnerMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const p of partners) {
      if (p.id) {
        map[p.id] = p.legalName ? `${p.legalName} (${p.code})` : p.code || p.id;
      }
    }
    return map;
  }, [partners]);

  const locationMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const p of ports) {
      if (p.id) {
        map[p.id] = `${p.nameZh ? `${p.nameZh} / ` : ''}${p.nameEn} (${p.unLocode})`;
      }
    }
    for (const a of airports) {
      if (a.id) {
        map[a.id] = `${a.nameZh ? `${a.nameZh} / ` : ''}${a.nameEn} (${a.iataCode})`;
      }
    }
    return map;
  }, [ports, airports]);

  const containerSpecMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const item of masterOptions) {
      if (isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC) && item.id) {
        map[item.id] = item.code ? `${item.name} (${item.code})` : (item.name ?? '');
      }
    }
    return map;
  }, [masterOptions]);

  const serviceTypeMap = useMemo(() => {
    const map: Record<string, string> = {};
    for (const item of masterOptions) {
      if (isMasterDataKind(item.kind, MASTER_DATA_KINDS.SERVICE_TYPE) && item.id) {
        map[item.id] = item.name ?? item.code ?? '';
      }
    }
    return map;
  }, [masterOptions]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '120px 0', background: '#f5f7fa', minHeight: '100vh' }}>
        <Spin size="large" tip="正在加载订单详情..." />
      </div>
    );
  }

  if (!order) {
    return (
      <div style={{ padding: 48, background: '#f5f7fa', minHeight: '100vh' }}>
        <Card bordered={false} style={{ borderRadius: 8, textAlign: 'center', padding: 32 }}>
          <Empty description="未找到对应的订单档案" />
          <Button type="primary" onClick={() => history.push(`/orders/${kind}`)} style={{ marginTop: 16 }}>
            返回订单列表
          </Button>
        </Card>
      </div>
    );
  }

  const originName = locationMap[order.originLocationId ?? ''] || order.originLocationId || '-';
  const destName = locationMap[order.destinationLocationId ?? ''] || order.destinationLocationId || '-';
  const customerName = partnerMap[order.customerId ?? ''] || order.customerId || '-';

  return (
    <div style={{ minHeight: '100vh', backgroundColor: '#f5f7fa', paddingBottom: 64 }}>
      {/* 顶部 PageHeaderShell 吸顶导航 */}
      <PageHeaderShell
        title={
          <Space size={8}>
            <span style={{ fontSize: 20, fontWeight: 700, fontFamily: 'monospace', color: '#0f172a' }}>
              {order.orderNo || order.id}
            </span>
            <Tag color={order.status === 'COMPLETED' ? 'success' : order.status === 'DRAFT' ? 'default' : 'processing'}>
              {order.status || '正常运作'}
            </Tag>
            {order.canModify === false && order.status !== 'DRAFT' && (
              <Tag color="warning" icon={<LockOutlined />}>
                已锁定
              </Tag>
            )}
          </Space>
        }
        subTitle={`${config.title} · ${customerName} · ${originName} ➔ ${destName}`}
        breadcrumbs={[
          { label: '订单管理' },
          { label: config.title, onClick: () => history.push(`/orders/${kind}`) },
          { label: '订单详情' },
        ]}
        onBack={() => history.push(`/orders/${kind}`)}
        extra={[
          <Button
            key="refresh"
            icon={<ReloadOutlined />}
            onClick={() => loadOrderDetail()}
          >
            刷新
          </Button>,
          <Button
            key="fee"
            icon={<DollarOutlined />}
            onClick={() => orderFeePanelRef.current?.open(order)}
          >
            费用核算
          </Button>,
          <Button
            key="pod"
            icon={<FileDoneOutlined />}
            onClick={() => releasePodPanelRef.current?.open(order)}
          >
            放货凭证 (POD)
          </Button>,
          <Button
            key="abnormal"
            icon={<WarningOutlined />}
            danger
            onClick={() => abnormalCasePanelRef.current?.open(order)}
          >
            异常登记
          </Button>,
        ]}
      />

      <div style={{ padding: '0 24px', display: 'flex', flexDirection: 'column', gap: 16 }}>
        {/* 顶部关键指标统计栏 */}
        <Card bordered={false} style={{ borderRadius: 8, border: '1px solid #f0f0f0', backgroundColor: '#ffffff' }}>
          <Row gutter={[24, 16]}>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="业务类型"
                value={config.title}
                valueStyle={{ fontSize: 16, fontWeight: 600, color: '#1677ff' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="委托总件数"
                value={order.totalPackages ? `${order.totalPackages} ${order.totalPackageUnit || 'CTNS'}` : '-'}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="总毛重 (KGS)"
                value={order.totalGrossWeightKg ?? '-'}
                precision={2}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="总体积 (CBM)"
                value={order.totalVolumeCbm ?? '-'}
                precision={3}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="集装箱数量"
                value={containers.length > 0 ? `${containers.length} 柜` : (order.containerRequests?.length ? `${order.containerRequests.length} 计划` : '散货/拼箱')}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="创建时间"
                value={order.createdAt ? dayjs(order.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                valueStyle={{ fontSize: 14, color: '#64748b' }}
              />
            </Col>
          </Row>
        </Card>

        {/* 区块 1：委托单位与商务条款 */}
        <SectionCard title="基础委托与商务条款">
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3, lg: 4 }} bordered>
            <Descriptions.Item label="委托单位">{customerName}</Descriptions.Item>
            <Descriptions.Item label="客户业务编号">
              <Text copyable={Boolean(order.customerReferenceNo)}>{order.customerReferenceNo || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="企业内部编号">
              <Text copyable={Boolean(order.internalReferenceNo)}>{order.internalReferenceNo || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="贸易条款">
              {tradeTermOptions.find((o) => o.value === order.tradeTerm)?.label || order.tradeTerm || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="付款方式">
              {paymentTermOptions.find((o) => o.value === order.paymentTerm)?.label || order.paymentTerm || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="订舱代理">{partnerMap[order.bookingAgentId ?? ''] || order.bookingAgentId || '-'}</Descriptions.Item>
            <Descriptions.Item label="船公司 / 航司">{partnerMap[order.carrierId ?? ''] || order.carrierId || '-'}</Descriptions.Item>
            <Descriptions.Item label="合约号">{order.contractNo || '-'}</Descriptions.Item>
            <Descriptions.Item label="服务类型" span={2}>
              <Space wrap size={4}>
                {(order.serviceTypeIds ?? []).map((id) => (
                  <Tag key={id} color="blue">
                    {serviceTypeMap[id] || id}
                  </Tag>
                ))}
                {(!order.serviceTypeIds || order.serviceTypeIds.length === 0) && '-'}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="货物申报价值">
              {order.cargoValue ? `${order.cargoValue} ${order.cargoCurrency || 'USD'}` : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="保险保费">
              {order.insurancePremium ? `${order.insurancePremium} ${order.insuranceCurrency || 'CNY'}` : '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 区块 2：航程与船运配载 */}
        <SectionCard title="航程路线与节点截关时间">
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3, lg: 4 }} bordered>
            <Descriptions.Item label="起运港 (POL)">{originName}</Descriptions.Item>
            <Descriptions.Item label="目的港 (POD)">{destName}</Descriptions.Item>
            <Descriptions.Item label="卸货港 (POD)">
              {locationMap[order.dischargeLocationId ?? ''] || order.dischargeLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="中转港 (Transit)">
              {locationMap[order.transitLocationId ?? ''] || order.transitLocationId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="船名航次 / 航班号">
              <Text strong>{order.vesselVoyage || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="预计离港 (ETD)">
              {order.etd ? dayjs(order.etd).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="预计到港 (ETA)">
              {order.eta ? dayjs(order.eta).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="装货条款">{order.loadingTerms || '-'}</Descriptions.Item>
            <Descriptions.Item label="截补料时间 (SI)">
              {order.siCutoff ? dayjs(order.siCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截单证时间">
              {order.docCutoff ? dayjs(order.docCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截关时间 (Customs)">
              {order.customsCutoff ? dayjs(order.customsCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截VGM时间">
              {order.vgmCutoff ? dayjs(order.vgmCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 区块 3：提单与单证明细 */}
        <SectionCard
          title="提单与单证档案"
          extra={
            <Tag color="geekblue">共 {shippingDocs.length} 份单证</Tag>
          }
        >
          {shippingDocs.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无提单记录" />
          ) : (
            <Table
              size="small"
              dataSource={shippingDocs}
              rowKey="id"
              pagination={false}
              columns={[
                {
                  title: '主单号 (MBL)',
                  dataIndex: 'masterNo',
                  render: (v) => <Text copyable strong>{v || '-'}</Text>,
                },
                {
                  title: '分单号 (HBL)',
                  dataIndex: 'houseNo',
                  render: (v) => <Text copyable>{v || '-'}</Text>,
                },
                {
                  title: '主单单证类型',
                  dataIndex: 'masterDocumentType',
                  render: (v) => v || '-',
                },
                {
                  title: '主单签放方式',
                  dataIndex: 'masterReleaseMethod',
                  render: (v) => v || '-',
                },
                {
                  title: '分单签放方式',
                  dataIndex: 'releaseType',
                  render: (v) => v || '-',
                },
                {
                  title: '单证状态',
                  dataIndex: 'status',
                  render: (v) => <Tag color="blue">{v || '正常'}</Tag>,
                },
                {
                  title: '创建时间',
                  dataIndex: 'createdAt',
                  render: (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-'),
                },
              ]}
            />
          )}
        </SectionCard>

        {/* 区块 4：集装箱配载与货物明细 */}
        <SectionCard
          title="集装箱装载与货物明细"
          extra={
            <Tag color="cyan">集装箱：{containers.length} 柜 ｜ 货物条目：{cargoItems.length} 条</Tag>
          }
        >
          {/* 集装箱表格 */}
          <div style={{ marginBottom: 16 }}>
            <Title level={5} style={{ fontSize: 14, marginBottom: 8, color: '#334155' }}>
              集装箱装载明细
            </Title>
            {containers.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无集装箱数据" />
            ) : (
              <Table
                size="small"
                dataSource={containers}
                rowKey="id"
                pagination={false}
                columns={[
                  {
                    title: '箱号',
                    dataIndex: 'containerNo',
                    render: (v) => <Text copyable strong>{v || '-'}</Text>,
                  },
                  {
                    title: '铅封号',
                    dataIndex: 'sealNo',
                    render: (v) => v || '-',
                  },
                  {
                    title: '箱型规格',
                    dataIndex: 'containerSpecId',
                    render: (v) => containerSpecMap[v] || v || '-',
                  },
                  {
                    title: '毛重 (KGS)',
                    dataIndex: 'grossWeightKg',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '体积 (CBM)',
                    dataIndex: 'volumeCbm',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '备注说明',
                    dataIndex: 'note',
                    render: (v) => v || '-',
                  },
                ]}
              />
            )}
          </div>

          <Divider style={{ margin: '16px 0' }} />

          {/* 货物明细表格 */}
          <div>
            <Title level={5} style={{ fontSize: 14, marginBottom: 8, color: '#334155' }}>
              品名与包装明细
            </Title>
            {cargoItems.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无货物条目数据" />
            ) : (
              <Table
                size="small"
                dataSource={cargoItems}
                rowKey="id"
                pagination={false}
                columns={[
                  {
                    title: '货物名称 / 品名',
                    dataIndex: 'cargoName',
                    render: (v) => <Text strong>{v || '-'}</Text>,
                  },
                  {
                    title: '件数',
                    dataIndex: 'packageCount',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '毛重 (KGS)',
                    dataIndex: 'grossWeightKg',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '体积 (CBM)',
                    dataIndex: 'volumeCbm',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '净重 (KGS)',
                    dataIndex: 'netWeightKg',
                    render: (v) => v ?? '-',
                  },
                  {
                    title: '备注',
                    dataIndex: 'note',
                    render: (v) => v || '-',
                  },
                ]}
              />
            )}
          </div>
        </SectionCard>

        {/* 区块 5：履约里程碑节点 */}
        <SectionCard
          title="业务履约里程碑轨迹"
          extra={<Tag color="purple">已记录 {milestones.length} 个节点</Tag>}
        >
          {milestones.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无履约轨迹" />
          ) : (
            <Table
              size="small"
              dataSource={milestones}
              rowKey="id"
              pagination={false}
              columns={[
                {
                  title: '里程碑节点类型',
                  dataIndex: 'type',
                  render: (v) => <Tag color="blue">{v || '-'}</Tag>,
                },
                {
                  title: '发生时间',
                  dataIndex: 'occurredAt',
                  render: (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : <Text type="secondary">待达成</Text>),
                },
                {
                  title: '确认时间',
                  dataIndex: 'confirmedAt',
                  render: (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-'),
                },
                {
                  title: '备注说明',
                  dataIndex: 'note',
                  render: (v) => v || '-',
                },
              ]}
            />
          )}
        </SectionCard>

        {/* 区块 6：附件档案明细 */}
        <SectionCard
          title="附件档案明细"
          extra={<Tag color="volcano">共 {attachments.length} 个附件</Tag>}
        >
          {attachments.length === 0 ? (
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无上传附件" />
          ) : (
            <Table
              size="small"
              dataSource={attachments}
              rowKey="id"
              pagination={false}
              columns={[
                {
                  title: '文档类型',
                  dataIndex: 'docType',
                  render: (v) => <Tag color="geekblue">{v || '-'}</Tag>,
                },
                {
                  title: '文件名',
                  dataIndex: 'fileName',
                  render: (v) => (
                    <Space>
                      <PaperClipOutlined />
                      <Text strong>{v || '-'}</Text>
                    </Space>
                  ),
                },
                {
                  title: '文件大小',
                  dataIndex: 'fileSize',
                  render: (v) => (v ? `${v} 字节` : '-'),
                },
                {
                  title: 'MIME 类型',
                  dataIndex: 'mimeType',
                  render: (v) => v || '-',
                },
                {
                  title: '登记时间',
                  dataIndex: 'createdAt',
                  render: (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-'),
                },
              ]}
            />
          )}
        </SectionCard>

        {/* 区块 7：操作干系人与备注 */}
        <SectionCard title="干系人员与内部备注">
          <Row gutter={24}>
            <Col xs={24} md={12}>
              <Descriptions size="small" column={1} bordered title="协作团队分配">
                {personnel.length === 0 ? (
                  <Descriptions.Item label="协作人员">未指派协作人员</Descriptions.Item>
                ) : (
                  personnel.map((p, idx) => (
                    <Descriptions.Item
                      key={p.id || idx}
                      label={orderPersonnelRoleValueEnum[p.role ?? 0]?.text || `角色 ${p.role}`}
                    >
                      <Space>
                        <UserOutlined />
                        <span>{p.userId || '-'}</span>
                      </Space>
                    </Descriptions.Item>
                  ))
                )}
              </Descriptions>
            </Col>
            <Col xs={24} md={12}>
              <Descriptions size="small" column={1} bordered title="业务备注信息">
                <Descriptions.Item label="订舱备注">{order.bookingNotes || '-'}</Descriptions.Item>
                <Descriptions.Item label="配舱备注">{order.allocationNotes || '-'}</Descriptions.Item>
                <Descriptions.Item label="操作备注">{order.operationNotes || '-'}</Descriptions.Item>
                <Descriptions.Item label="综合备注">{order.notes || '-'}</Descriptions.Item>
              </Descriptions>
            </Col>
          </Row>
        </SectionCard>
      </div>

      {/* 挂载功能弹窗 */}
      <ReleasePodPanel
        ref={releasePodPanelRef}
        canManage={access.canOrder(config.businessType, 'release_pod.create')}
      />
      <OrderFeePanel ref={orderFeePanelRef} />
      <AbnormalCasePanel
        ref={abnormalCasePanelRef}
        canManage={access.canOrder(config.businessType, 'abnormal_case.create')}
        masterOptions={masterOptions}
      />
    </div>
  );
}
