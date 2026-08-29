export const seStatusTabs = [
  { key: 'all', label: '全部订单' },
  { key: 'draft', label: '草稿/待订舱', badgeColor: '#d9d9d9' },
  { key: 'booked', label: '已订舱', badgeColor: '#1677ff' },
  { key: 'allocated', label: '已配舱', badgeColor: '#13c2c2' },
  { key: 'trucking', label: '拖车安排', badgeColor: '#722ed1' },
  { key: 'cutoff', label: '已截单', badgeColor: '#eb2f96' },
  { key: 'customs', label: '报关放行', badgeColor: '#2f54eb' },
  { key: 'released', label: '已放单', badgeColor: '#52c41a' },
  { key: 'terminating', label: '退关中', badgeColor: '#fa8c16' },
  { key: 'terminated', label: '已退关', badgeColor: '#ff4d4f' },
  { key: 'completed', label: '已完结', badgeColor: '#52c41a' },
  { key: 'abnormal', label: '异常挂起', badgeColor: '#ff4d4f' },
];

export const lifecycleFiltersByStage: Record<
  string,
  {
    flowStatus?: number;
    terminationStatus?: number;
    closureStatus?: number;
    hasActiveException?: boolean;
  }
> = {
  draft: { flowStatus: 1 },
  booked: { flowStatus: 2 },
  allocated: { flowStatus: 3 },
  trucking: { flowStatus: 4 },
  cutoff: { flowStatus: 5 },
  customs: { flowStatus: 6 },
  released: { flowStatus: 7 },
  terminating: { terminationStatus: 2 },
  terminated: { terminationStatus: 3 },
  completed: { closureStatus: 2 },
  abnormal: { hasActiveException: true },
  unreturned: { terminationStatus: 1, closureStatus: 1 },
  returned: { terminationStatus: 3 },
};
