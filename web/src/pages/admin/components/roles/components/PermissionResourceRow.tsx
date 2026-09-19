import {
  DownOutlined,
  EditOutlined,
  EyeOutlined,
  UpOutlined,
} from '@ant-design/icons';
import { Button, Checkbox, Space, Tag, Tooltip } from 'antd';
import {
  getResourceState,
  type PermissionAction,
  type PermissionResource,
} from '../permissionMatrix';

type PermissionResourceRowProps = {
  resource: PermissionResource;
  isExpanded: boolean;
  selectedPermissionKeys: string[];
  selectedSet: Set<string>;
  permissionNameByKey: Record<string, string>;
  onToggleExpandResource: (id: string) => void;
  onToggleResourceRead: (
    resource: PermissionResource,
    checked: boolean,
  ) => void;
  onToggleResourceWrite: (
    resource: PermissionResource,
    checked: boolean,
  ) => void;
  onToggleSingleAction: (action: PermissionAction, checked: boolean) => void;
};

/** 权限矩阵中的单个资源行：查看/编辑快捷勾选、状态标签与细粒度操作明细。 */
export default function PermissionResourceRow({
  resource,
  isExpanded,
  selectedPermissionKeys,
  selectedSet,
  permissionNameByKey,
  onToggleExpandResource,
  onToggleResourceRead,
  onToggleResourceWrite,
  onToggleSingleAction,
}: PermissionResourceRowProps) {
  const resState = getResourceState(resource, selectedPermissionKeys);

  return (
    <div
      key={resource.id}
      style={{
        border: isExpanded ? '1px solid #bfdbfe' : '1px solid #f1f5f9',
        borderRadius: 6,
        marginBottom: 8,
        backgroundColor: '#ffffff',
        boxShadow: isExpanded ? '0 1px 3px rgba(0,0,0,0.05)' : 'none',
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
            onChange={(e) => onToggleResourceRead(resource, e.target.checked)}
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
            onChange={(e) => onToggleResourceWrite(resource, e.target.checked)}
          >
            <span
              style={{
                fontSize: 12,
                fontWeight: 500,
                color: resource.writeKeys.length === 0 ? '#cbd5e1' : '#d97706',
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
              自定义 ({resState.selectedCount}/{resState.totalCount})
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
            onClick={() => onToggleExpandResource(resource.id)}
          >
            操作明细 {isExpanded ? <UpOutlined /> : <DownOutlined />}
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
                        backgroundColor: isChecked ? '#eff6ff' : '#ffffff',
                        border: isChecked
                          ? '1px solid #bfdbfe'
                          : '1px solid #e2e8f0',
                        borderRadius: 4,
                      }}
                    >
                      <Checkbox
                        checked={isChecked}
                        onChange={(e) =>
                          onToggleSingleAction(action, e.target.checked)
                        }
                      >
                        <span
                          style={{
                            fontSize: 11,
                            color: isChecked ? '#1d4ed8' : '#334155',
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
                        backgroundColor: isChecked ? '#fffbeb' : '#ffffff',
                        border: isChecked
                          ? '1px solid #fde68a'
                          : '1px solid #e2e8f0',
                        borderRadius: 4,
                      }}
                    >
                      <Checkbox
                        checked={isChecked}
                        onChange={(e) =>
                          onToggleSingleAction(action, e.target.checked)
                        }
                      >
                        <span
                          style={{
                            fontSize: 11,
                            color: isChecked ? '#b45309' : '#334155',
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
                      {action.requires && action.requires.length > 0 && (
                        <Tooltip
                          title={`需前置配套：${action.requires
                            .map((k) => permissionNameByKey[k] ?? k)
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
}
