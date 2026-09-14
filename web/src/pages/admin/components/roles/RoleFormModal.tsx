import {
  AppstoreOutlined,
  CheckSquareOutlined,
  ClearOutlined,
  DatabaseOutlined,
  DollarOutlined,
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  IdcardOutlined,
  MinusSquareOutlined,
  SearchOutlined,
  SettingOutlined,
  ShoppingCartOutlined,
  UpOutlined,
} from '@ant-design/icons';
import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSwitch,
  ProFormText,
} from '@ant-design/pro-components';
import {
  App,
  Button,
  Checkbox,
  Col,
  Empty,
  Input,
  Row,
  Space,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useMemo, useState } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import {
  adminServiceCreateRole,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';
import { applyPermissionLinkage } from './permissionLinkage';
import {
  buildPermissionMatrix,
  filterResources,
  getResourceState,
  type PermissionAction,
  type PermissionDefinition,
  type PermissionModule,
  type PermissionResource,
  setBatchScopeLevel,
  toggleResourceRead,
  toggleResourceWrite,
} from './permissionMatrix';
import {
  dataScopeOptions,
  type PermissionTreeNode,
  type RoleFormValues,
} from './roleConstants';

const { Text } = Typography;

function getModuleIcon(name: string) {
  switch (name) {
    case '订单管理':
      return <ShoppingCartOutlined style={{ color: '#1677ff' }} />;
    case '费用管理':
      return <DollarOutlined style={{ color: '#52c41a' }} />;
    case '业务资料':
      return <IdcardOutlined style={{ color: '#722ed1' }} />;
    case '主数据':
      return <DatabaseOutlined style={{ color: '#fa8c16' }} />;
    case '系统管理':
      return <SettingOutlined style={{ color: '#13c2c2' }} />;
    default:
      return <AppstoreOutlined style={{ color: '#64748b' }} />;
  }
}

interface RoleFormModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing?: API.AdminRole;
  formRef: React.RefObject<ProFormInstance | undefined>;
  allLeafKeys?: string[];
  allGroupKeys?: string[];
  filteredTreeData?: PermissionTreeNode[];
  requiresByPermission?: Record<string, string[]>;
  permissionNameByKey?: Record<string, string>;
  selectedPermissionKeys: string[];
  setSelectedPermissionKeys: (keys: string[]) => void;
  expandedKeys?: React.Key[];
  setExpandedKeys?: (keys: React.Key[]) => void;
  autoExpandParent?: boolean;
  setAutoExpandParent?: (val: boolean) => void;
  permissionKeyword?: string;
  setPermissionKeyword?: (kw: string) => void;
  permissions?: API.AdminPermission[];
  onSuccess: () => void;
}

