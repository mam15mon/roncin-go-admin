import {
  DeleteOutlined,
  EditOutlined,
  MailOutlined,
  PhoneOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { App, Button, Card, Col, Popconfirm, Space, Tag, Typography } from 'antd';
import React from 'react';
import { SubEntityCardGrid } from '@/components/ui/sub-entity-card-grid';

const { Text } = Typography;

export type ContactItem = API.PartnerContact & { name: string };

interface ContactCardListProps {
  contacts: ContactItem[];
  onChange: (contacts: ContactItem[]) => void;
}

export default function ContactCardList({
  contacts,
  onChange,
}: ContactCardListProps) {
  const { message } = App.useApp();

  const handleSave = (
    values: any,
    _editingItem?: ContactItem,
    editingIndex?: number,
  ) => {
    const newItem: ContactItem = {
      name: values.name?.trim(),
      phone: values.phone?.trim(),
      email: values.email?.trim(),
      note: values.note?.trim(),
      isPrimary: Boolean(values.isPrimary),
    };

    let next = [...contacts];
    if (newItem.isPrimary) {
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
    return true;
  };

  const handleDelete = (_item: ContactItem, index: number) => {
    const next = contacts.filter((_, i) => i !== index);
    onChange(next);
    message.success('联系人已移除');
  };

  return (
    <SubEntityCardGrid<ContactItem>
      entityName="联系人"
      items={contacts}
      modalWidth={480}
      colSpan={{ xs: 24, sm: 12, md: 8, lg: 6 }}
      onSave={handleSave}
      onDelete={handleDelete}
      initialValues={(editing) =>
        editing
          ? {
              name: editing.name,
              phone: editing.phone,
              email: editing.email,
              note: editing.note,
              isPrimary: editing.isPrimary ?? false,
            }
          : {
              isPrimary: contacts.length === 0,
            }
      }
      renderCard={(item, _index, { openEdit, deleteItem }) => (
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
                <Tag
                  color="blue"
                  style={{ fontSize: 10, padding: '0 4px', margin: 0 }}
                >
                  主联系人
                </Tag>
              )}
            </Space>
            <Space size={4}>
              <Button
                type="text"
                size="small"
                icon={<EditOutlined style={{ color: '#1677ff' }} />}
                onClick={openEdit}
                style={{ padding: '0 4px', height: 22 }}
              />
              <Popconfirm
                title="确定要移除此联系人吗？"
                onConfirm={deleteItem}
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
                <PhoneOutlined
                  style={{ color: '#8c8c8c', marginRight: 4 }}
                />
                <span style={{ fontFamily: 'monospace' }}>{item.phone}</span>
              </div>
            )}
            {item.email && (
              <div
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                <MailOutlined
                  style={{ color: '#8c8c8c', marginRight: 4 }}
                />
                <span>{item.email}</span>
              </div>
            )}
          </div>
        </Card>
      )}
      renderFormItems={() => (
        <>
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
        </>
      )}
    />
  );
}
