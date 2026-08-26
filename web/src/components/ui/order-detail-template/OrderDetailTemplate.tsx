import {
  DollarOutlined,
  EditOutlined,
  FileDoneOutlined,
  FlagOutlined,
  LockOutlined,
  PaperClipOutlined,
  ReloadOutlined,
  UserOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import {
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
import React from 'react';
import { PageHeaderShell, SectionCard } from '../page-shell';
import type { OrderDetailTemplateProps } from './types';

const { Text, Title } = Typography;

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

  const shippingDocs = data.shippingDocuments || [];
  const containers = data.containers || [];
  const cargoItems = data.cargoItems || [];
  const milestones = data.milestones || [];
  const attachments = data.attachments || [];
  const personnel = data.personnel || [];

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
            <Button
              key="fee"
              icon={<DollarOutlined />}
              onClick={onOpenFees}
            >
              费用核算
            </Button>
          ),
          onOpenMilestones && (
            <Button
              key="milestone"
              icon={<FlagOutlined />}
              onClick={onOpenMilestones}
            >
              履约里程碑
            </Button>
          ),
          onOpenReleasePod && (
            <Button
              key="pod"
              icon={<FileDoneOutlined />}
              onClick={onOpenReleasePod}
            >
              放货凭证 (POD)
            </Button>
          ),
          onOpenAbnormal && (
            <Button
              key="abnormal"
              icon={<WarningOutlined />}
              danger
              onClick={onOpenAbnormal}
            >
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
        {/* 顶部核心指标卡 */}
        <Card
          bordered={false}
          style={{
            borderRadius: 8,
            border: '1px solid #f0f0f0',
            backgroundColor: '#ffffff',
          }}
        >
          <Row gutter={[24, 16]}>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="业务品类"
                value={data.businessTypeTitle || title}
                valueStyle={{ fontSize: 16, fontWeight: 600, color: '#1677ff' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="委托总件数"
                value={
                  data.totalPackages
                    ? `${data.totalPackages} ${data.packageUnit || 'CTNS'}`
                    : '-'
                }
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="总毛重 (KGS)"
                value={data.grossWeightKg ?? '-'}
                precision={2}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="总体积 (CBM)"
                value={data.volumeCbm ?? '-'}
                precision={3}
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="集装箱数量"
                value={
                  containers.length > 0
                    ? `${containers.length} 柜`
                    : '散货 / 拼箱'
                }
                valueStyle={{ fontSize: 16, fontWeight: 600 }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title="创建时间"
                value={
                  data.createdAt
                    ? dayjs(data.createdAt).format('YYYY-MM-DD HH:mm')
                    : '-'
                }
                valueStyle={{ fontSize: 14, color: '#64748b' }}
              />
            </Col>
          </Row>
        </Card>

        {/* 区块 1：基础委托与商务条款 */}
        <SectionCard title="基础委托与商务条款">
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, md: 3, lg: 4 }}
            bordered
          >
            <Descriptions.Item label="委托单位">
              {data.customerName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="客户业务编号">
              <Text copyable={Boolean(data.customerReferenceNo)}>
                {data.customerReferenceNo || '-'}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="企业内部编号">
              <Text copyable={Boolean(data.internalReferenceNo)}>
                {data.internalReferenceNo || '-'}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="贸易条款">
              {data.tradeTermName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="付款方式">
              {data.paymentTermName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="订舱代理">
              {data.bookingAgentName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="船公司 / 航司">
              {data.carrierName || '-'}
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
                {(!data.serviceTypeNames || data.serviceTypeNames.length === 0) &&
                  '-'}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="货物申报价值">
              {data.cargoValueWithCurrency || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="保险保费">
              {data.insurancePremiumWithCurrency || '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 区块 2：航程路线与节点截关时间 */}
        <SectionCard title="航程路线与节点截关时间">
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, md: 3, lg: 4 }}
            bordered
          >
            <Descriptions.Item label="起运港 (POL)">
              {data.originName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="目的港 (POD)">
              {data.destinationName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="卸货港 (POD)">
              {data.dischargeName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="中转港 (Transit)">
              {data.transitName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="船名航次 / 航班号">
              <Text strong>{data.vesselVoyage || '-'}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="预计离港 (ETD)">
              {data.etd ? dayjs(data.etd).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="预计到港 (ETA)">
              {data.eta ? dayjs(data.eta).format('YYYY-MM-DD') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="装货条款">
              {data.loadingTerms || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截补料时间 (SI)">
              {data.siCutoff
                ? dayjs(data.siCutoff).format('YYYY-MM-DD HH:mm')
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截单证时间">
              {data.docCutoff
                ? dayjs(data.docCutoff).format('YYYY-MM-DD HH:mm')
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截关时间 (Customs)">
              {data.customsCutoff
                ? dayjs(data.customsCutoff).format('YYYY-MM-DD HH:mm')
                : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="截VGM时间">
              {data.vgmCutoff
                ? dayjs(data.vgmCutoff).format('YYYY-MM-DD HH:mm')
                : '-'}
            </Descriptions.Item>
          </Descriptions>
        </SectionCard>

        {/* 区块 3：提单与单证档案 */}
        <SectionCard
          title="提单与单证档案"
          extra={<Tag color="geekblue">共 {shippingDocs.length} 份单证</Tag>}
        >
          {shippingDocs.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无提单记录"
            />
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
                  render: (v) => (
                    <Text copyable strong>
                      {v || '-'}
                    </Text>
                  ),
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
                  render: (v) =>
                    v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-',
                },
              ]}
            />
          )}
        </SectionCard>

        {/* 区块 4：集装箱装载与货物明细 */}
        <SectionCard
          title="集装箱装载与货物明细"
          extra={
            <Tag color="cyan">
              集装箱：{containers.length} 柜 ｜ 货物条目：{cargoItems.length} 条
            </Tag>
          }
        >
          {/* 集装箱表格 */}
          <div style={{ marginBottom: 16 }}>
            <Title
              level={5}
              style={{ fontSize: 14, marginBottom: 8, color: '#334155' }}
            >
              集装箱装载明细
            </Title>
            {containers.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="暂无集装箱数据"
              />
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
                    render: (v) => (
                      <Text copyable strong>
                        {v || '-'}
                      </Text>
                    ),
                  },
                  {
                    title: '铅封号',
                    dataIndex: 'sealNo',
                    render: (v) => v || '-',
                  },
                  {
                    title: '箱型规格',
                    dataIndex: 'containerSpecName',
                    render: (v) => v || '-',
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
            <Title
              level={5}
              style={{ fontSize: 14, marginBottom: 8, color: '#334155' }}
            >
              品名与包装明细
            </Title>
            {cargoItems.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="暂无货物条目数据"
              />
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
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无履约轨迹"
            />
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
                  render: (v) =>
                    v ? (
                      dayjs(v).format('YYYY-MM-DD HH:mm:ss')
                    ) : (
                      <Text type="secondary">待达成</Text>
                    ),
                },
                {
                  title: '确认时间',
                  dataIndex: 'confirmedAt',
                  render: (v) =>
                    v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-',
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
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无上传附件"
            />
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
                  render: (v) =>
                    v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-',
                },
              ]}
            />
          )}
        </SectionCard>

        {/* 区块 7：操作干系人与备注 */}
        <SectionCard title="干系人员与内部备注">
          <Row gutter={24}>
            <Col xs={24} md={12}>
              <Descriptions
                size="small"
                column={1}
                bordered
                title="协作团队分配"
              >
                {personnel.length === 0 ? (
                  <Descriptions.Item label="协作人员">
                    未指派协作人员
                  </Descriptions.Item>
                ) : (
                  personnel.map((p, idx) => (
                    <Descriptions.Item
                      key={p.id || idx}
                      label={p.roleName || `角色 ${idx + 1}`}
                    >
                      <Space>
                        <UserOutlined />
                        <span>{p.userName || p.userId || '-'}</span>
                      </Space>
                    </Descriptions.Item>
                  ))
                )}
              </Descriptions>
            </Col>
            <Col xs={24} md={12}>
              <Descriptions
                size="small"
                column={1}
                bordered
                title="业务备注信息"
              >
                <Descriptions.Item label="订舱备注">
                  {data.bookingNotes || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="配舱备注">
                  {data.allocationNotes || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="操作备注">
                  {data.operationNotes || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="综合备注">
                  {data.notes || '-'}
                </Descriptions.Item>
              </Descriptions>
            </Col>
          </Row>
        </SectionCard>

        {/* 自定义扩展区块插槽 */}
        {children}
      </div>
    </div>
  );
}

export default OrderDetailTemplate;
