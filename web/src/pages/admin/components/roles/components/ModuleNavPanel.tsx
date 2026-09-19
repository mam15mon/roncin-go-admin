import { AppstoreOutlined } from '@ant-design/icons';
import { Space, Tag } from 'antd';
import type { PermissionModule } from '../permissionMatrix';
import { getModuleIcon } from './moduleIcon';

type ModuleNavPanelProps = {
  modules: PermissionModule[];
  allResourcesCount: number;
  selectedModuleId: string;
  selectedSet: Set<string>;
  onSelectModule: (moduleId: string) => void;
};

/** 权限矩阵左侧的服务与功能模块导航。 */
export default function ModuleNavPanel({
  modules,
  allResourcesCount,
  selectedModuleId,
  selectedSet,
  onSelectModule,
}: ModuleNavPanelProps) {
  return (
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
          onSelectModule('all');
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
          {allResourcesCount}
        </span>
      </div>

      {/* 各服务模块列表 */}
      {modules.map((mod) => {
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
        if (modSelectedCount === mod.allKeys.length && mod.allKeys.length > 0) {
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
              onSelectModule(mod.id);
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
  );
}
