import type { ProFormInstance } from '@ant-design/pro-components';
import { ModalForm } from '@ant-design/pro-components';
import { App } from 'antd';
import React, { useMemo, useState } from 'react';
import {
  adminServiceCreateRole,
  adminServiceUpdateRole,
} from '@/services/roncin/adminService';
import MatrixContentHeader from './components/MatrixContentHeader';
import ModuleNavPanel from './components/ModuleNavPanel';
import PermissionMatrixHeader from './components/PermissionMatrixHeader';
import PermissionResourceList from './components/PermissionResourceList';
import RoleBasicFormFields from './components/RoleBasicFormFields';
import { applyPermissionLinkage } from './permissionLinkage';
import {
  buildPermissionMatrix,
  filterResources,
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
        destroyOnHidden: true,
        width: 1040,
        onCancel: () => onOpenChange(false),
      }}
      onOpenChange={onOpenChange}
      onFinish={async (values) => {
        if (
          !dataScopeOptions.some((option) => option.value === values.dataScope)
        ) {
          message.error('请选择有效的数据范围；仅本人范围已停用');
          return false;
        }
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
      <RoleBasicFormFields editing={editing} />

      {/* Huawei Cloud IAM Style Permission Policy Matrix */}
      <div style={{ marginTop: 8 }}>
        <PermissionMatrixHeader
          selectedCount={selectedPermissionKeys.length}
          totalCount={matrixModel.allLeafKeys.length}
          readSelectedCount={readSelectedCount}
          writeSelectedCount={writeSelectedCount}
          keyword={keyword}
          setKeyword={setKeyword}
          onSelectAllWrite={handleSelectAllWrite}
          onSelectAllRead={handleSelectAllRead}
          onClearAll={handleClearAll}
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
          <ModuleNavPanel
            modules={matrixModel.modules}
            allResourcesCount={matrixModel.allResources.length}
            selectedModuleId={selectedModuleId}
            selectedSet={selectedSet}
            onSelectModule={(moduleId) => {
              setSelectedModuleId(moduleId);
              setSelectedSubModule('all');
            }}
          />

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
            <MatrixContentHeader
              currentModule={currentModule}
              isSearchMode={isSearchMode}
              filteredCount={filteredResources.length}
              selectedSubModule={selectedSubModule}
              onSelectSubModule={setSelectedSubModule}
              onExpandAllActions={handleExpandAllActions}
              onCollapseAllActions={handleCollapseAllActions}
              onModuleScopeLevel={handleModuleScopeLevel}
              onSubModuleScopeLevel={handleSubModuleScopeLevel}
            />

            {/* Resource Rows Matrix Container */}
            <PermissionResourceList
              filteredResources={filteredResources}
              matchedActionKeys={matchedActionKeys}
              isSearchMode={isSearchMode}
              keyword={keyword}
              expandedResourceIds={expandedResourceIds}
              selectedPermissionKeys={selectedPermissionKeys}
              selectedSet={selectedSet}
              permissionNameByKey={matrixModel.permissionNameByKey}
              onToggleExpandResource={toggleExpandResource}
              onToggleResourceRead={handleToggleResourceRead}
              onToggleResourceWrite={handleToggleResourceWrite}
              onToggleSingleAction={handleToggleSingleAction}
            />
          </div>
        </div>
      </div>
    </ModalForm>
  );
}
