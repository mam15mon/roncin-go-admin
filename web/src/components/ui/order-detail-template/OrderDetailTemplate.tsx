import {
  ClockCircleOutlined,
  DollarOutlined,
  EditOutlined,
  FileDoneOutlined,
  FlagOutlined,
  HistoryOutlined,
  LockOutlined,
  ReloadOutlined,
  UserOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  Col,
  Descriptions,
  Row,
  Space,
  Spin,
  Steps,
  Table,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React from 'react';
import { PageHeaderShell, SectionCard } from '../page-shell';
import type { OrderDetailTemplateProps, ProcessNodeItem } from './types';

const { Text, Paragraph } = Typography;

const defaultProcessNodes: ProcessNodeItem[] = [
  { key: 'booked', title: '已订舱', status: 'finish' },
  { key: 'allocated', title: '已配舱', status: 'finish' },
  { key: 'truck_arranged', title: '拖车已安排', status: 'process' },
  { key: 'doc_cutoff', title: '已截单', status: 'wait' },
  { key: 'customs_arranged', title: '报关已安排', status: 'wait' },
  { key: 'released', title: '已签单', status: 'wait' },
];

export function OrderDetailTemplate({
  orderKind: _orderKind = 'sea-export',
  title = '订单详情',
  data,
  loading = false,
  onBack,
  onRefresh,
  onEdit,
  onOpenFees,
  onOpenMilestones,
  onOpenReleasePod,
  onOpenAbnormal,
  extraActions = [],
  children,
}: OrderDetailTemplateProps) {
  if (loading) {
    return (
      <div
        style={{
          textAlign: 'center',
          padding: '120px 0',
          background: '#f5f7fa',
          minHeight: '100vh',
        }}
      >
        <Spin size="large" tip="正在加载订单档案..." />
      </div>
    );
  }

  const processNodes = data.processNodes && data.processNodes.length > 0 ? data.processNodes : defaultProcessNodes;
  const currentStep = data.currentStepIndex ?? processNodes.findIndex((n) => n.status === 'process');
  const operationLogs = data.operationLogs || [];
  const shippingDocs = data.shippingDocuments || [];
  const containers = data.containers || [];
  const cargoItems = data.cargoItems || [];

  return (
    <div
      style={{
        minHeight: '100vh',
        backgroundColor: '#f5f7fa',
        paddingBottom: 64,
      }}
    >
      {/* 顶部 PageHeaderShell 吸顶导航 */}
      <PageHeaderShell
        title={
          <Space size={8}>
            <span
              style={{
                fontSize: 20,
                fontWeight: 700,
                fontFamily: 'monospace',
                color: '#0f172a',
              }}
            >
              {data.orderNo || data.id}
            </span>
            <Tag
              color={
                data.status === 'COMPLETED'
                  ? 'success'
                  : data.status === 'DRAFT'
                    ? 'default'
                    : 'processing'
              }
            >
              {data.status || '正常运作'}
            </Tag>
            {(data.isLocked || (data.canModify === false && data.status !== 'DRAFT')) && (
              <Tag color="warning" icon={<LockOutlined />}>
                已锁定
              </Tag>
            )}
          </Space>
        }
        subTitle={`${data.businessTypeTitle || title} · ${data.customerName || '-'} · ${data.originName || '-'} ➔ ${data.destinationName || '-'}`}
        breadcrumbs={[
          { label: '订单管理' },
          { label: data.businessTypeTitle || title, onClick: onBack },
          { label: '订单详情' },
        ]}
        onBack={onBack}
        extra={[
          onRefresh && (
            <Button key="refresh" icon={<ReloadOutlined />} onClick={onRefresh}>
              刷新
            </Button>
          ),
          onEdit && (data.canModify || data.status === 'DRAFT') && (
            <Button
              key="edit"
              type="primary"
              icon={<EditOutlined />}
              onClick={onEdit}
            >
              编辑订单
            </Button>
          ),
          onOpenFees && (
            <Button key="fee" icon={<DollarOutlined />} onClick={onOpenFees}>
              费用核算
            </Button>
          ),
          onOpenMilestones && (
            <Button key="milestone" icon={<FlagOutlined />} onClick={onOpenMilestones}>
              履约里程碑
            </Button>
          ),
          onOpenReleasePod && (
            <Button key="pod" icon={<FileDoneOutlined />} onClick={onOpenReleasePod}>
              放货凭证 (POD)
            </Button>
          ),
          onOpenAbnormal && (
            <Button key="abnormal" icon={<WarningOutlined />} danger onClick={onOpenAbnormal}>
              异常登记
            </Button>
          ),
          ...extraActions,
        ].filter(Boolean)}
      />

      <div
        style={{
          padding: '0 24px',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
        }}
      >
        {/* 模块 1：订单状态（流程节点展示） */}
        <SectionCard title="订单状态流程" extra={<Tag color="blue">系统默认业务流程</Tag>}>
          <div style={{ padding: '8px 16px 4px' }}>
            <Steps
              current={currentStep >= 0 ? currentStep : 0}
              size="small"
              items={processNodes.map((node) => ({
                title: node.title,
                description: node.occurredAt
                  ? dayjs(node.occurredAt).format('YYYY-MM-DD HH:mm')
                  : node.description,
                status: node.status,
              }))}
            />
          </div>
        </SectionCard>

        {/* 模块 2：业务信息（基础委托与服务属性） */}
        <SectionCard title="业务信息">
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3, lg: 4 }} bordered>
            <Descriptions.Item label="订单编号">
              <Text copyable strong style={{ fontFamily: 'monospace' }}>
                {data.orderNo || data.id || '-'}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="委托单位">
              <Text strong>{data.customerName || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="集运 / 托运类型">
              {data.shipmentTypeName || '整箱 (FCL)'}
            </Descriptions.Item>
            <Descriptions.Item label="业务模式">
              {data.shipmentModeName || '跨境运输'}
            </Descriptions.Item>
            <Descriptions.Item label="贸易条款">
              <Tag color="cyan">{data.tradeTermName || '-'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="船公司 / 航司">
              {data.carrierName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="船代公司">
              {data.shippingAgentName || data.bookingAgentName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="合约号">
              {data.contractNo || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="服务类型" span={2}>
              <Space wrap size={4}>
                {(data.serviceTypeNames ?? []).map((name) => (
                  <Tag key={name} color="blue">
                    {name}
                  </Tag>
                ))}
                {(!data.serviceTypeNames || data.serviceTypeNames.length === 0) && '-'}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="货物品类">
              <Space wrap size={4}>
                {(data.cargoCategoryNames ?? []).map((name) => (
                  <Tag key={name} color="purple">
                    {name}
                  </Tag>
                ))}
                {(!data.cargoCategoryNames || data.cargoCategoryNames.length === 0) && '-'}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="申报货值 / 保费">
              {data.cargoValueWithCurrency || data.insurancePremiumWithCurrency
                ? `${data.cargoValueWithCurrency || '货值 -'} / ${data.insurancePremiumWithCurrency || '保费 -'}`
                : '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 模块 3：配舱信息（订舱与船期详情） */}
        <SectionCard
          title="配舱信息"
          extra={
            data.containerSummary && (
              <Tag color="volcano">箱型箱量：{data.containerSummary}</Tag>
            )
          }
        >
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3, lg: 4 }} bordered>
            <Descriptions.Item label="主单号 (MBL)">
              <Text copyable strong style={{ color: '#1677ff', fontFamily: 'monospace' }}>
                {data.masterBlNo || (shippingDocs[0]?.masterNo) || '-'}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="分单号 (HBL)">
              <Text copyable style={{ fontFamily: 'monospace' }}>
                {data.houseBlNo || (shippingDocs[0]?.houseNo) || '-'}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="箱型箱量">
              <Text strong>{data.containerSummary || (containers.length > 0 ? `${containers.length} 柜` : '-')}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="船名航次 / 航班">
              <Text strong>{data.vesselVoyage || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="起运港 (POL)">
              {data.originName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目的港 (POD)">
              {data.destinationName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="卸货港 (Discharge)">
              {data.dischargeName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="中转港 (Transit)">
              {data.transitName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="预计离港 (ETD)">
              {data.etd ? dayjs(data.etd).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="预计到港 (ETA)">
              {data.eta ? dayjs(data.eta).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截补料 (SI)">
              {data.siCutoff ? dayjs(data.siCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截单证时间">
              {data.docCutoff ? dayjs(data.docCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截关时间 (Customs)">
              {data.customsCutoff ? dayjs(data.customsCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截VGM时间">
              {data.vgmCutoff ? dayjs(data.vgmCutoff).format('YYYY-MM-DD HH:mm') : '-'}
            </Descriptions.Item>
          </Descriptions>

          {/* 集装箱装载子表（如有） */}
          {containers.length > 0 && (
            <div style={{ marginTop: 12 }}>
              <Table
                size="small"
                dataSource={containers}
                rowKey="id"
                pagination={false}
                columns={[
                  { title: '箱号', dataIndex: 'containerNo', render: (v) => <Text copyable strong>{v || '-'}</Text> },
                  { title: '铅封号', dataIndex: 'sealNo', render: (v) => v || '-' },
                  { title: '箱型规格', dataIndex: 'containerSpecName', render: (v) => v || '-' },
                  { title: '毛重 (KGS)', dataIndex: 'grossWeightKg', render: (v) => v ?? '-' },
                  { title: '体积 (CBM)', dataIndex: 'volumeCbm', render: (v) => v ?? '-' },
                  { title: '备注', dataIndex: 'note', render: (v) => v || '-' },
                ]}
              />
            </div>
          )}
        </SectionCard>

        {/* 模块 4：提单信息（发收通、唛头、品名、件重尺、运输条款及提单形式） */}
        <SectionCard
          title="提单信息"
          extra={
            <Tag color="geekblue">
              件重尺：{data.totalPackages ? `${data.totalPackages} ${data.packageUnit || 'CARTONS'}` : '-'} / {data.grossWeightKg ?? '-'} KGS / {data.volumeCbm ?? '-'} CBM
            </Tag>
          }
        >
          <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3 }} bordered style={{ marginBottom: 12 }}>
            <Descriptions.Item label="发货人 (Shipper)" span={3}>
              <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
                {data.shipperName || 'N/M (未填报或同主单)'}
              </Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="收货人 (Consignee)" span={3}>
              <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
                {data.consigneeName || 'TO ORDER'}
              </Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="通知人 (Notify Party)" span={3}>
              <Paragraph style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
                {data.notifyPartyName || 'SAME AS CONSIGNEE'}
              </Paragraph>
            </Descriptions.Item>
            <Descriptions.Item label="国外代理">
              {data.foreignAgentName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="运输条款">
              <Tag color="geekblue">{data.transportTerms || data.loadingTerms || 'CY-CY'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="提单形式">
              <Tag color="blue">{data.blTypeName || (shippingDocs[0]?.masterDocumentType) || '正本提单 (Original)'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="付款方式">
              {data.paymentTermName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="唛头 (Marks & Nos)" span={2}>
              {data.shippingMarks || 'N/M'}
            </Descriptions.Item>
            <Descriptions.Item label="货物英文品名" span={3}>
              <Text strong>{data.goodsEnglishDescription || data.goodsDescription || (cargoItems[0]?.cargoName) || '-'}</Text>
            </Descriptions.Item>
          </Descriptions>

          {/* 品名包装明细子表 */}
          {cargoItems.length > 0 && (
            <div>
              <Table
                size="small"
                dataSource={cargoItems}
                rowKey="id"
                pagination={false}
                columns={[
                  { title: '品名 (Cargo Name)', dataIndex: 'cargoName', render: (v) => <Text strong>{v || '-'}</Text> },
                  { title: '件数 (Packages)', dataIndex: 'packageCount', render: (v) => v ?? '-' },
                  { title: '毛重 (Gross Wt KGS)', dataIndex: 'grossWeightKg', render: (v) => v ?? '-' },
                  { title: '体积 (Volume CBM)', dataIndex: 'volumeCbm', render: (v) => v ?? '-' },
                  { title: '净重 (Net Wt KGS)', dataIndex: 'netWeightKg', render: (v) => v ?? '-' },
                  { title: '包装备注', dataIndex: 'note', render: (v) => v || '-' },
                ]}
              />
            </div>
          )}
        </SectionCard>

        {/* 模块 5：3 个备注（订舱备注、配舱备注、操作备注） */}
        <SectionCard title="业务与操作备注">
          <Row gutter={[16, 16]}>
            <Col xs={24} md={8}>
              <Card
                size="small"
                title={<Space><ClockCircleOutlined style={{ color: '#1677ff' }} /><span>订舱备注</span></Space>}
                style={{ height: '100%', backgroundColor: '#fcfcfc', border: '1px solid #f0f0f0' }}
              >
                <Paragraph style={{ minHeight: 48, margin: 0, color: '#475569' }}>
                  {data.bookingNotes || '暂无订舱备注'}
                </Paragraph>
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card
                size="small"
                title={<Space><FlagOutlined style={{ color: '#52c41a' }} /><span>配舱备注</span></Space>}
                style={{ height: '100%', backgroundColor: '#fcfcfc', border: '1px solid #f0f0f0' }}
              >
                <Paragraph style={{ minHeight: 48, margin: 0, color: '#475569' }}>
                  {data.allocationNotes || '暂无配舱备注'}
                </Paragraph>
              </Card>
            </Col>
            <Col xs={24} md={8}>
              <Card
                size="small"
                title={<Space><HistoryOutlined style={{ color: '#fa8c16' }} /><span>操作备注</span></Space>}
                style={{ height: '100%', backgroundColor: '#fcfcfc', border: '1px solid #f0f0f0' }}
              >
                <Paragraph style={{ minHeight: 48, margin: 0, color: '#475569' }}>
                  {data.operationNotes || data.notes || '暂无操作备注'}
                </Paragraph>
              </Card>
            </Col>
          </Row>
        </SectionCard>

        {/* 模块 6：内部信息与操作记录 */}
        <SectionCard title="内部信息与操作记录">
          <Row gutter={[24, 16]}>
            {/* 6.1 内部人员分工 */}
            <Col xs={24} lg={12}>
              <Descriptions size="small" column={{ xs: 1, sm: 2 }} bordered title="内部人员分工配置">
                <Descriptions.Item label="创建人员">
                  <Space><UserOutlined /><span>{data.creatorName || '系统建单'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="主操作员">
                  <Space><UserOutlined style={{ color: '#1677ff' }} /><span>{data.operatorName || '-'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="业务人员 (销售)">
                  <Space><UserOutlined style={{ color: '#52c41a' }} /><span>{data.salesName || '-'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="客服人员">
                  <Space><UserOutlined /><span>{data.customerServiceName || '-'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="单证人员">
                  <Space><UserOutlined /><span>{data.documentOperatorName || '-'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="商务人员">
                  <Space><UserOutlined /><span>{data.commercialOperatorName || '-'}</span></Space>
                </Descriptions.Item>
                <Descriptions.Item label="关联协作人员" span={2}>
                  {data.associateNames && data.associateNames.length > 0 ? (
                    <Space wrap size={4}>
                      {data.associateNames.map((n) => (
                        <Tag key={n}>{n}</Tag>
                      ))}
                    </Space>
                  ) : (
                    '-'
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="所属主体机构" span={2}>
                  {data.belongOrganizationName || '总部运营中心'}
                </Descriptions.Item>
              </Descriptions>
            </Col>

            {/* 6.2 操作记录日志 */}
            <Col xs={24} lg={12}>
              <Card
                size="small"
                title="操作记录与流转日志"
                style={{ height: '100%', border: '1px solid #f0f0f0' }}
              >
                {operationLogs.length === 0 ? (
                  <Timeline
                    style={{ marginTop: 12 }}
                    items={[
                      {
                        color: 'blue',
                        children: (
                          <div>
                            <Text strong>订单创建成功</Text>
                            <div style={{ fontSize: 12, color: '#94a3b8' }}>
                              {data.createdAt ? dayjs(data.createdAt).format('YYYY-MM-DD HH:mm:ss') : '初始建立'}
                            </div>
                          </div>
                        ),
                      },
                      {
                        color: 'green',
                        children: (
                          <div>
                            <Text strong>业务信息与配舱已录入</Text>
                            <div style={{ fontSize: 12, color: '#94a3b8' }}>
                              {data.updatedAt ? dayjs(data.updatedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
                            </div>
                          </div>
                        ),
                      },
                    ]}
                  />
                ) : (
                  <Timeline
                    style={{ marginTop: 12 }}
                    items={operationLogs.map((log) => ({
                      color: 'blue',
                      children: (
                        <div>
                          <Space size={8}>
                            <Text strong>{log.action}</Text>
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {log.operatorName || log.operatorUserId || '操作员'}
                            </Text>
                          </Space>
                          {log.detail && <div style={{ fontSize: 12, color: '#64748b' }}>{log.detail}</div>}
                          <div style={{ fontSize: 12, color: '#94a3b8' }}>
                            {log.occurredAt ? dayjs(log.occurredAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
                          </div>
                        </div>
                      ),
                    }))}
                  />
                )}
              </Card>
            </Col>
          </Row>
        </SectionCard>

        {/* 自定义扩展插槽 */}
        {children}
      </div>
    </div>
  );
}

export default OrderDetailTemplate;
