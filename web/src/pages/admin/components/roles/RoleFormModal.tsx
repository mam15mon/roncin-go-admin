import {
  AppstoreOutlined,
  CheckSquareOutlined,
  DownOutlined,
  MinusSquareOutlined,
  SearchOutlined,
  UpOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  App,
  Button,
  Col,
  Empty,
  Input,
  Row,
  Select,
  Space,
  Tag,
  Tooltip,
  Tree,
  Typography,
} from 'antd';
import type { DataNode } from 'antd/es/tree';
import React from 'react';
import {
  adminServiceCreateRole,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';
import {
  applyPermissionLinkage,
  mergeVisiblePermissionSelection,
} from './permissionLinkage';
import {
  type OrderOrganizationAccess,
  type PermissionGroupNode,
  type PermissionLeafNode,
  type RoleFormValues,
  dataScopeOptions,
} from './roleConstants';

const { Text } = Typography;

interface RoleFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing?: API.AdminRole;
  formRef: React.RefObject<ProFormInstance | undefined>;
  companyOptions: { label: string; value: string }[];
  allLeafKeys: string[];
  allGroupKeys: string[];
  filteredTreeData: PermissionGroupNode[];
  requiresByPermission: Record<string, string[]>;
  permissionNameByKey: Record<string, string>;
  selectedPermissionKeys: string[];
  setSelectedPermissionKeys: (keys: string[]) => void;
  orderOrganizationAccesses: OrderOrganizationAccess[];
  setOrderOrganizationAccesses: React.Dispatch<
    React.SetStateAction<OrderOrganizationAccess[]>
  >;
  expandedKeys: React.Key[];
  setExpandedKeys: (keys: React.Key[]) => void;
  autoExpandParent: boolean;
  setAutoExpandParent: (val: boolean) => void;
  permissionKeyword: string;
  setPermissionKeyword: (kw: string) => void;
  onSuccess: () => void;
}

