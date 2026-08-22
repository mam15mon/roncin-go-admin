import {
  CalendarOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  ModalForm,
  ProFormDateRangePicker,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  App,
  Button,
  Card,
  Col,
  Form,
  Popconfirm,
  Row,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import React, { useEffect, useState } from 'react';
import {
  partnerServiceCreatePartnerContract,
  partnerServiceListPartnerContracts,
  partnerServiceUpdatePartnerContract,
} from '@/services/roncin/partnerService';

const { Text, Paragraph } = Typography;

const CONTRACT_STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: '待生效', color: 'processing' },
  2: { label: '生效中', color: 'success' },
  3: { label: '已到期', color: 'default' },
  4: { label: '已终止', color: 'error' },
};

interface ContractCardListProps {
  partnerId?: string;
}

export default function ContractCardList({ partnerId }: ContractCardListProps) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [contracts, setContracts] = useState<API.PartnerContract[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingContract, setEditingContract] = useState<API.PartnerContract | undefined>(undefined);
  const [form] = Form.useForm();

  const fetchContracts = async () => {
    if (!partnerId) return;
    setLoading(true);
    try {
      const res = await partnerServiceListPartnerContracts({ partnerId });
      setContracts(res.data || []);
    } catch {
      message.error('加载合同列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchContracts();
  }, [partnerId]);

  const handleOpenAdd = () => {
    if (!partnerId) {
      message.info('请先保存客户基本信息后再添加合同');
      return;
    }
    setEditingContract(undefined);
    form.resetFields();
    form.setFieldsValue({
      status: 2, // 生效中
      dateRange: [dayjs(), dayjs().add(1, 'year')],
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (item: API.PartnerContract) => {
    setEditingContract(item);
    form.resetFields();
    form.setFieldsValue({
      contractNo: item.contractNo,
      name: item.name,
      status: item.status ?? 2,
      dateRange:
        item.startDate && item.endDate
          ? [dayjs(item.startDate), dayjs(item.endDate)]
          : undefined,
      paymentTerms: item.paymentTerms,
      disputeResolution: item.disputeResolution,
      otherNotes: item.otherNotes,
    });
    setModalOpen(true);
  };

  const handleTerminate = async (item: API.PartnerContract) => {
    if (!partnerId || !item.id) return;
    try {
      await partnerServiceUpdatePartnerContract(
        { partnerId, id: item.id },
        {
          partnerId,
          id: item.id,
          contract: {
            name: item.name || '',
            status: 4, // 已终止
            startDate: item.startDate || '',
            endDate: item.endDate || '',
            paymentTerms: item.paymentTerms,
            disputeResolution: item.disputeResolution,
            otherNotes: item.otherNotes,
          },
        },
      );
      message.success('合同已终止');
      fetchContracts();
    } catch {
      message.error('操作失败');
    }
  };

  const handleSave = async (values: any) => {
    if (!partnerId) return false;
    try {
      const dateRange: [Dayjs, Dayjs] = values.dateRange || [dayjs(), dayjs()];
      const startDate = dateRange[0].format('YYYY-MM-DD');
      const endDate = dateRange[1].format('YYYY-MM-DD');

      if (editingContract?.id) {
        await partnerServiceUpdatePartnerContract(
          { partnerId, id: editingContract.id },
          {
            partnerId,
            id: editingContract.id,
            contract: {
              name: values.name?.trim(),
              status: values.status,
              startDate,
              endDate,
              paymentTerms: values.paymentTerms?.trim(),
              disputeResolution: values.disputeResolution?.trim(),
              otherNotes: values.otherNotes?.trim(),
            },
          },
        );
        message.success('合同信息已更新');
      } else {
        await partnerServiceCreatePartnerContract(
          { partnerId },
          {
            partnerId,
            contract: {
              contractNo: values.contractNo?.trim(),
              name: values.name?.trim(),
              status: values.status,
              startDate,
              endDate,
              paymentTerms: values.paymentTerms?.trim(),
              disputeResolution: values.disputeResolution?.trim(),
              otherNotes: values.otherNotes?.trim(),
            },
          },
        );
        message.success('合同已添加');
      }
      setModalOpen(false);
      fetchContracts();
      return true;
    } catch (err: any) {
      message.error(err?.message || '保存失败');
      return false;
    }
  };

  return (
    <div>
      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          {contracts.map((item) => {
            const statusMeta = CONTRACT_STATUS_MAP[item.status ?? 1] || {
              label: '未知',
              color: 'default',
            };
            return (
              <Col xs={24} sm={12} md={8} lg={6} key={item.id}>
                <Card
                  size="small"
                  style={{
                    height: '100%',
                    borderColor: item.status === 2 ? '#91caff' : '#e8e8e8',
                    borderRadius: 6,
                    backgroundColor: '#ffffff',
                  }}
                  styles={{
                    body: {
                      padding: '12px 14px',
                      fontSize: 12,
                      lineHeight: '1.8',
                    },
                  }}
                >
                  {/* Header */}
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'flex-start',
                      marginBottom: 6,
                    }}
                  >
                    <div style={{ flex: 1, paddingRight: 8 }}>
                      <div style={{ fontWeight: 600, fontSize: 13, color: '#262626', wordBreak: 'break-all' }}>
                        {item.name}
                      </div>
                      <Space size={6} style={{ marginTop: 2 }}>
                        <Tag style={{ fontFamily: 'monospace', fontSize: 11, padding: '0 4px', margin: 0 }}>
                          {item.contractNo}
                        </Tag>
                        <Tag color={statusMeta.color} style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
                          {statusMeta.label}
                        </Tag>
                      </Space>
                    </div>
                    <Space size={4}>
                      <Button
                        type="text"
                        size="small"
                        icon={<EditOutlined style={{ color: '#1677ff' }} />}
                        onClick={() => handleOpenEdit(item)}
                        style={{ padding: '0 4px', height: 22 }}
                      />
                      {item.status !== 4 && (
                        <Popconfirm
                          title="确定要终止此合同吗？"
                          onConfirm={() => handleTerminate(item)}
                          okText="终止"
                          cancelText="取消"
                        >
                          <Button
                            type="text"
                            size="small"
                            icon={<DeleteOutlined style={{ color: '#ff4d4f' }} />}
                            style={{ padding: '0 4px', height: 22 }}
                          />
                        </Popconfirm>
                      )}
                    </Space>
                  </div>

                  {/* Details */}
                  <div style={{ color: '#595959' }}>
                    <div>
                      <CalendarOutlined style={{ color: '#8c8c8c', marginRight: 4 }} />
                      <span style={{ color: '#8c8c8c' }}>有效期: </span>
                      <span>
                        {item.startDate ? dayjs(item.startDate).format('YYYY-MM-DD') : '-'} 至{' '}
                        {item.endDate ? dayjs(item.endDate).format('YYYY-MM-DD') : '-'}
                      </span>
                    </div>
                    {item.paymentTerms && (
                      <Paragraph
                        ellipsis={{ rows: 2 }}
                        style={{ margin: 0, color: '#595959', fontSize: 12 }}
                      >
                        <span style={{ color: '#8c8c8c' }}>结算条款: </span>
                        {item.paymentTerms}
                      </Paragraph>
                    )}
                  </div>
                </Card>
              </Col>
            );
          })}

          {/* Add Contract Card Button */}
          <Col xs={24} sm={12} md={8} lg={6}>
            <div
              onClick={handleOpenAdd}
              style={{
                height: '100%',
                minHeight: 140,
                border: '1px dashed #91caff',
                borderRadius: 6,
                backgroundColor: '#e6f4ff',
                cursor: 'pointer',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 8,
                transition: 'all 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.backgroundColor = '#bae0ff';
                e.currentTarget.style.borderColor = '#1677ff';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.backgroundColor = '#e6f4ff';
                e.currentTarget.style.borderColor = '#91caff';
              }}
            >
              <div
                style={{
                  width: 44,
                  height: 44,
                  borderRadius: '50%',
                  border: '2px solid #1677ff',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#1677ff',
                  fontSize: 22,
                }}
              >
                <PlusOutlined />
              </div>
              <Text strong style={{ color: '#1677ff', fontSize: 14 }}>
                添加合同
              </Text>
            </div>
          </Col>
        </Row>
      </Spin>

      {/* Contract Modal */}
      <ModalForm
        title={editingContract ? '编辑合同信息' : '添加客户合同'}
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleSave}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
          width: 560,
        }}
        layout="horizontal"
        grid
      >
        <Col span={12}>
          <ProFormText
            name="contractNo"
            label="合同编号"
            placeholder="例如：CT-2026-0089"
            disabled={Boolean(editingContract)}
            rules={[{ required: true, message: '请输入合同编号' }]}
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="name"
            label="合同名称"
            placeholder="例如：2026年度海运代理框架合同"
            rules={[{ required: true, message: '请输入合同名称' }]}
          />
        </Col>
        <Col span={12}>
          <ProFormSelect
            name="status"
            label="合同状态"
            options={[
              { label: '待生效', value: 1 },
              { label: '生效中', value: 2 },
              { label: '已到期', value: 3 },
              { label: '已终止', value: 4 },
            ]}
            rules={[{ required: true, message: '请选择状态' }]}
          />
        </Col>
        <Col span={12}>
          <ProFormDateRangePicker
            name="dateRange"
            label="合同有效期"
            rules={[{ required: true, message: '请选择起止日期' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormTextArea
            name="paymentTerms"
            label="付款/结算条款"
            placeholder="例如：提单签发后30天内电汇付清"
            fieldProps={{ rows: 2 }}
          />
        </Col>
        <Col span={24}>
          <ProFormTextArea
            name="disputeResolution"
            label="争议解决条款"
            placeholder="例如：交由上海国际经济贸易仲裁委员会仲裁"
            fieldProps={{ rows: 2 }}
          />
        </Col>
        <Col span={24}>
          <ProFormTextArea
            name="otherNotes"
            label="其他备注说明"
            placeholder="合同附件及附加说明"
            fieldProps={{ rows: 2 }}
          />
        </Col>
      </ModalForm>
    </div>
  );
}
