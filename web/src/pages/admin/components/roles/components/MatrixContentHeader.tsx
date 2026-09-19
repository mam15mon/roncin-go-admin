import {
  AppstoreOutlined,
  CheckSquareOutlined,
  ClearOutlined,
  DownOutlined,
  EyeOutlined,
  UpOutlined,
} from '@ant-design/icons';
import { Button, Space } from 'antd';
import type { PermissionModule } from '../permissionMatrix';
import { getModuleIcon } from './moduleIcon';

type MatrixContentHeaderProps = {
  currentModule: PermissionModule | null;
  isSearchMode: boolean;
  filteredCount: number;
  selectedSubModule: string;
  onSelectSubModule: (subModule: string) => void;
  onExpandAllActions: () => void;
  onCollapseAllActions: () => void;
  onModuleScopeLevel: (
    module: PermissionModule,
    level: 'read' | 'full' | 'none',
  ) => void;
  onSubModuleScopeLevel: (
    subModuleName: string,
    level: 'read' | 'full' | 'none',
  ) => void;
};

/** 权限矩阵右侧顶部标题栏与二级业务线切换栏。 */
export default function MatrixContentHeader({
  currentModule,
  isSearchMode,
  filteredCount,
  selectedSubModule,
  onSelectSubModule,
  onExpandAllActions,
  onCollapseAllActions,
  onModuleScopeLevel,
  onSubModuleScopeLevel,
}: MatrixContentHeaderProps) {
  return (
    <>
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
          <span style={{ fontWeight: 600, fontSize: 13, color: '#1e293b' }}>
            {isSearchMode
              ? `搜索结果：匹配 ${filteredCount} 个功能`
              : currentModule
                ? currentModule.name
                : '全部功能'}
          </span>
          <span style={{ fontSize: 11, color: '#64748b' }}>
            （展示 {filteredCount} 个资源项）
          </span>
        </Space>

        <Space size={6}>
          <Button
            size="small"
            icon={<DownOutlined />}
            onClick={onExpandAllActions}
          >
            展开操作
          </Button>
          <Button
            size="small"
            icon={<UpOutlined />}
            onClick={onCollapseAllActions}
          >
            收起操作
          </Button>

          {currentModule && !isSearchMode && (
            <>
              <span style={{ color: '#cbd5e1' }}>|</span>
              <Button
                size="small"
                icon={<EyeOutlined />}
                onClick={() => onModuleScopeLevel(currentModule, 'read')}
              >
                本模块只读
              </Button>
              <Button
                size="small"
                icon={<CheckSquareOutlined />}
                onClick={() => onModuleScopeLevel(currentModule, 'full')}
              >
                本模块读写
              </Button>
              <Button
                size="small"
                icon={<ClearOutlined />}
                onClick={() => onModuleScopeLevel(currentModule, 'none')}
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
                onClick={() => onSelectSubModule('all')}
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
                    type={selectedSubModule === sm ? 'primary' : 'default'}
                    onClick={() => onSelectSubModule(sm)}
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
                    onSubModuleScopeLevel(selectedSubModule, 'read')
                  }
                  style={{ fontSize: 11 }}
                >
                  本业务线只读
                </Button>
                <Button
                  size="small"
                  icon={<CheckSquareOutlined />}
                  onClick={() =>
                    onSubModuleScopeLevel(selectedSubModule, 'full')
                  }
                  style={{ fontSize: 11 }}
                >
                  本业务线读写
                </Button>
              </Space>
            )}
          </div>
        )}
    </>
  );
}