export default function RoleFormModal({
  open,
  onOpenChange,
  editing,
  formRef,
  companyOptions,
  allLeafKeys,
  allGroupKeys,
  filteredTreeData,
  requiresByPermission,
  permissionNameByKey,
  selectedPermissionKeys,
  setSelectedPermissionKeys,
  orderOrganizationAccesses,
  setOrderOrganizationAccesses,
  expandedKeys,
  setExpandedKeys,
  autoExpandParent,
  setAutoExpandParent,
  permissionKeyword,
  setPermissionKeyword,
  onSuccess,
}: RoleFormModalProps) {
  const { message } = App.useApp();

  const handleSelectAll = () => {
    setSelectedPermissionKeys([...allLeafKeys]);
  };

  const handleClearAll = () => {
    setSelectedPermissionKeys([]);
  };

  const handleExpandAll = () => {
    setExpandedKeys(allGroupKeys);
    setAutoExpandParent(false);
  };

  const handleCollapseAll = () => {
    setExpandedKeys([]);
    setAutoExpandParent(false);
  };

  const handleTreeCheck = (
    checked: React.Key[] | { checked: React.Key[]; halfChecked: React.Key[] },
  ) => {
    const checkedKeys = Array.isArray(checked) ? checked : checked.checked;
    const leafKeys = (checkedKeys as string[]).filter(
      (key) => !key.startsWith('group:'),
    );
    const visibleKeys = filteredTreeData.flatMap((group) =>
      group.children.map((leaf) => leaf.key),
    );
    const mergedKeys = mergeVisiblePermissionSelection(
      selectedPermissionKeys,
      visibleKeys,
      leafKeys,
    );
    // 勾选联动：勾选操作权限连带补齐其依赖；取消基础权限级联移除依赖它的权限。
    setSelectedPermissionKeys(
      applyPermissionLinkage(
        selectedPermissionKeys,
        mergedKeys,
        requiresByPermission,
      ),
    );
  };

  return (
    <ModalForm<RoleFormValues>
      title={
        editing ? `编辑角色：${editing.name} (${editing.code})` : '新增角色'
      }
      open={open}
      formRef={formRef}
      initialValues={
        editing
          ? { ...editing, permissionKeys: editing.permissionKeys }
          : { dataScope: 2, enabled: true }
      }
      modalProps={{
        destroyOnClose: true,
        width: 800,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        try {
          if (editing?.id) {
            await adminServiceUpdateRole(
              { id: editing.id },
              {
                id: editing.id,
                name: values.name?.trim() ?? '',
                dataScope: values.dataScope ?? 2,
                enabled: values.enabled ?? true,
                permissionKeys: selectedPermissionKeys,
                orderOrganizationAccesses,
              },
            );
            message.success('角色已成功更新');
          } else {
            await adminServiceCreateRole({
              code: values.code?.trim() ?? '',
              name: values.name?.trim() ?? '',
              dataScope: values.dataScope ?? 2,
              permissionKeys: selectedPermissionKeys,
              orderOrganizationAccesses,
            });
            message.success('角色已成功创建');
          }
          onOpenChange(false);
          onSuccess();
          return true;
        } catch {
          message.error('保存角色失败，请重试');
          return false;
        }
      }}
    >
      <Row gutter={16}>
        <Col span={12}>
          <ProFormText
            name="code"
            label="角色编码"
            placeholder="例如：ROLE_OPS_LEAD"
            disabled={Boolean(editing)}
            rules={[
              { required: true, message: '请输入角色编码' },
              {
                pattern: /^[A-Za-z0-9_-]+$/,
                message: '编码仅支持英文字母、数字、下划线及连字符',
              },
            ]}
          />
        </Col>
        <Col span={12}>
          <ProFormText
            name="name"
            label="角色名称"
            placeholder="例如：操作主管 / 财务专员"
            rules={[{ required: true, message: '请输入角色名称' }]}
          />
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={16}>
          <ProFormSearchableSelect
            name="dataScope"
            label="数据访问范围"
            options={dataScopeOptions.map((opt) => ({
              label: `${opt.label} —— ${opt.description}`,
              value: opt.value,
            }))}
            rules={[{ required: true, message: '请选择数据范围' }]}
          />
        </Col>
        <Col span={8}>
          {editing && (
            <ProFormSwitch
              name="enabled"
              label="角色状态"
              extra="停用后关联用户将失去此角色权限"
            />
          )}
        </Col>
      </Row>

      <div
        style={{
          marginTop: 8,
          marginBottom: 16,
          border: '1px solid #d9d9d9',
          borderRadius: 6,
          padding: 12,
        }}
      >
        <Text strong>跨公司订单范围</Text>
        <div style={{ marginTop: 8 }}>
          <Text type="secondary">
            指定公司订单默认仅查看；勾选可修改后，仍需同时拥有对应的订单操作权限。
          </Text>
        </div>
        <div style={{ marginTop: 12 }}>
          <Text>可查看的公司</Text>
          <Select
            allowClear
            mode="multiple"
            options={companyOptions}
            placeholder="不选择时仅可访问当前公司订单"
            style={{ display: 'block', width: '100%', marginTop: 4 }}
            value={orderOrganizationAccesses.map(
              (access) => access.organizationId,
            )}
            onChange={(organizationIds: string[]) => {
              setOrderOrganizationAccesses((previous) =>
                organizationIds.map((organizationId) => ({
                  organizationId,
                  writable:
                    previous.find(
                      (access) => access.organizationId === organizationId,
                    )?.writable ?? false,
                })),
              );
            }}
          />
        </div>
        <div style={{ marginTop: 12 }}>
          <Text>其中允许修改的公司</Text>
          <Select
            allowClear
            mode="multiple"
            options={companyOptions.filter((option) =>
              orderOrganizationAccesses.some(
                (access) => access.organizationId === option.value,
              ),
            )}
            placeholder="不选择时跨公司订单均为仅查看"
            style={{ display: 'block', width: '100%', marginTop: 4 }}
            value={orderOrganizationAccesses
              .filter((access) => access.writable)
              .map((access) => access.organizationId)}
            onChange={(organizationIds: string[]) => {
              const writableOrganizationIDs = new Set(organizationIds);
              setOrderOrganizationAccesses((previous) =>
                previous.map((access) => ({
                  ...access,
                  writable: writableOrganizationIDs.has(access.organizationId),
                })),
              );
            }}
          />
        </div>
      </div>

      {/* Permission Configuration Tree Panel */}
      <div style={{ marginTop: 8 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <Space size={8}>
            <Text strong style={{ fontSize: 13, color: 'rgba(0, 0, 0, 0.88)' }}>
              功能权限配置
            </Text>
            <Tag color="blue" variant="filled">
              已选 {selectedPermissionKeys.length} / {allLeafKeys.length} 项
            </Tag>
          </Space>
          <Space size={6}>
            <Button
              size="small"
              icon={<CheckSquareOutlined />}
              onClick={handleSelectAll}
            >
              全选
            </Button>
            <Button
              size="small"
              icon={<MinusSquareOutlined />}
              onClick={handleClearAll}
            >
              清空
            </Button>
            <Button
              size="small"
              icon={<DownOutlined />}
              onClick={handleExpandAll}
            >
              展开全部
            </Button>
            <Button
              size="small"
              icon={<UpOutlined />}
              onClick={handleCollapseAll}
            >
              收起全部
            </Button>
          </Space>
        </div>

        <div
          style={{
            border: '1px solid #f0f0f0',
            borderRadius: 6,
            padding: 12,
            backgroundColor: '#fafafa',
          }}
        >
          <Input
            placeholder="搜索权限名称、权限码或说明..."
            prefix={<SearchOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />}
            allowClear
            size="small"
            value={permissionKeyword}
            onChange={(e) => {
              setPermissionKeyword(e.target.value);
              setAutoExpandParent(true);
              if (e.target.value.trim()) {
                setExpandedKeys(allGroupKeys);
              }
            }}
            style={{ marginBottom: 10, backgroundColor: '#ffffff' }}
          />

          <div
            style={{
              maxHeight: 280,
              overflowY: 'auto',
              backgroundColor: '#ffffff',
              border: '1px solid #f1f5f9',
              borderRadius: 4,
              padding: '8px 10px',
            }}
          >
            {filteredTreeData.length > 0 ? (
              <Tree
                checkable
                blockNode
                treeData={filteredTreeData as unknown as DataNode[]}
                checkedKeys={selectedPermissionKeys}
                expandedKeys={expandedKeys}
                autoExpandParent={autoExpandParent}
                onExpand={(keys) => {
                  setExpandedKeys(keys);
                  setAutoExpandParent(false);
                }}
                onCheck={handleTreeCheck}
                titleRender={(nodeData) => {
                  const node = nodeData as unknown as
                    | PermissionGroupNode
                    | PermissionLeafNode;
                  if ('children' in node && Array.isArray(node.children)) {
                    // Group Node
                    const groupLeaves = node.children;
                    const checkedInGroup = groupLeaves.filter((c) =>
                      selectedPermissionKeys.includes(c.key),
                    ).length;
                    return (
                      <span
                        style={{
                          fontWeight: 600,
                          fontSize: 13,
                          color: '#1e293b',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: 6,
                        }}
                      >
                        <AppstoreOutlined
                          style={{ color: '#1677ff', fontSize: 13 }}
                        />
                        <span>{node.title}</span>
                        <span
                          style={{
                            fontSize: 11,
                            fontWeight: 400,
                            color: checkedInGroup > 0 ? '#1677ff' : '#94a3b8',
                          }}
                        >
                          ({checkedInGroup}/{groupLeaves.length})
                        </span>
                      </span>
                    );
                  }

                  // Leaf Permission Node
                  const leaf = node as PermissionLeafNode;
                  const kw = permissionKeyword.trim().toLowerCase();
                  const isMatched =
                    kw &&
                    (leaf.name.toLowerCase().includes(kw) ||
                      leaf.key.toLowerCase().includes(kw) ||
                      leaf.description?.toLowerCase().includes(kw));

                  return (
                    <span
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 8,
                        fontSize: 12,
                        padding: '1px 0',
                      }}
                    >
                      <span
                        style={{
                          color: isMatched ? '#1677ff' : '#334155',
                          fontWeight: isMatched ? 600 : 400,
                        }}
                      >
                        {leaf.name}
                      </span>
                      <Tag
                        variant="filled"
                        style={{
                          margin: 0,
                          fontSize: 10,
                          lineHeight: '16px',
                          padding: '0 4px',
                          backgroundColor: '#f1f5f9',
                          color: '#64748b',
                          fontFamily: 'monospace',
                        }}
                      >
                        {leaf.key}
                      </Tag>
                      {leaf.requires && leaf.requires.length > 0 && (
                        <Tooltip
                          title={`需先选：${leaf.requires
                            .map((key) => permissionNameByKey[key] ?? key)
                            .join('、')}`}
                        >
                          <Tag
                            variant="filled"
                            style={{
                              margin: 0,
                              fontSize: 10,
                              lineHeight: '16px',
                              padding: '0 4px',
                              backgroundColor: '#eff6ff',
                              color: '#3b82f6',
                              cursor: 'help',
                            }}
                          >
                            需配套
                          </Tag>
                        </Tooltip>
                      )}
                      {leaf.description && (
                        <span
                          style={{
                            fontSize: 11,
                            color: '#94a3b8',
                            maxWidth: 240,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {leaf.description}
                        </span>
                      )}
                    </span>
                  );
                }}
              />
            ) : (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  permissionKeyword ? '未找到匹配的权限项' : '暂无可用权限'
                }
                style={{ margin: '20px 0' }}
              />
            )}
          </div>
        </div>
      </div>
    </ModalForm>
  );
}