export default function RoleFormModal({
  open,
  onOpenChange,
  editing,
  formRef,
  allLeafKeys = [],
  requiresByPermission = {},
  permissionNameByKey = {},
  selectedPermissionKeys,
  setSelectedPermissionKeys,
  permissionKeyword: controlledKeyword,
  setPermissionKeyword: controlledSetKeyword,
  permissions,
  onSuccess,
}: RoleFormModalProps) {
  const { message } = App.useApp();

  // 当前所选的服务模块与二级业务线
  const [selectedModuleId, setSelectedModuleId] = useState<string>('all');
  const [selectedSubModule, setSelectedSubModule] = useState<string>('all');

  // 本地搜索关键字（与受控关键字同步）
  const [internalKeyword, setInternalKeyword] = useState<string>('');
  const keyword = controlledKeyword ?? internalKeyword;
  const setKeyword = controlledSetKeyword ?? setInternalKeyword;

  // 展开的细粒度操作资源 ID 集合
  const [expandedResourceIds, setExpandedResourceIds] = useState<Set<string>>(
    new Set(),
  );

  // 构造权限矩阵模型：优先使用传入的 permissions，若未提供则降级由 allLeafKeys 等字段构造
  const effectivePermissions: PermissionDefinition[] = useMemo(() => {
    if (permissions && permissions.length > 0) {
      return permissions;
    }
    if (allLeafKeys && allLeafKeys.length > 0) {
      return allLeafKeys.map((key) => ({
        key,
        name: permissionNameByKey[key] ?? key,
        group: '其他功能',
        requires: requiresByPermission[key] ?? [],
      }));
    }
    return [];
  }, [permissions, allLeafKeys, permissionNameByKey, requiresByPermission]);

  const matrixModel = useMemo(
    () => buildPermissionMatrix(effectivePermissions),
    [effectivePermissions],
  );

  // 已选键 Set 索引
  const selectedSet = useMemo(
    () => new Set(selectedPermissionKeys),
    [selectedPermissionKeys],
  );

  // 统计已选查看与编辑数量
  const readSelectedCount = useMemo(
    () => matrixModel.allReadKeys.filter((k) => selectedSet.has(k)).length,
    [matrixModel.allReadKeys, selectedSet],
  );
  const writeSelectedCount = useMemo(
    () => matrixModel.allWriteKeys.filter((k) => selectedSet.has(k)).length,
    [matrixModel.allWriteKeys, selectedSet],
  );

  // 展开/收起细粒度操作
  const toggleExpandResource = (id: string) => {
    setExpandedResourceIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // 全局快捷操作
  const handleSelectAllWrite = () => {
    setSelectedPermissionKeys([...matrixModel.allLeafKeys]);
  };

  const handleSelectAllRead = () => {
    const readOnlyKeys = setBatchScopeLevel(
      matrixModel,
      'read',
      [],
      matrixModel.requiresByPermission,
    );
    setSelectedPermissionKeys(readOnlyKeys);
  };

  const handleClearAll = () => {
    setSelectedPermissionKeys([]);
  };

  // 当前所选模块
  const currentModule = useMemo(() => {
    if (selectedModuleId === 'all') return null;
    return matrixModel.modules.find((m) => m.id === selectedModuleId) ?? null;
  }, [matrixModel.modules, selectedModuleId]);

  // 候选资源集合：搜索模式下全量检索，否则按当前模块和二级业务线过滤
  const isSearchMode = Boolean(keyword.trim());
  const { filteredResources, matchedActionKeys } = useMemo(() => {
    if (isSearchMode) {
      const { filtered, matchedActionKeys } = filterResources(
        matrixModel.allResources,
        keyword,
      );
      return { filteredResources: filtered, matchedActionKeys };
    }

    let list = currentModule
      ? currentModule.resources
      : matrixModel.allResources;

    if (currentModule && selectedSubModule !== 'all') {
      list = list.filter((r) => r.subModule === selectedSubModule);
    }

    return { filteredResources: list, matchedActionKeys: new Set<string>() };
  }, [
    isSearchMode,
    keyword,
    matrixModel.allResources,
    currentModule,
    selectedSubModule,
  ]);

  // 全部展开与收起
  const handleExpandAllActions = () => {
    setExpandedResourceIds(new Set(filteredResources.map((r) => r.id)));
  };

  const handleCollapseAllActions = () => {
    setExpandedResourceIds(new Set());
  };

  // 模块级批量操作
  const handleModuleScopeLevel = (
    module: PermissionModule,
    level: 'read' | 'full' | 'none',
  ) => {
    const next = setBatchScopeLevel(
      module,
      level,
      selectedPermissionKeys,
      matrixModel.requiresByPermission,
    );
    setSelectedPermissionKeys(next);
  };

  // 二级业务线批量操作（例如 海运出口 SE）
  const handleSubModuleScopeLevel = (
    subModuleName: string,
    level: 'read' | 'full' | 'none',
  ) => {
    if (!currentModule) return;
    const subResources = currentModule.resources.filter(
      (r) => r.subModule === subModuleName,
    );
    const scope = {
      readKeys: subResources.flatMap((r) => r.readKeys),
      writeKeys: subResources.flatMap((r) => r.writeKeys),
      allKeys: subResources.flatMap((r) => r.allKeys),
    };
    const next = setBatchScopeLevel(
      scope,
      level,
      selectedPermissionKeys,
      matrixModel.requiresByPermission,
    );
    setSelectedPermissionKeys(next);
  };

  // 切换单个资源的查看权限
  const handleToggleResourceRead = (
    resource: PermissionResource,
    checked: boolean,
  ) => {
    const next = toggleResourceRead(
      resource,
      checked,
      selectedPermissionKeys,
      matrixModel.requiresByPermission,
    );
    setSelectedPermissionKeys(next);
  };

  // 切换单个资源的编辑/管理权限
  const handleToggleResourceWrite = (
    resource: PermissionResource,
    checked: boolean,
  ) => {
    const next = toggleResourceWrite(
      resource,
      checked,
      selectedPermissionKeys,
      matrixModel.requiresByPermission,
    );
    setSelectedPermissionKeys(next);
  };

  // 勾选单个细粒度操作
  const handleToggleSingleAction = (
    action: PermissionAction,
    checked: boolean,
  ) => {
    let next: string[];
    if (checked) {
      next = [...selectedPermissionKeys, action.key];
    } else {
      next = selectedPermissionKeys.filter((k) => k !== action.key);
    }
    setSelectedPermissionKeys(
      applyPermissionLinkage(
        selectedPermissionKeys,
        next,
        matrixModel.requiresByPermission,
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
        width: 1040,
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
              },
            );
            message.success('角色已成功更新');
          } else {
            await adminServiceCreateRole({
              code: '',
              name: values.name?.trim() ?? '',
              dataScope: values.dataScope ?? 2,
              permissionKeys: selectedPermissionKeys,
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

      {/* Huawei Cloud IAM Style Permission Policy Matrix */}
      <div style={{ marginTop: 8 }}>
        {/* Top Header & Global Actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <Space size={8}>
            <div
              style={{
                width: 3,
                height: 15,
                backgroundColor: '#1677ff',
                borderRadius: 2,
              }}
            />
            <Text strong style={{ fontSize: 13, color: '#1e293b' }}>
              功能权限配置（华为云 IAM 策略模式）
            </Text>
            <Tag color="blue" variant="filled">
              已选 {selectedPermissionKeys.length} /{' '}
              {matrixModel.allLeafKeys.length} 项
            </Tag>
            <Tag color="cyan" variant="filled">
              查看 (只读): {readSelectedCount} 项
            </Tag>
            <Tag color="orange" variant="filled">
              编辑/管理: {writeSelectedCount} 项
            </Tag>
          </Space>

          <Space size={6}>
            <Button
              size="small"
              type="primary"
              ghost
              icon={<CheckSquareOutlined />}
              onClick={handleSelectAllWrite}
            >
              全选读写
            </Button>
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={handleSelectAllRead}
            >
              全选只读
            </Button>
            <Button
              size="small"
              danger
              ghost
              icon={<MinusSquareOutlined />}
              onClick={handleClearAll}
            >
              清空
            </Button>
          </Space>
        </div>

        {/* Search Bar */}
        <Input
          placeholder="搜索功能对象、业务线或权限操作（如：海运、用户、账单、delete）..."
          prefix={<SearchOutlined style={{ color: 'rgba(0, 0, 0, 0.45)' }} />}
          allowClear
          size="small"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          style={{ marginBottom: 10, backgroundColor: '#ffffff' }}
        />

        {/* Matrix Main Layout */}
        <div
          style={{
            display: 'flex',
            height: 460,
            border: '1px solid #e2e8f0',
            borderRadius: 6,
            backgroundColor: '#ffffff',
            overflow: 'hidden',
          }}
        >
          {/* Left Service Modules Nav */}
          <div
            style={{
              width: 195,
              backgroundColor: '#f8fafc',
              borderRight: '1px solid #e2e8f0',
              display: 'flex',
              flexDirection: 'column',
              overflowY: 'auto',
            }}
          >
            <div
              style={{
                padding: '8px 12px',
                fontSize: 11,
                fontWeight: 600,
                color: '#64748b',
                borderBottom: '1px solid #f1f5f9',
              }}
            >
              服务与功能模块
            </div>

            {/* "全部功能" 项 */}
            <div
              onClick={() => {
                setSelectedModuleId('all');
                setSelectedSubModule('all');
              }}
              style={{
                padding: '9px 12px',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                backgroundColor:
                  selectedModuleId === 'all' ? '#e6f4ff' : 'transparent',
                color: selectedModuleId === 'all' ? '#1677ff' : '#334155',
                fontWeight: selectedModuleId === 'all' ? 600 : 400,
                borderLeft:
                  selectedModuleId === 'all'
                    ? '3px solid #1677ff'
                    : '3px solid transparent',
                transition: 'all 0.15s',
              }}
            >
              <Space size={6}>
                <AppstoreOutlined
                  style={{
                    color: selectedModuleId === 'all' ? '#1677ff' : '#94a3b8',
                  }}
                />
                <span style={{ fontSize: 12 }}>全部功能</span>
              </Space>
              <span style={{ fontSize: 11, color: '#94a3b8' }}>
                {matrixModel.allResources.length}
              </span>
            </div>

            {/* 各服务模块列表 */}
            {matrixModel.modules.map((mod) => {
              const isActive = selectedModuleId === mod.id;
              const modSelectedCount = mod.allKeys.filter((k) =>
                selectedSet.has(k),
              ).length;
              const modReadCount = mod.readKeys.filter((k) =>
                selectedSet.has(k),
              ).length;
              const modWriteCount = mod.writeKeys.filter((k) =>
                selectedSet.has(k),
              ).length;

              let statusTag = (
                <span style={{ fontSize: 10, color: '#94a3b8' }}>未授权</span>
              );
              if (
                modSelectedCount === mod.allKeys.length &&
                mod.allKeys.length > 0
              ) {
                statusTag = (
                  <Tag
                    color="green"
                    variant="filled"
                    style={{
                      margin: 0,
                      fontSize: 10,
                      lineHeight: '16px',
                      padding: '0 4px',
                    }}
                  >
                    读写
                  </Tag>
                );
              } else if (
                modReadCount === mod.readKeys.length &&
                modWriteCount === 0 &&
                modReadCount > 0
              ) {
                statusTag = (
                  <Tag
                    color="blue"
                    variant="filled"
                    style={{
                      margin: 0,
                      fontSize: 10,
                      lineHeight: '16px',
                      padding: '0 4px',
                    }}
                  >
                    只读
                  </Tag>
                );
              } else if (modSelectedCount > 0) {
                statusTag = (
                  <Tag
                    color="orange"
                    variant="filled"
                    style={{
                      margin: 0,
                      fontSize: 10,
                      lineHeight: '16px',
                      padding: '0 4px',
                    }}
                  >
                    {modSelectedCount}/{mod.allKeys.length}
                  </Tag>
                );
              }

              return (
                <div
                  key={mod.id}
                  onClick={() => {
                    setSelectedModuleId(mod.id);
                    setSelectedSubModule('all');
                  }}
                  style={{
                    padding: '9px 12px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    backgroundColor: isActive ? '#e6f4ff' : 'transparent',
                    color: isActive ? '#1677ff' : '#334155',
                    fontWeight: isActive ? 600 : 400,
                    borderLeft: isActive
                      ? '3px solid #1677ff'
                      : '3px solid transparent',
                    transition: 'all 0.15s',
                  }}
                >
                  <Space size={6} style={{ overflow: 'hidden' }}>
                    {getModuleIcon(mod.name)}
                    <span
                      style={{
                        fontSize: 12,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                      }}
                    >
                      {mod.name}
                    </span>
                  </Space>
                  {statusTag}
                </div>
              );
            })}
          </div>

          {/* Right Resource Matrix & Action List */}
          <div
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              backgroundColor: '#ffffff',
            }}
          >
            {/* Top Sub-Header */}
            <div
              style={{
                padding: '8px 14px',
                borderBottom: '1px solid #f0f0f0',
                backgroundColor: '#fafafa',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
              }}
            >
              <Space size={8}>
                {currentModule ? (
                  getModuleIcon(currentModule.name)
                ) : (
                  <AppstoreOutlined style={{ color: '#1677ff' }} />
                )}
                <span
                  style={{ fontWeight: 600, fontSize: 13, color: '#1e293b' }}
                >
                  {isSearchMode
                    ? `搜索结果：匹配 ${filteredResources.length} 个功能`
                    : currentModule
                      ? currentModule.name
                      : '全部功能'}
                </span>
                <span style={{ fontSize: 11, color: '#64748b' }}>
                  （展示 {filteredResources.length} 个资源项）
                </span>
              </Space>

              <Space size={6}>
                <Button
                  size="small"
                  icon={<DownOutlined />}
                  onClick={handleExpandAllActions}
                >
                  展开操作
                </Button>
                <Button
                  size="small"
                  icon={<UpOutlined />}
                  onClick={handleCollapseAllActions}
                >
                  收起操作
                </Button>

                {currentModule && !isSearchMode && (
                  <>
                    <span style={{ color: '#cbd5e1' }}>|</span>
                    <Button
                      size="small"
                      icon={<EyeOutlined />}
                      onClick={() =>
                        handleModuleScopeLevel(currentModule, 'read')
                      }
                    >
                      本模块只读
                    </Button>
                    <Button
                      size="small"
                      icon={<CheckSquareOutlined />}
                      onClick={() =>
                        handleModuleScopeLevel(currentModule, 'full')
                      }
                    >
                      本模块读写
                    </Button>
                    <Button
                      size="small"
                      icon={<ClearOutlined />}
                      onClick={() =>
                        handleModuleScopeLevel(currentModule, 'none')
                      }
                    >
                      清空本模块
                    </Button>
                  </>
                )}
              </Space>
            </div>

            {/* SubModule Bar (e.g. 订单管理的6条业务线 / 往来单位) */}
            {currentModule &&
              currentModule.subModules.length > 0 &&
              !isSearchMode && (
                <div
                  style={{
                    padding: '6px 14px',
                    backgroundColor: '#f8fafc',
                    borderBottom: '1px solid #f0f0f0',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                  }}
                >
                  <Space size={4} wrap>
                    <Button
                      size="small"
                      type={selectedSubModule === 'all' ? 'primary' : 'default'}
                      onClick={() => setSelectedSubModule('all')}
                      style={{ fontSize: 11 }}
                    >
                      全部业务线 ({currentModule.resources.length})
                    </Button>
                    {currentModule.subModules.map((sm) => {
                      const count = currentModule.resources.filter(
                        (r) => r.subModule === sm,
                      ).length;
                      return (
                        <Button
                          key={sm}
                          size="small"
                          type={
                            selectedSubModule === sm ? 'primary' : 'default'
                          }
                          onClick={() => setSelectedSubModule(sm)}
                          style={{ fontSize: 11 }}
                        >
                          {sm} ({count})
                        </Button>
                      );
                    })}
                  </Space>

                  {selectedSubModule !== 'all' && (
                    <Space size={4}>
                      <Button
                        size="small"
                        icon={<EyeOutlined />}
                        onClick={() =>
                          handleSubModuleScopeLevel(selectedSubModule, 'read')
                        }
                        style={{ fontSize: 11 }}
                      >
                        本业务线只读
                      </Button>
                      <Button
                        size="small"
                        icon={<CheckSquareOutlined />}
                        onClick={() =>
                          handleSubModuleScopeLevel(selectedSubModule, 'full')
                        }
                        style={{ fontSize: 11 }}
                      >
                        本业务线读写
                      </Button>
                    </Space>
                  )}
                </div>
              )}

            {/* Resource Rows Matrix Container */}
            <div
              style={{
                flex: 1,
                overflowY: 'auto',
                padding: '10px 14px',
              }}
            >
              {filteredResources.length > 0 ? (
                filteredResources.map((resource) => {
                  const resState = getResourceState(
                    resource,
                    selectedPermissionKeys,
                  );
                  const isExpanded =
                    expandedResourceIds.has(resource.id) ||
                    (isSearchMode &&
                      resource.allActions.some((a) =>
                        matchedActionKeys.has(a.key),
                      ));

                  return (
                    <div
                      key={resource.id}
                      style={{
                        border: isExpanded
                          ? '1px solid #bfdbfe'
                          : '1px solid #f1f5f9',
                        borderRadius: 6,
                        marginBottom: 8,
                        backgroundColor: '#ffffff',
                        boxShadow: isExpanded
                          ? '0 1px 3px rgba(0,0,0,0.05)'
                          : 'none',
                        transition: 'all 0.15s',
                      }}
                    >
                      {/* Row Header */}
                      <div
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          padding: '7px 12px',
                          backgroundColor: isExpanded ? '#f0f7ff' : '#ffffff',
                          borderTopLeftRadius: 5,
                          borderTopRightRadius: 5,
                          borderBottomLeftRadius: isExpanded ? 0 : 5,
                          borderBottomRightRadius: isExpanded ? 0 : 5,
                        }}
                      >
                        {/* Left: Resource Name */}
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            minWidth: 210,
                          }}
                        >
                          {resource.subModule && (
                            <Tag
                              color="blue"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 10,
                                lineHeight: '16px',
                                padding: '0 4px',
                              }}
                            >
                              {resource.subModule}
                            </Tag>
                          )}
                          <span
                            style={{
                              fontWeight: 600,
                              fontSize: 12,
                              color: '#1e293b',
                            }}
                          >
                            {resource.name}
                          </span>
                          <span style={{ fontSize: 10, color: '#94a3b8' }}>
                            ({resource.allKeys.length}项)
                          </span>
                        </div>

                        {/* Center: Quick Read / Write Checkboxes */}
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 20,
                          }}
                        >
                          <Checkbox
                            checked={resState.readState === 'all'}
                            indeterminate={resState.readState === 'some'}
                            onChange={(e) =>
                              handleToggleResourceRead(
                                resource,
                                e.target.checked,
                              )
                            }
                          >
                            <span
                              style={{
                                fontSize: 12,
                                fontWeight: 500,
                                color: '#1677ff',
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: 4,
                              }}
                            >
                              <EyeOutlined /> 查看 (只读)
                            </span>
                          </Checkbox>

                          <Checkbox
                            disabled={resource.writeKeys.length === 0}
                            checked={resState.writeState === 'all'}
                            indeterminate={resState.writeState === 'some'}
                            onChange={(e) =>
                              handleToggleResourceWrite(
                                resource,
                                e.target.checked,
                              )
                            }
                          >
                            <span
                              style={{
                                fontSize: 12,
                                fontWeight: 500,
                                color:
                                  resource.writeKeys.length === 0
                                    ? '#cbd5e1'
                                    : '#d97706',
                                display: 'inline-flex',
                                alignItems: 'center',
                                gap: 4,
                              }}
                            >
                              <EditOutlined /> 编辑 / 管理
                            </span>
                          </Checkbox>
                        </div>

                        {/* Right: Status Tag & Expand Button */}
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                          }}
                        >
                          {resState.overallLevel === 'none' && (
                            <Tag
                              style={{
                                margin: 0,
                                fontSize: 11,
                                color: '#94a3b8',
                              }}
                            >
                              未授权
                            </Tag>
                          )}
                          {resState.overallLevel === 'read' && (
                            <Tag
                              color="blue"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 11,
                                fontWeight: 500,
                              }}
                            >
                              只读
                            </Tag>
                          )}
                          {resState.overallLevel === 'full' && (
                            <Tag
                              color="green"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 11,
                                fontWeight: 500,
                              }}
                            >
                              读写
                            </Tag>
                          )}
                          {resState.overallLevel === 'custom' && (
                            <Tag
                              color="orange"
                              variant="filled"
                              style={{
                                margin: 0,
                                fontSize: 11,
                                fontWeight: 500,
                              }}
                            >
                              自定义 ({resState.selectedCount}/
                              {resState.totalCount})
                            </Tag>
                          )}

                          <Button
                            type="link"
                            size="small"
                            style={{
                              fontSize: 11,
                              padding: '0 4px',
                              color: isExpanded ? '#1677ff' : '#64748b',
                            }}
                            onClick={() => toggleExpandResource(resource.id)}
                          >
                            操作明细{' '}
                            {isExpanded ? <UpOutlined /> : <DownOutlined />}
                          </Button>
                        </div>
                      </div>

                      {/* Expanded Fine-grained Actions */}
                      {isExpanded && (
                        <div
                          style={{
                            padding: '8px 12px',
                            backgroundColor: '#fafbfc',
                            borderTop: '1px dashed #e2e8f0',
                            borderBottomLeftRadius: 5,
                            borderBottomRightRadius: 5,
                          }}
                        >
                          {/* 查看权限列表 */}
                          {resource.readActions.length > 0 && (
                            <div style={{ marginBottom: 6 }}>
                              <div
                                style={{
                                  fontSize: 11,
                                  fontWeight: 600,
                                  color: '#1677ff',
                                  marginBottom: 4,
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 4,
                                }}
                              >
                                <EyeOutlined /> 查看权限（只读）
                              </div>
                              <Space wrap size={[8, 6]}>
                                {resource.readActions.map((action) => {
                                  const isChecked = selectedSet.has(action.key);
                                  return (
                                    <div
                                      key={action.key}
                                      style={{
                                        display: 'inline-flex',
                                        alignItems: 'center',
                                        gap: 4,
                                        padding: '2px 6px',
                                        backgroundColor: isChecked
                                          ? '#eff6ff'
                                          : '#ffffff',
                                        border: isChecked
                                          ? '1px solid #bfdbfe'
                                          : '1px solid #e2e8f0',
                                        borderRadius: 4,
                                      }}
                                    >
                                      <Checkbox
                                        checked={isChecked}
                                        onChange={(e) =>
                                          handleToggleSingleAction(
                                            action,
                                            e.target.checked,
                                          )
                                        }
                                      >
                                        <span
                                          style={{
                                            fontSize: 11,
                                            color: isChecked
                                              ? '#1d4ed8'
                                              : '#334155',
                                            fontWeight: isChecked ? 600 : 400,
                                          }}
                                        >
                                          {action.name}
                                        </span>
                                      </Checkbox>
                                      <Tag
                                        style={{
                                          margin: 0,
                                          fontSize: 10,
                                          lineHeight: '14px',
                                          padding: '0 3px',
                                          fontFamily: 'monospace',
                                          backgroundColor: '#f1f5f9',
                                        }}
                                      >
                                        {action.key}
                                      </Tag>
                                      {action.description && (
                                        <Tooltip title={action.description}>
                                          <span
                                            style={{
                                              fontSize: 10,
                                              color: '#94a3b8',
                                              cursor: 'help',
                                            }}
                                          >
                                            ⓘ
                                          </span>
                                        </Tooltip>
                                      )}
                                    </div>
                                  );
                                })}
                              </Space>
                            </div>
                          )}

                          {/* 编辑与管理权限列表 */}
                          {resource.writeActions.length > 0 && (
                            <div>
                              <div
                                style={{
                                  fontSize: 11,
                                  fontWeight: 600,
                                  color: '#d97706',
                                  marginBottom: 4,
                                  display: 'flex',
                                  alignItems: 'center',
                                  gap: 4,
                                }}
                              >
                                <EditOutlined /> 编辑与管理权限（读写/特殊操作）
                              </div>
                              <Space wrap size={[8, 6]}>
                                {resource.writeActions.map((action) => {
                                  const isChecked = selectedSet.has(action.key);
                                  return (
                                    <div
                                      key={action.key}
                                      style={{
                                        display: 'inline-flex',
                                        alignItems: 'center',
                                        gap: 4,
                                        padding: '2px 6px',
                                        backgroundColor: isChecked
                                          ? '#fffbeb'
                                          : '#ffffff',
                                        border: isChecked
                                          ? '1px solid #fde68a'
                                          : '1px solid #e2e8f0',
                                        borderRadius: 4,
                                      }}
                                    >
                                      <Checkbox
                                        checked={isChecked}
                                        onChange={(e) =>
                                          handleToggleSingleAction(
                                            action,
                                            e.target.checked,
                                          )
                                        }
                                      >
                                        <span
                                          style={{
                                            fontSize: 11,
                                            color: isChecked
                                              ? '#b45309'
                                              : '#334155',
                                            fontWeight: isChecked ? 600 : 400,
                                          }}
                                        >
                                          {action.name}
                                        </span>
                                      </Checkbox>
                                      <Tag
                                        style={{
                                          margin: 0,
                                          fontSize: 10,
                                          lineHeight: '14px',
                                          padding: '0 3px',
                                          fontFamily: 'monospace',
                                          backgroundColor: '#f1f5f9',
                                        }}
                                      >
                                        {action.key}
                                      </Tag>
                                      {action.requires &&
                                        action.requires.length > 0 && (
                                          <Tooltip
                                            title={`需前置配套：${action.requires
                                              .map(
                                                (k) =>
                                                  matrixModel
                                                    .permissionNameByKey[k] ??
                                                  k,
                                              )
                                              .join('、')}`}
                                          >
                                            <Tag
                                              color="blue"
                                              style={{
                                                margin: 0,
                                                fontSize: 10,
                                                lineHeight: '14px',
                                                padding: '0 3px',
                                                cursor: 'help',
                                              }}
                                            >
                                              需配套
                                            </Tag>
                                          </Tooltip>
                                        )}
                                      {action.description && (
                                        <Tooltip title={action.description}>
                                          <span
                                            style={{
                                              fontSize: 10,
                                              color: '#94a3b8',
                                              cursor: 'help',
                                            }}
                                          >
                                            ⓘ
                                          </span>
                                        </Tooltip>
                                      )}
                                    </div>
                                  );
                                })}
                              </Space>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })
              ) : (
                <Empty
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  description={
                    keyword ? '未找到匹配的权限项' : '当前模块暂无可用权限'
                  }
                  style={{ margin: '30px 0' }}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </ModalForm>
  );
}
