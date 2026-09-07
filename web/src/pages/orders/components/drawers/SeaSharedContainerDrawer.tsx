import {
  CheckCircleOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  PlusOutlined,
  RollbackOutlined,
  SaveOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Divider,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import Decimal from 'decimal.js';
import React, { useEffect, useMemo, useState } from 'react';
import {
  seaSharedContainerServiceConfirmSeaSharedContainer,
  seaSharedContainerServiceCreateSeaSharedContainer,
  seaSharedContainerServiceDeleteSeaSharedContainer,
  seaSharedContainerServiceListSeaSharedContainerCandidates,
  seaSharedContainerServiceListSeaSharedContainers,
  seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft,
  seaSharedContainerServiceWithdrawSeaSharedContainer,
} from '@/services/roncin/seaSharedContainerService';

const { Text, Title } = Typography;

export type SeaSharedContainerDrawerProps = {
  open: boolean;
  onClose: () => void;
  transportExecutionId?: string;
  orderId?: string;
  orderNo?: string;
  canManage?: boolean;
  containerSpecOptions?: { label: string; value: string | number }[];
};

type CargoAllocationItem = {
  orderId: string;
  orderNo: string;
  houseBillId: string;
  houseNo: string;
  orderVersion: string;
  linkVersion: string;
  houseBillVersion: string;
  cargoItemId: string;
  cargoName: string;
  cargoVersion: string;
  totalPackageCount: number;
  totalGrossWeightKg: string;
  totalVolumeCbm: string;
  packageCount: number;
  grossWeightKg: string;
  volumeCbm: string;
};

export default function SeaSharedContainerDrawer({
  open,
  onClose,
  transportExecutionId,
  orderId: _currentOrderId,
  orderNo: _currentOrderNo,
  canManage = true,
  containerSpecOptions = [],
}: SeaSharedContainerDrawerProps) {
  const { message } = App.useApp();

  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [containers, setContainers] = useState<API.SeaSharedContainer[]>([]);
  const [selectedContainerId, setSelectedContainerId] = useState<string | null>(
    null,
  );
  const [candidates, setCandidates] = useState<
    API.SeaSharedContainerCandidateOrder[]
  >([]);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createForm] = Form.useForm();

  // 本地分配矩阵：key = `${orderId}:${cargoItemId}`
  const [allocationMap, setAllocationMap] = useState<
    Record<
      string,
      { packageCount: number; grossWeightKg: string; volumeCbm: string }
    >
  >({});

  // 加载共享箱列表与候选订单
  const loadData = async () => {
    if (!transportExecutionId) return;
    setLoading(true);
    try {
      const [containersResp, candidatesResp] = await Promise.all([
        seaSharedContainerServiceListSeaSharedContainers({
          transportExecutionId,
          pageSize: 100,
        }),
        seaSharedContainerServiceListSeaSharedContainerCandidates({
          transportExecutionId,
        }),
      ]);

      const containerList = containersResp?.data || [];
      setContainers(containerList);
      setCandidates(candidatesResp?.data || []);

      if (containerList.length > 0) {
        if (
          !selectedContainerId ||
          !containerList.some((c) => c.id === selectedContainerId)
        ) {
          setSelectedContainerId(containerList[0].id || null);
        }
      } else {
        setSelectedContainerId(null);
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '加载共享箱数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open && transportExecutionId) {
      void loadData();
    }
  }, [open, transportExecutionId]);

  // 当前选中的共享箱
  const selectedContainer = useMemo(() => {
    return containers.find((c) => c.id === selectedContainerId) || null;
  }, [containers, selectedContainerId]);

  // 同步当前选中箱的已有分配到本地编辑状态
  useEffect(() => {
    if (!selectedContainer) {
      setAllocationMap({});
      return;
    }
    const map: Record<
      string,
      { packageCount: number; grossWeightKg: string; volumeCbm: string }
    > = {};
    if (selectedContainer.allocations) {
      for (const alloc of selectedContainer.allocations) {
        if (alloc.orderId && alloc.cargoItemId) {
          const key = `${alloc.orderId}:${alloc.cargoItemId}`;
          map[key] = {
            packageCount: alloc.packageCount || 0,
            grossWeightKg: alloc.grossWeightKg || '0.000',
            volumeCbm: alloc.volumeCbm || '0.000000',
          };
        }
      }
    }
    setAllocationMap(map);
  }, [selectedContainer]);

  // 展开候选订单与货物项供表格展示
  const flatCargoList = useMemo<CargoAllocationItem[]>(() => {
    const list: CargoAllocationItem[] = [];
    for (const order of candidates) {
      if (!order.cargoItems) continue;
      for (const cargo of order.cargoItems) {
        const key = `${order.orderId}:${cargo.id}`;
        const currentAlloc = allocationMap[key] || {
          packageCount: 0,
          grossWeightKg: '0.000',
          volumeCbm: '0.000000',
        };
        list.push({
          orderId: order.orderId || '',
          orderNo: order.orderNo || '',
          houseBillId: order.houseBillId || '',
          houseNo: order.houseNo || '',
          orderVersion: String(order.orderVersion || 1),
          linkVersion: String(order.linkVersion || 1),
          houseBillVersion: String(order.houseBillVersion || 1),
          cargoItemId: cargo.id || '',
          cargoName: cargo.cargoName || '未命名货物',
          cargoVersion: String(cargo.version || 1),
          totalPackageCount: cargo.packageCount || 0,
          totalGrossWeightKg: cargo.grossWeightKg || '0.000',
          totalVolumeCbm: cargo.volumeCbm || '0.000000',
          packageCount: currentAlloc.packageCount,
          grossWeightKg: currentAlloc.grossWeightKg,
          volumeCbm: currentAlloc.volumeCbm,
        });
      }
    }
    return list;
  }, [candidates, allocationMap]);

  // 本地实时汇总已分配件重尺与剩余
  const summary = useMemo(() => {
    if (!selectedContainer) {
      return {
        totalPackages: 0,
        totalWeight: '0.000',
        totalVolume: '0.000000',
        allocPackages: 0,
        allocWeight: '0.000',
        allocVolume: '0.000000',
        diffPackages: 0,
        diffWeight: '0.000',
        diffVolume: '0.000000',
        isBalanced: false,
      };
    }

    let allocPackages = 0;
    let allocWeight = new Decimal(0);
    let allocVolume = new Decimal(0);

    for (const item of Object.values(allocationMap)) {
      allocPackages += Number(item.packageCount || 0);
      allocWeight = allocWeight.plus(new Decimal(item.grossWeightKg || 0));
      allocVolume = allocVolume.plus(new Decimal(item.volumeCbm || 0));
    }

    const totalPackages = selectedContainer.packageCount || 0;
    const totalWeight = new Decimal(selectedContainer.grossWeightKg || 0);
    const totalVolume = new Decimal(selectedContainer.volumeCbm || 0);

    const diffPackages = totalPackages - allocPackages;
    const diffWeight = totalWeight.minus(allocWeight);
    const diffVolume = totalVolume.minus(allocVolume);

    const isBalanced =
      diffPackages === 0 && diffWeight.isZero() && diffVolume.isZero();

    return {
      totalPackages,
      totalWeight: totalWeight.toFixed(3),
      totalVolume: totalVolume.toFixed(6),
      allocPackages,
      allocWeight: allocWeight.toFixed(3),
      allocVolume: allocVolume.toFixed(6),
      diffPackages,
      diffWeight: diffWeight.toFixed(3),
      diffVolume: diffVolume.toFixed(6),
      isBalanced,
    };
  }, [selectedContainer, allocationMap]);

  // 处理分配更新
  const handleUpdateAllocation = (
    orderId: string,
    cargoItemId: string,
    field: 'packageCount' | 'grossWeightKg' | 'volumeCbm',
    value: any,
  ) => {
    const key = `${orderId}:${cargoItemId}`;
    setAllocationMap((prev) => {
      const existing = prev[key] || {
        packageCount: 0,
        grossWeightKg: '0.000',
        volumeCbm: '0.000000',
      };
      return {
        ...prev,
        [key]: {
          ...existing,
          [field]:
            field === 'packageCount'
              ? Number(value || 0)
              : String(value ?? '0.000'),
        },
      };
    });
  };

  // 快捷填入该货物的全部总件重尺
  const handleFillAllCargo = (record: CargoAllocationItem) => {
    const key = `${record.orderId}:${record.cargoItemId}`;
    setAllocationMap((prev) => ({
      ...prev,
      [key]: {
        packageCount: record.totalPackageCount,
        grossWeightKg: record.totalGrossWeightKg,
        volumeCbm: record.totalVolumeCbm,
      },
    }));
  };

  // 构建提交 payload
  const buildAllocationInputs = (): API.SeaSharedContainerAllocationInput[] => {
    const inputs: API.SeaSharedContainerAllocationInput[] = [];
    for (const record of flatCargoList) {
      const key = `${record.orderId}:${record.cargoItemId}`;
      const alloc = allocationMap[key];
      if (
        alloc &&
        (alloc.packageCount > 0 ||
          new Decimal(alloc.grossWeightKg || 0).gt(0) ||
          new Decimal(alloc.volumeCbm || 0).gt(0))
      ) {
        inputs.push({
          orderId: record.orderId,
          houseBillId: record.houseBillId,
          cargoItemId: record.cargoItemId,
          packageCount: alloc.packageCount,
          grossWeightKg: alloc.grossWeightKg,
          volumeCbm: alloc.volumeCbm,
          expectedOrderVersion: String(record.orderVersion),
          expectedLinkVersion: String(record.linkVersion),
          expectedHouseBillVersion: String(record.houseBillVersion),
          expectedCargoItemVersion: String(record.cargoVersion),
        });
      }
    }
    return inputs;
  };

  // 保存草稿
  const handleSaveDraft = async () => {
    if (!selectedContainer?.id) return;
    setSubmitting(true);
    try {
      const payload = buildAllocationInputs();
      await seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft(
        { id: selectedContainer.id },
        {
          id: selectedContainer.id,
          expectedVersion: String(selectedContainer.version || 1),
          allocations: payload,
        },
      );
      message.success('共享箱分配草稿已保存');
      await loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存草稿失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 确认分配
  const handleConfirm = async () => {
    if (!selectedContainer?.id) return;
    if (!summary.isBalanced) {
      message.warning('箱件重尺尚未平衡守恒，无法确认分配');
      return;
    }
    setSubmitting(true);
    try {
      const payload = buildAllocationInputs();
      const saveRes =
        await seaSharedContainerServiceSaveSeaSharedContainerAllocationsDraft(
          { id: selectedContainer.id },
          {
            id: selectedContainer.id,
            expectedVersion: String(selectedContainer.version || 1),
            allocations: payload,
          },
        );
      const nextVersion = String(
        saveRes?.data?.version ??
          Number(selectedContainer.version || 1) + 1,
      );
      await seaSharedContainerServiceConfirmSeaSharedContainer(
        { id: selectedContainer.id },
        {
          id: selectedContainer.id,
          expectedVersion: nextVersion,
        },
      );
      message.success('共享箱分配已确认生效');
      await loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '确认分配失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 撤回确认
  const handleWithdraw = async () => {
    if (!selectedContainer?.id) return;
    setSubmitting(true);
    try {
      await seaSharedContainerServiceWithdrawSeaSharedContainer(
        { id: selectedContainer.id },
        {
          id: selectedContainer.id,
          expectedVersion: String(selectedContainer.version || 1),
        },
      );
      message.success('已撤回至草稿状态');
      await loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '撤回失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 删除共享箱
  const handleDelete = async () => {
    if (!selectedContainer?.id) return;
    setSubmitting(true);
    try {
      await seaSharedContainerServiceDeleteSeaSharedContainer({
        id: selectedContainer.id,
        expectedVersion: String(selectedContainer.version || 1),
      });
      message.success('共享物理箱已删除');
      setSelectedContainerId(null);
      await loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除共享箱失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 新建共享箱提交
  const handleCreateSubmit = async (values: any) => {
    if (!transportExecutionId) return;
    setSubmitting(true);
    try {
      await seaSharedContainerServiceCreateSeaSharedContainer({
        input: {
          transportExecutionId,
          containerNo: values.containerNo.trim().toUpperCase(),
          containerSpecId: values.containerSpecId,
          sealNo: values.sealNo?.trim() || undefined,
          packageCount: Number(values.packageCount || 0),
          grossWeightKg: String(values.grossWeightKg || '0.000'),
          volumeCbm: String(values.volumeCbm || '0.000000'),
          note: values.note?.trim() || undefined,
        },
      });
      message.success('共享物理箱已创建');
      setCreateModalOpen(false);
      createForm.resetFields();
      await loadData();
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '创建共享箱失败');
    } finally {
      setSubmitting(false);
    }
  };

  const isConfirmed = selectedContainer?.status === 2; // SEA_SHARED_CONTAINER_STATUS_CONFIRMED

  const columns: ColumnsType<CargoAllocationItem> = [
    {
      title: '所属订单 / 分单',
      dataIndex: 'orderNo',
      width: 180,
      render: (_, record) => (
        <div>
          <Text strong>{record.orderNo}</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            HBL: {record.houseNo || '直单'}
          </Text>
        </div>
      ),
    },
    {
      title: '货物描述',
      dataIndex: 'cargoName',
      width: 160,
    },
    {
      title: '货物基准总量',
      width: 160,
      render: (_, record) => (
        <div style={{ fontSize: 12 }}>
          <div>件数: {record.totalPackageCount} PCS</div>
          <div>毛重: {record.totalGrossWeightKg} KG</div>
          <div>体积: {record.totalVolumeCbm} CBM</div>
        </div>
      ),
    },
    {
      title: '分配至本共享箱件数 (PCS)',
      width: 140,
      render: (_, record) => (
        <InputNumber
          min={0}
          max={record.totalPackageCount}
          value={record.packageCount}
          disabled={isConfirmed || !canManage}
          onChange={(val) =>
            handleUpdateAllocation(
              record.orderId,
              record.cargoItemId,
              'packageCount',
              val,
            )
          }
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: '分配毛重 (KG)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.grossWeightKg}
          disabled={isConfirmed || !canManage}
          onChange={(e) =>
            handleUpdateAllocation(
              record.orderId,
              record.cargoItemId,
              'grossWeightKg',
              e.target.value,
            )
          }
          placeholder="0.000"
        />
      ),
    },
    {
      title: '分配体积 (CBM)',
      width: 150,
      render: (_, record) => (
        <Input
          value={record.volumeCbm}
          disabled={isConfirmed || !canManage}
          onChange={(e) =>
            handleUpdateAllocation(
              record.orderId,
              record.cargoItemId,
              'volumeCbm',
              e.target.value,
            )
          }
          placeholder="0.000000"
        />
      ),
    },
    {
      title: '快捷操作',
      width: 110,
      render: (_, record) =>
        !isConfirmed && canManage ? (
          <Button
            size="small"
            type="link"
            onClick={() => handleFillAllCargo(record)}
          >
            全部填入
          </Button>
        ) : (
          '-'
        ),
    },
  ];

  return (
    <Drawer
      title="跨订单共享箱 / 客户拼货工作台"
      width={1100}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={
        <Space>
          {canManage && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalOpen(true)}
            >
              新建共享物理箱
            </Button>
          )}
        </Space>
      }
    >
      {!transportExecutionId ? (
        <Alert
          type="warning"
          showIcon
          message="当前订单未关联实际运输执行"
          description="共享集装箱以同一实际航次（TransportExecution）为协同边界。请先为订单指定运输执行与船名航次后，再开展客户拼货与跨订单分配。"
        />
      ) : (
        <Spin spinning={loading || submitting}>
          <div style={{ marginBottom: 16 }}>
            <Alert
              type="info"
              showIcon
              message="共享箱业务原则"
              description="普通单票订单常态下直接使用独占箱。若多张 HOUSE 订单同航次拼装同一物理箱，请在下方选择或新建共享箱，逐票分配件重尺。确认前系统将严格检查箱容量与各订单货物件重尺守恒。"
            />
          </div>

          {/* 共享箱卡片选择栏 */}
          <div style={{ marginBottom: 16 }}>
            <Title level={5}>本航次共享物理箱 ({containers.length})</Title>
            {containers.length === 0 ? (
              <Empty
                description="当前航次暂无共享物理箱"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              >
                {canManage && (
                  <Button
                    type="primary"
                    icon={<PlusOutlined />}
                    onClick={() => setCreateModalOpen(true)}
                  >
                    立即创建共享箱
                  </Button>
                )}
              </Empty>
            ) : (
              <Row gutter={[12, 12]}>
                {containers.map((cntr) => {
                  const isSelected = cntr.id === selectedContainerId;
                  const cntrConfirmed = cntr.status === 2;
                  return (
                    <Col xs={24} sm={12} md={8} key={cntr.id}>
                      <Card
                        size="small"
                        hoverable
                        onClick={() => setSelectedContainerId(cntr.id || null)}
                        style={{
                          borderColor: isSelected ? '#1677ff' : '#f0f0f0',
                          borderWidth: isSelected ? 2 : 1,
                          backgroundColor: isSelected ? '#f6ffed' : '#ffffff',
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                          }}
                        >
                          <Text
                            strong
                            style={{ fontFamily: 'monospace', fontSize: 15 }}
                          >
                            {cntr.containerNo}
                          </Text>
                          {cntrConfirmed ? (
                            <Tag color="success">已确认</Tag>
                          ) : (
                            <Tag color="warning">草稿</Tag>
                          )}
                        </div>
                        <div
                          style={{
                            marginTop: 4,
                            fontSize: 12,
                            color: 'rgba(0,0,0,0.65)',
                          }}
                        >
                          <div>
                            规格: {cntr.containerSpecName || cntr.containerSpecId}
                            {cntr.sealNo ? ` | 封号: ${cntr.sealNo}` : ''}
                          </div>
                          <div>
                            容量: {cntr.packageCount} PCS / {cntr.grossWeightKg}{' '}
                            KG / {cntr.volumeCbm} CBM
                          </div>
                        </div>
                      </Card>
                    </Col>
                  );
                })}
              </Row>
            )}
          </div>

          {/* 选定共享箱详情与守恒监控 */}
          {selectedContainer && (
            <>
              <Divider style={{ margin: '16px 0' }} />
              <Card
                title={
                  <Space>
                    <span>共享物理箱：</span>
                    <Text
                      style={{
                        fontFamily: 'monospace',
                        fontWeight: 600,
                        fontSize: 16,
                      }}
                    >
                      {selectedContainer.containerNo}
                    </Text>
                    <Tag color="blue">
                      {selectedContainer.containerSpecName ||
                        selectedContainer.containerSpecId}
                    </Tag>
                    {isConfirmed ? (
                      <Tag color="success" icon={<CheckCircleOutlined />}>
                        已确认生效
                      </Tag>
                    ) : (
                      <Tag color="warning" icon={<ExclamationCircleOutlined />}>
                        草稿待确认
                      </Tag>
                    )}
                    {summary.isBalanced ? (
                      <Tag color="cyan">件重尺完全守恒</Tag>
                    ) : (
                      <Tag color="red">
                        差额: {summary.diffPackages} PCS / {summary.diffWeight} KG
                        / {summary.diffVolume} CBM
                      </Tag>
                    )}
                  </Space>
                }
                extra={
                  <Space>
                    {!isConfirmed && canManage && (
                      <>
                        <Button
                          icon={<SaveOutlined />}
                          onClick={handleSaveDraft}
                          loading={submitting}
                        >
                          保存草稿
                        </Button>
                        <Button
                          type="primary"
                          icon={<CheckCircleOutlined />}
                          onClick={handleConfirm}
                          loading={submitting}
                          disabled={!summary.isBalanced}
                        >
                          确认分配
                        </Button>
                      </>
                    )}
                    {isConfirmed && canManage && (
                      <Button
                        icon={<RollbackOutlined />}
                        onClick={handleWithdraw}
                        loading={submitting}
                      >
                        撤回至草稿
                      </Button>
                    )}
                    {!isConfirmed && canManage && (
                      <Popconfirm
                        title="确定删除此共享物理箱？"
                        description="删除后相关草稿分配将一并清理。"
                        onConfirm={handleDelete}
                        okText="确定"
                        cancelText="取消"
                      >
                        <Button danger icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    )}
                  </Space>
                }
                style={{ marginBottom: 16 }}
              >
                <Descriptions size="small" column={{ xs: 1, sm: 3 }}>
                  <Descriptions.Item label="箱体总容量">
                    {summary.totalPackages} PCS / {summary.totalWeight} KG /{' '}
                    {summary.totalVolume} CBM
                  </Descriptions.Item>
                  <Descriptions.Item label="已分配总量">
                    {summary.allocPackages} PCS / {summary.allocWeight} KG /{' '}
                    {summary.allocVolume} CBM
                  </Descriptions.Item>
                  <Descriptions.Item label="剩余待分配">
                    <Text
                      type={
                        summary.diffPackages === 0 ? 'secondary' : 'danger'
                      }
                      strong
                    >
                      {summary.diffPackages} PCS / {summary.diffWeight} KG /{' '}
                      {summary.diffVolume} CBM
                    </Text>
                  </Descriptions.Item>
                </Descriptions>
              </Card>

              {/* 跨订单货物分配表格 */}
              <Card
                title={`同航次待分配 HOUSE 订单货物 (${candidates.length} 票)`}
                size="small"
              >
                {candidates.length === 0 ? (
                  <Empty
                    description="本航次暂无其他 HOUSE 订单可供拼箱"
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                  />
                ) : (
                  <Table
                    columns={columns}
                    dataSource={flatCargoList}
                    rowKey={(r) => `${r.orderId}:${r.cargoItemId}`}
                    pagination={false}
                    size="small"
                    bordered
                  />
                )}
              </Card>
            </>
          )}
        </Spin>
      )}

      {/* 新建共享箱 Modal */}
      <Modal
        title="新建跨订单共享物理箱"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form
          form={createForm}
          layout="vertical"
          onFinish={handleCreateSubmit}
          initialValues={{
            packageCount: 0,
            grossWeightKg: '0.000',
            volumeCbm: '0.000000',
          }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="containerNo"
                label="箱号 (Container No.)"
                rules={[
                  { required: true, message: '请输入箱号' },
                  {
                    pattern: /^[A-Z0-9]+$/i,
                    message: '箱号仅允许英文与数字',
                  },
                ]}
              >
                <Input placeholder="例如 COSU1234567" maxLength={30} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="containerSpecId"
                label="集装箱规格"
                rules={[{ required: true, message: '请选择箱型规格' }]}
              >
                <Select
                  placeholder="选择规格"
                  options={containerSpecOptions}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sealNo" label="铅封号 (Seal No.)">
                <Input placeholder="可选填铅封号" maxLength={50} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="packageCount"
                label="总件数 (PCS)"
                rules={[{ required: true, message: '请输入总件数' }]}
              >
                <InputNumber min={1} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="grossWeightKg"
                label="总毛重 (KG)"
                rules={[{ required: true, message: '请输入总毛重' }]}
              >
                <Input placeholder="例如 20000.000" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="volumeCbm"
                label="总体积 (CBM)"
                rules={[{ required: true, message: '请输入总体积' }]}
              >
                <Input placeholder="例如 65.000000" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="note" label="备注说明">
            <Input.TextArea rows={2} placeholder="拼箱注意事项或客户说明" />
          </Form.Item>
        </Form>
      </Modal>
    </Drawer>
  );
}
