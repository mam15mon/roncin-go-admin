import { Empty } from 'antd';
import type { PermissionAction, PermissionResource } from '../permissionMatrix';
import PermissionResourceRow from './PermissionResourceRow';

type PermissionResourceListProps = {
  filteredResources: PermissionResource[];
  matchedActionKeys: Set<string>;
  isSearchMode: boolean;
  keyword: string;
  expandedResourceIds: Set<string>;
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

/** 权限矩阵右侧的资源行列表容器与空态。 */
export default function PermissionResourceList({
  filteredResources,
  matchedActionKeys,
  isSearchMode,
  keyword,
  expandedResourceIds,
  selectedPermissionKeys,
  selectedSet,
  permissionNameByKey,
  onToggleExpandResource,
  onToggleResourceRead,
  onToggleResourceWrite,
  onToggleSingleAction,
}: PermissionResourceListProps) {
  return (
    <div
      style={{
        flex: 1,
        overflowY: 'auto',
        padding: '10px 14px',
      }}
    >
      {filteredResources.length > 0 ? (
        filteredResources.map((resource) => {
          const isExpanded =
            expandedResourceIds.has(resource.id) ||
            (isSearchMode &&
              resource.allActions.some((a) => matchedActionKeys.has(a.key)));

          return (
            <PermissionResourceRow
              key={resource.id}
              resource={resource}
              isExpanded={isExpanded}
              selectedPermissionKeys={selectedPermissionKeys}
              selectedSet={selectedSet}
              permissionNameByKey={permissionNameByKey}
              onToggleExpandResource={onToggleExpandResource}
              onToggleResourceRead={onToggleResourceRead}
              onToggleResourceWrite={onToggleResourceWrite}
              onToggleSingleAction={onToggleSingleAction}
            />
          );
        })
      ) : (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={keyword ? '未找到匹配的权限项' : '当前模块暂无可用权限'}
          style={{ margin: '30px 0' }}
        />
      )}
    </div>
  );
}
