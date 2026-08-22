import {
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  ModalForm,
  ProFormSelect,
  ProFormSwitch,
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
import React, { useEffect, useState } from 'react';
import {
  partnerServiceCreatePartnerAccount,
  partnerServiceListPartnerAccounts,
  partnerServiceUpdatePartnerAccount,
} from '@/services/roncin/partnerService';

const { Text } = Typography;

interface AccountCardListProps {
  partnerId?: string;
  currencyOptions: { label: string; value: string }[];
}

export default function AccountCardList({
  partnerId,
  currencyOptions,
}: AccountCardListProps) {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [accounts, setAccounts] = useState<API.PartnerAccount[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingAccount, setEditingAccount] = useState<API.PartnerAccount | undefined>(undefined);
  const [form] = Form.useForm();

  const fetchAccounts = async () => {
    if (!partnerId) return;
    setLoading(true);
    try {
      const res = await partnerServiceListPartnerAccounts({ partnerId });
      setAccounts(res.data || []);
    } catch {
      message.error('加载账户列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
  }, [partnerId]);

  const handleOpenAdd = () => {
    if (!partnerId) {
      message.info('请先保存客户基本信息后再添加账户');
      return;
    }
    setEditingAccount(undefined);
    form.resetFields();
    form.setFieldsValue({
      currency: 'CNY',
      status: 1, // ACTIVE
      isDefault: accounts.length === 0,
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (item: API.PartnerAccount) => {
    setEditingAccount(item);
    form.resetFields();
    form.setFieldsValue({
      currency: item.currency || 'CNY',
      invoiceTitle: item.invoiceTitle,
      unifiedSocialCreditCode: item.unifiedSocialCreditCode,
      billingAddress: item.billingAddress,
      billingPhone: item.billingPhone,
      bankName: item.bankName,
      bankAccount: item.bankAccount,
      swiftCode: item.swiftCode,
      isDefault: item.isDefault ?? false,
      status: item.status === 2 ? 2 : 1,
      remark: item.remark,
    });
    setModalOpen(true);
  };

  const handleDelete = async (item: API.PartnerAccount) => {
    if (!partnerId || !item.id) return;
    try {
      await partnerServiceUpdatePartnerAccount(
        { partnerId, id: item.id },
        {
          partnerId,
          id: item.id,
          account: {
            currency: item.currency || 'CNY',
            invoiceTitle: item.invoiceTitle || '',
            unifiedSocialCreditCode: item.unifiedSocialCreditCode,
            billingAddress: item.billingAddress,
            billingPhone: item.billingPhone,
            bankName: item.bankName,
            bankAccount: item.bankAccount,
            swiftCode: item.swiftCode,
            isDefault: false,
            status: 2, // INACTIVE
            remark: item.remark,
          },
        },
      );
      message.success('账户已停用并移出常用列表');
      fetchAccounts();
    } catch {
      message.error('操作失败');
    }
  };

  const handleSave = async (values: any) => {
    if (!partnerId) return false;
    try {
      const payload: API.PartnerAccountInput = {
        currency: values.currency,
        invoiceTitle: values.invoiceTitle?.trim(),
        unifiedSocialCreditCode: values.unifiedSocialCreditCode?.trim(),
        billingAddress: values.billingAddress?.trim(),
        billingPhone: values.billingPhone?.trim(),
        bankName: values.bankName?.trim(),
        bankAccount: values.bankAccount?.trim(),
        swiftCode: values.swiftCode?.trim(),
        isDefault: Boolean(values.isDefault),
        status: values.status ?? 1,
        remark: values.remark?.trim(),
      };

      if (editingAccount?.id) {
        await partnerServiceUpdatePartnerAccount(
          { partnerId, id: editingAccount.id },
          {
            partnerId,
            id: editingAccount.id,
            account: payload,
          },
        );
        message.success('账户信息已更新');
      } else {
        await partnerServiceCreatePartnerAccount(
          { partnerId },
          {
            partnerId,
            account: payload,
          },
        );
        message.success('账户已添加');
      }
      setModalOpen(false);
      fetchAccounts();
      return true;
    } catch (err: any) {
      message.error(err?.message || '保存失败');
      return false;
    }
  };

  const activeAccounts = accounts.filter((a) => a.status !== 2);

  return (
    <div>
      <Spin spinning={loading}>
        <Row gutter={[16, 16]}>
          {activeAccounts.map((item) => (
            <Col xs={24} sm={12} md={8} lg={6} key={item.id}>
              <Card
                size="small"
                style={{
                  height: '100%',
                  borderColor: item.isDefault ? '#1677ff' : '#e8e8e8',
                  borderRadius: 6,
                  position: 'relative',
                  backgroundColor: item.isDefault ? '#f8faff' : '#ffffff',
                }}
                styles={{
                  body: {
                    padding: '12px 14px',
                    fontSize: 12,
                    lineHeight: '1.8',
                  },
                }}
              >
                {/* Header: Title and Actions */}
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                    marginBottom: 6,
                  }}
                >
                  <div style={{ fontWeight: 600, fontSize: 13, color: '#262626', wordBreak: 'break-all', flex: 1, paddingRight: 8 }}>
                    {item.invoiceTitle || '未命名发票抬头'}
                    {item.isDefault && (
                      <Tag color="blue" style={{ marginLeft: 6, fontSize: 10, padding: '0 4px' }}>
                        默认导出
                      </Tag>
                    )}
                  </div>
                  <Space size={4}>
                    <Button
                      type="text"
                      size="small"
                      icon={<EditOutlined style={{ color: '#1677ff' }} />}
                      onClick={() => handleOpenEdit(item)}
                      style={{ padding: '0 4px', height: 22 }}
                    />
                    <Popconfirm
                      title="确定要停用/移除此账户吗？"
                      onConfirm={() => handleDelete(item)}
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

                {/* Account Details */}
                <div style={{ color: '#595959' }}>
                  {item.unifiedSocialCreditCode && (
                    <div>
                      <span style={{ color: '#8c8c8c' }}>社会统一信用代码: </span>
                      <span style={{ fontFamily: 'monospace' }}>{item.unifiedSocialCreditCode}</span>
                    </div>
                  )}
                  {item.billingAddress && (
                    <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      <span style={{ color: '#8c8c8c' }}>开票地址: </span>
                      {item.billingAddress}
                    </div>
                  )}
                  {item.billingPhone && (
                    <div>
                      <span style={{ color: '#8c8c8c' }}>开票电话: </span>
                      {item.billingPhone}
                    </div>
                  )}
                  {item.bankName && (
                    <div>
                      <span style={{ color: '#8c8c8c' }}>{item.currency || 'CNY'}开户行: </span>
                      {item.bankName}
                    </div>
                  )}
                  {item.bankAccount && (
                    <div>
                      <span style={{ color: '#8c8c8c' }}>{item.currency || 'CNY'}账号: </span>
                      <span style={{ fontFamily: 'monospace', fontWeight: 500 }}>{item.bankAccount}</span>
                    </div>
                  )}
                  {item.swiftCode && (
                    <div>
                      <span style={{ color: '#8c8c8c' }}>SWIFT Code: </span>
                      <span style={{ fontFamily: 'monospace' }}>{item.swiftCode}</span>
                    </div>
                  )}
                  <div>
                    <span style={{ color: '#8c8c8c' }}>是否默认导出: </span>
                    <span>{item.isDefault ? '是' : '否'}</span>
                  </div>
                </div>
              </Card>
            </Col>
          ))}

          {/* Add Account Card Button */}
          <Col xs={24} sm={12} md={8} lg={6}>
            <div
              onClick={handleOpenAdd}
              style={{
                height: '100%',
                minHeight: 150,
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
                添加账户
              </Text>
            </div>
          </Col>
        </Row>
      </Spin>

      {/* Account Add/Edit Modal */}
      <ModalForm
        title={editingAccount ? '编辑银行/开票账户' : '添加银行/开票账户'}
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
            name="invoiceTitle"
            label="发票抬头"
            placeholder="请输入发票开具抬头全称"
            rules={[{ required: true, message: '请输入发票抬头' }]}
          />
        </Col>
        <Col span={12}>
          <ProFormSelect
            name="currency"
            label="账户币种"
            options={currencyOptions}
            rules={[{ required: true, message: '请选择币种' }]}
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="unifiedSocialCreditCode"
            label="统一税号"
            placeholder="纳税人识别号/统一社会信用代码"
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="bankName"
            label="开户银行"
            placeholder="例如：中国工商银行高新支行"
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="bankAccount"
            label="银行账号"
            placeholder="请输入银行结算账号"
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="billingPhone"
            label="开票电话"
            placeholder="开票联系电话"
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="swiftCode"
            label="SWIFT Code"
            placeholder="境外汇款识别码"
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="billingAddress"
            label="开票地址"
            placeholder="请输入开票地址"
          />
        </Col>
        <Col span={12}>
          <ProFormSwitch
            name="isDefault"
            label="设为默认开票账户"
            extra="勾选后此账户作为默认对外开票导出账户"
          />
        </Col>
        <Col span={24}>
          <ProFormTextArea
            name="remark"
            label="账户备注"
            placeholder="特殊结算或开票要求备注"
            fieldProps={{ rows: 2 }}
          />
        </Col>
      </ModalForm>
    </div>
  );
}
