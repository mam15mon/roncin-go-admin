import {
  ContainerOutlined,
  DeleteOutlined,
  EditOutlined,
  FileTextOutlined,
  NumberOutlined,
  PlusOutlined,
  SendOutlined,
  TagsOutlined,
  UsergroupAddOutlined,
} from '@ant-design/icons';
import {
  ModalForm,
  ProFormDigit,
  ProFormSwitch,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  App,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Popconfirm,
  Row,
  Space,
  Spin,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import React, { useEffect, useMemo, useState } from 'react';
import {
  partnerServiceCreatePartnerShippingPreset,
  partnerServiceListPartnerShippingPresets,
  partnerServiceUpdatePartnerShippingPreset,
} from '@/services/roncin/partnerService';

const { Text, Paragraph } = Typography;

export const PRESET_TYPES = [
  { key: 1, label: '发货人 (Shipper)', icon: <SendOutlined />, short: '发货人', isParty: true },
  { key: 2, label: '收货人 (Consignee)', icon: <ContainerOutlined />, short: '收货人', isParty: true },
  { key: 3, label: '通知人 (Notify)', icon: <UsergroupAddOutlined />, short: '通知人', isParty: true },
  { key: 4, label: '英文品名 (Cargo Name)', icon: <FileTextOutlined />, short: '英文品名', isParty: false },
  { key: 5, label: 'HS编码 (HS Code)', icon: <NumberOutlined />, short: 'HS', isParty: false },
  { key: 6, label: '唛头 (Shipping Marks)', icon: <TagsOutlined />, short: '唛头', isParty: false },
];

const PRESET_TYPE_MAP = new Map(PRESET_TYPES.map((t) => [t.key, t]));

interface ShippingPresetSectionProps {
  partnerId?: string;
}

export default function ShippingPresetSection({
  partnerId,
}: ShippingPresetSectionProps) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [presets, setPresets] = useState<API.PartnerShippingPreset[]>([]);
  const [activeTab, setActiveTab] = useState<string>('all');
  const [modalOpen, setModalOpen] = useState(false);
  const [currentType, setCurrentType] = useState<number>(1);
  const [editingPreset, setEditingPreset] = useState<API.PartnerShippingPreset | undefined>(undefined);
  const [form] = Form.useForm();

  const fetchPresets = async () => {
    if (!partnerId) return;
    setLoading(true);
    try {
      const res = await partnerServiceListPartnerShippingPresets({ partnerId });
      setPresets(res.data || []);
    } catch {
      message.error('加载单证常用信息失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPresets();
  }, [partnerId]);

  const handleOpenAdd = (type: number) => {
    if (!partnerId) {
      message.info('请先保存客户基本信息后再添加常用单证预设');
      return;
    }
    setCurrentType(type);
    setEditingPreset(undefined);
    form.resetFields();
    form.setFieldsValue({
      presetType: type,
      isDefault: false,
      enabled: true,
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (preset: API.PartnerShippingPreset) => {
    const type = preset.presetType || 1;
    setCurrentType(type);
    setEditingPreset(preset);
    form.resetFields();
    form.setFieldsValue({
      presetType: type,
      title: preset.title,
      isDefault: preset.isDefault ?? false,
      sortOrder: preset.sortOrder ?? 0,
      remark: preset.remark,
      enabled: preset.enabled ?? true,

      // Party fields
      companyName: preset.party?.companyName,
      address: preset.party?.address,
      contactName: preset.party?.contactName,
      phone: preset.party?.phone,
      email: preset.party?.email,
      countryCode: preset.party?.countryCode || 'CN',
      taxIdentifier: preset.party?.taxIdentifier,

      // Text fields
      content: preset.text?.content,
      code: preset.text?.code,
    });
    setModalOpen(true);
  };

  const handleDelete = async (preset: API.PartnerShippingPreset) => {
    if (!partnerId || !preset.id) return;
    try {
      await partnerServiceUpdatePartnerShippingPreset(
        { partnerId, id: preset.id },
        {
          partnerId,
          id: preset.id,
          preset: {
            presetType: preset.presetType || 1,
            title: preset.title || '',
            party: preset.party,
            text: preset.text,
            isDefault: false,
            sortOrder: preset.sortOrder,
            remark: preset.remark,
            enabled: false, // 软删除停用
          },
        },
      );
      message.success('预设信息已停用并移出列表');
      fetchPresets();
    } catch {
      message.error('操作失败');
    }
  };

  const handleSave = async (values: any) => {
    if (!partnerId) return false;
    try {
      const isParty = currentType <= 3;
      const payload: API.PartnerShippingPresetInput = {
        presetType: currentType,
        title: values.title?.trim(),
        isDefault: Boolean(values.isDefault),
        sortOrder: Number(values.sortOrder || 0),
        remark: values.remark?.trim(),
        enabled: values.enabled ?? true,
      };

      if (isParty) {
        payload.party = {
          companyName: values.companyName?.trim(),
          address: values.address?.trim(),
          contactName: values.contactName?.trim(),
          phone: values.phone?.trim(),
          email: values.email?.trim(),
          countryCode: values.countryCode?.trim() || 'CN',
          taxIdentifier: values.taxIdentifier?.trim(),
        };
      } else {
        payload.text = {
          content: values.content?.trim(),
          code: values.code?.trim(),
        };
      }

      if (editingPreset?.id) {
        await partnerServiceUpdatePartnerShippingPreset(
          { partnerId, id: editingPreset.id },
          {
            partnerId,
            id: editingPreset.id,
            preset: payload,
          },
        );
        message.success('常用单证预设已更新');
      } else {
        await partnerServiceCreatePartnerShippingPreset(
          { partnerId },
          {
            partnerId,
            preset: payload,
          },
        );
        message.success('常用单证预设已添加');
      }
      setModalOpen(false);
      fetchPresets();
      return true;
    } catch (err: any) {
      message.error(err?.message || '保存失败');
      return false;
    }
  };

  const activePresets = useMemo(
    () => presets.filter((p) => p.enabled !== false),
    [presets],
  );

  const filteredPresets = useMemo(() => {
    if (activeTab === 'all') return activePresets;
    const typeNum = Number(activeTab);
    return activePresets.filter((p) => p.presetType === typeNum);
  }, [activePresets, activeTab]);

  const isPartyForm = currentType <= 3;
  const currentPresetMeta = PRESET_TYPE_MAP.get(currentType);

  return (
    <div>
      {/* 6 Quick Action Buttons (Exact Match to Competitor Design) */}
      <div style={{ marginBottom: 16 }}>
        <Row gutter={[12, 12]}>
          {PRESET_TYPES.map((type) => (
            <Col xs={12} sm={8} md={4} key={type.key}>
              <Button
                block
                style={{
                  height: 40,
                  borderRadius: 6,
                  borderColor: '#91caff',
                  backgroundColor: '#e6f4ff',
                  color: '#1677ff',
                  fontWeight: 500,
                  fontSize: 13,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  gap: 6,
                  transition: 'all 0.2s',
                }}
                icon={<PlusOutlined style={{ fontSize: 12 }} />}
                onClick={() => handleOpenAdd(type.key)}
              >
                添加{type.short}
              </Button>
            </Col>
          ))}
        </Row>
      </div>

      {/* Preset List / Card View with Tabs */}
      <Spin spinning={loading}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          size="small"
          items={[
            { key: 'all', label: `全部预设 (${activePresets.length})` },
            ...PRESET_TYPES.map((t) => ({
              key: String(t.key),
              label: `${t.short} (${activePresets.filter((p) => p.presetType === t.key).length})`,
            })),
          ]}
        />

        {filteredPresets.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="暂无常用单证预设，点击上方按钮快捷录入"
            style={{ padding: '20px 0' }}
          />
        ) : (
          <Row gutter={[16, 16]}>
            {filteredPresets.map((preset) => {
              const meta = PRESET_TYPE_MAP.get(preset.presetType || 1);
              const isParty = (preset.presetType || 1) <= 3;
              return (
                <Col xs={24} sm={12} md={8} lg={6} key={preset.id}>
                  <Card
                    size="small"
                    style={{
                      height: '100%',
                      borderColor: preset.isDefault ? '#1677ff' : '#e8e8e8',
                      borderRadius: 6,
                      backgroundColor: preset.isDefault ? '#f8faff' : '#ffffff',
                    }}
                    styles={{
                      body: {
                        padding: '12px 14px',
                        fontSize: 12,
                        lineHeight: '1.7',
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
                      <Space size={6} align="start" style={{ flex: 1, paddingRight: 8 }}>
                        <Tag color="blue" style={{ fontSize: 11, padding: '0 4px', margin: 0 }}>
                          {meta?.short}
                        </Tag>
                        <Text strong style={{ fontSize: 13, color: '#262626', wordBreak: 'break-all' }}>
                          {preset.title}
                        </Text>
                      </Space>
                      <Space size={4}>
                        <Button
                          type="text"
                          size="small"
                          icon={<EditOutlined style={{ color: '#1677ff' }} />}
                          onClick={() => handleOpenEdit(preset)}
                          style={{ padding: '0 4px', height: 22 }}
                        />
                        <Popconfirm
                          title="确定要停用此预设吗？"
                          onConfirm={() => handleDelete(preset)}
                          okText="确定"
                          cancelText="取消"
                        >
                          <Button
                            type="text"
                            size="small"
                            icon={<DeleteOutlined style={{ color: '#ff4d4f' }} />}
                            style={{ padding: '0 4px', height: 22 }}
                          />
                        </Popconfirm>
                      </Space>
                    </div>

                    {/* Content */}
                    <div style={{ color: '#595959' }}>
                      {isParty ? (
                        <>
                          {preset.party?.companyName && (
                            <div style={{ fontWeight: 500, color: '#262626' }}>
                              {preset.party.companyName}
                            </div>
                          )}
                          {preset.party?.address && (
                            <Paragraph
                              ellipsis={{ rows: 2 }}
                              style={{ margin: 0, color: '#595959', fontSize: 12 }}
                            >
                              <span style={{ color: '#8c8c8c' }}>地址: </span>
                              {preset.party.address}
                            </Paragraph>
                          )}
                          {preset.party?.contactName && (
                            <div>
                              <span style={{ color: '#8c8c8c' }}>联系人: </span>
                              {preset.party.contactName} {preset.party.phone && `(${preset.party.phone})`}
                            </div>
                          )}
                          {preset.party?.taxIdentifier && (
                            <div>
                              <span style={{ color: '#8c8c8c' }}>税号: </span>
                              <span style={{ fontFamily: 'monospace' }}>{preset.party.taxIdentifier}</span>
                            </div>
                          )}
                        </>
                      ) : (
                        <>
                          {preset.text?.code && (
                            <div>
                              <span style={{ color: '#8c8c8c' }}>编码/HS: </span>
                              <span style={{ fontFamily: 'monospace', fontWeight: 600, color: '#1677ff' }}>
                                {preset.text.code}
                              </span>
                            </div>
                          )}
                          {preset.text?.content && (
                            <Paragraph
                              ellipsis={{ rows: 3 }}
                              style={{ margin: 0, color: '#262626', whiteSpace: 'pre-wrap', fontSize: 12 }}
                            >
                              {preset.text.content}
                            </Paragraph>
                          )}
                        </>
                      )}
                      {preset.isDefault && (
                        <div style={{ marginTop: 4 }}>
                          <Tag color="green" style={{ fontSize: 10, padding: '0 4px' }}>
                            默认带出
                          </Tag>
                        </div>
                      )}
                    </div>
                  </Card>
                </Col>
              );
            })}
          </Row>
        )}
      </Spin>

      {/* Preset Modal */}
      <ModalForm
        title={
          editingPreset
            ? `编辑常用${currentPresetMeta?.short || ''}`
            : `添加常用${currentPresetMeta?.short || ''}`
        }
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleSave}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
          width: 580,
        }}
        layout="horizontal"
        grid
      >
        <Col span={24}>
          <ProFormText
            name="title"
            label="预设简称/标识"
            placeholder="例如：工厂出货抬头 / 常用HS-电子配件"
            rules={[{ required: true, message: '请输入预设简称' }]}
          />
        </Col>

        {isPartyForm ? (
          <>
            <Col span={24}>
              <ProFormText
                name="companyName"
                label="公司抬头 (EN/CN)"
                placeholder="提单/托运单制单英文公司全称"
                rules={[{ required: true, message: '请输入公司抬头' }]}
              />
            </Col>
            <Col span={24}>
              <ProFormTextArea
                name="address"
                label="单证地址"
                placeholder="请输入详细外文/单证地址（多行格式）"
                fieldProps={{ rows: 3 }}
                rules={[{ required: true, message: '请输入地址' }]}
              />
            </Col>
            <Col span={12}>
              <ProFormText
                name="contactName"
                label="联系人姓名"
                placeholder="联系人姓名"
              />
            </Col>
            <Col span={12}>
              <ProFormText
                name="phone"
                label="联系电话"
                placeholder="联系电话/传真"
              />
            </Col>
            <Col span={12}>
              <ProFormText
                name="email"
                label="电子邮箱"
                placeholder="通知/确认邮箱"
              />
            </Col>
            <Col span={12}>
              <ProFormText
                name="taxIdentifier"
                label="税号/注册号"
                placeholder="EIN / VAT / 税号"
              />
            </Col>
          </>
        ) : (
          <>
            {currentType === 5 && (
              <Col span={24}>
                <ProFormText
                  name="code"
                  label="HS海关编码"
                  placeholder="例如：8471.30.0000"
                  rules={[{ required: true, message: '请输入海关编码' }]}
                />
              </Col>
            )}
            <Col span={24}>
              <ProFormTextArea
                name="content"
                label={currentType === 6 ? '唛头与件号' : '英文详细品名'}
                placeholder={
                  currentType === 6
                    ? 'N/M 或\nABC CO., LTD\nC/NO. 1-100\nMADE IN CHINA'
                    : 'AUTOMATIC DATA PROCESSING MACHINES AND UNITS THEREOF'
                }
                fieldProps={{ rows: 4 }}
                rules={[{ required: true, message: '请输入内容' }]}
              />
            </Col>
          </>
        )}

        <Col span={12}>
          <ProFormSwitch
            name="isDefault"
            label="设为默认带出"
            extra="在该业务类型制单时自动作为优先带出项"
          />
        </Col>
        <Col span={12}>
          <ProFormDigit
            name="sortOrder"
            label="排序权重"
            min={0}
            max={999}
            initialValue={0}
          />
        </Col>
        <Col span={24}>
          <ProFormTextArea
            name="remark"
            label="操作备注"
            placeholder="特殊操作要求或单证注意事项"
            fieldProps={{ rows: 2 }}
          />
        </Col>
      </ModalForm>
    </div>
  );
}
