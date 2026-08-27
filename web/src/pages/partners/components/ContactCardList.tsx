import {
  DeleteOutlined,
  EditOutlined,
  MailOutlined,
  PhoneOutlined,
  PlusOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
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
  Tag,
  Typography,
} from 'antd';
import React, { useState } from 'react';

const { Text } = Typography;

export interface ContactItem {
  id?: string;
  name: string;
  phone?: string;
  email?: string;
  note?: string;
  isPrimary?: boolean;
}

interface ContactCardListProps {
  contacts: ContactItem[];
  onChange: (contacts: ContactItem[]) => void;
}

export default function ContactCardList({
  contacts,
  onChange,
}: ContactCardListProps) {
  const { message } = App.useApp();
  const [modalOpen, setModalOpen] = useState(false);
  const [editingIndex, setEditingIndex] = useState<number | undefined>(undefined);
  const [form] = Form.useForm();

  const handleOpenAdd = () => {
    setEditingIndex(undefined);
    form.resetFields();
    form.setFieldsValue({
      isPrimary: contacts.length === 0,
    });
    setModalOpen(true);
  };

  const handleOpenEdit = (index: number) => {
    setEditingIndex(index);
    const item = contacts[index];
    form.resetFields();
    form.setFieldsValue({
      name: item.name,
      phone: item.phone,
      email: item.email,
      note: item.note,
      isPrimary: item.isPrimary ?? false,
    });
    setModalOpen(true);
  };

  const handleDelete = (index: number) => {
    const next = contacts.filter((_, i) => i !== index);
    onChange(next);
    message.success('联系人已移除');
  };

  const handleSave = async (values: any) => {
    const newItem: ContactItem = {
      name: values.name?.trim(),
      phone: values.phone?.trim(),
      email: values.email?.trim(),
      note: values.note?.trim(),
      isPrimary: Boolean(values.isPrimary),
    };

    let next = [...contacts];
    if (newItem.isPrimary) {
      // 保证主联系人唯一
      next = next.map((c) => ({ ...c, isPrimary: false }));
    }

    if (editingIndex !== undefined && editingIndex >= 0) {
      next[editingIndex] = { ...next[editingIndex], ...newItem };
      message.success('联系人信息已更新');
    } else {
      next.push(newItem);
      message.success('联系人已添加');
    }

    onChange(next);
    setModalOpen(false);
    return true;
  };

  return (
    <div>
      <Row gutter={[16, 16]}>
        {contacts.map((item, index) => (
          <Col xs={24} sm={12} md={8} lg={6} key={item.id || index}>
            <Card
              size="small"
              style={{
                height: '100%',
                borderColor: item.isPrimary ? '#1677ff' : '#e8e8e8',
                borderRadius: 6,
                backgroundColor: item.isPrimary ? '#f8faff' : '#ffffff',
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
                  alignItems: 'center',
                  marginBottom: 6,
                }}
              >
                <Space size={6} align="center">
                  <UserOutlined style={{ color: '#1677ff' }} />
                  <Text strong style={{ fontSize: 13, color: '#262626' }}>
                    {item.name}
                  </Text>
                  {item.isPrimary && (
                    <Tag color="blue" style={{ fontSize: 10, padding: '0 4px', margin: 0 }}>
                      主联系人
                    </Tag>
                  )}
                </Space>
                <Space size={4}>
                  <Button
                    type="text"
                    size="small"
                    icon={<EditOutlined style={{ color: '#1677ff' }} />}
                    onClick={() => handleOpenEdit(index)}
                    style={{ padding: '0 4px', height: 22 }}
                  />
                  <Popconfirm
                    title="确定要移除此联系人吗？"
                    onConfirm={() => handleDelete(index)}
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

              {/* Details */}
              <div style={{ color: '#595959' }}>
                {item.note && (
                  <div>
                    <span style={{ color: '#8c8c8c' }}>职务/部门: </span>
                    <span>{item.note}</span>
                  </div>
                )}
                {item.phone && (
                  <div>
                    <PhoneOutlined style={{ color: '#8c8c8c', marginRight: 4 }} />
                    <span style={{ fontFamily: 'monospace' }}>{item.phone}</span>
                  </div>
                )}
                {item.email && (
                  <div style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    <MailOutlined style={{ color: '#8c8c8c', marginRight: 4 }} />
                    <span>{item.email}</span>
                  </div>
                )}
              </div>
            </Card>
          </Col>
        ))}

        {/* Add Contact Card Button */}
        <Col xs={24} sm={12} md={8} lg={6}>
          <div
            onClick={handleOpenAdd}
            style={{
              height: '100%',
              minHeight: 110,
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
                width: 40,
                height: 40,
                borderRadius: '50%',
                border: '2px solid #1677ff',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                color: '#1677ff',
                fontSize: 20,
              }}
            >
              <PlusOutlined />
            </div>
            <Text strong style={{ color: '#1677ff', fontSize: 13 }}>
              添加联系方式
            </Text>
          </div>
        </Col>
      </Row>

      {/* Modal Form */}
      <ModalForm
        title={editingIndex !== undefined ? '编辑联系人' : '添加联系人'}
        open={modalOpen}
        form={form}
        onOpenChange={setModalOpen}
        onFinish={handleSave}
        modalProps={{
          destroyOnClose: true,
          maskClosable: false,
          width: 480,
        }}
        layout="horizontal"
        labelAlign="right"
        labelCol={{ flex: '110px' }}
        wrapperCol={{ flex: 'auto' }}
        grid
      >
        <Col span={24}>
          <ProFormText
            name="name"
            label="姓名"
            placeholder="请输入联系人姓名"
            rules={[{ required: true, message: '请输入姓名' }]}
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="note"
            label="职务/备注"
            placeholder="例如：操作主管 / 财务负责"
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="phone"
            label="联系电话"
            placeholder="手机号码或座机分机号"
          />
        </Col>
        <Col span={24}>
          <ProFormText
            name="email"
            label="电子邮箱"
            placeholder="name@company.com"
          />
        </Col>
        <Col span={24}>
          <ProFormSwitch
            name="isPrimary"
            label="设为主联系人"
            extra="主联系人将作为默认对接与单证通知人"
          />
        </Col>
      </ModalForm>
    </div>
  );
}
