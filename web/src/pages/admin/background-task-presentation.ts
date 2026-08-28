export interface BackgroundTaskPresentation {
  label: string;
  description: string;
  color: string;
}

const genericTaskPresentation: Record<number, BackgroundTaskPresentation> = {
  1: {
    label: '主数据导入',
    description: '导入并处理主数据文件',
    color: 'blue',
  },
  2: {
    label: 'UNLOCODE 导入',
    description: '同步联合国口岸位置代码',
    color: 'cyan',
  },
  3: {
    label: '订单提醒',
    description: '按计划发送订单业务提醒',
    color: 'orange',
  },
  4: {
    label: '外部系统集成',
    description: '与外部系统交换业务数据',
    color: 'purple',
  },
  5: {
    label: '钉钉业务通知',
    description: '通过钉钉企业机器人发送消息',
    color: 'geekblue',
  },
};

export function backgroundTaskPresentation(
  record: API.BackgroundTask,
): BackgroundTaskPresentation {
  if (record.kind === 5) {
    if (record.idempotencyKey?.startsWith('user-authorized:')) {
      return {
        label: '账号授权完成通知',
        description: '通知用户账号已经完成组织和角色授权',
        color: 'green',
      };
    }
    if (record.idempotencyKey?.startsWith('order-personnel:')) {
      return {
        label: '订单人员分配通知',
        description: '通知相关人员参与订单协作',
        color: 'geekblue',
      };
    }
  }
  return (
    genericTaskPresentation[record.kind ?? 0] ?? {
      label: '未知任务',
      description: '当前版本无法识别该任务类型',
      color: 'default',
    }
  );
}

export function backgroundTaskExecutionSummary(
  record: API.BackgroundTask,
): string {
  const attempts = record.attempts ?? 0;
  switch (record.status) {
    case 1:
      return attempts === 0 ? '等待首次执行' : `等待第 ${attempts + 1} 次执行`;
    case 2:
      return attempts === 0 ? '首次执行中' : `第 ${attempts + 1} 次执行中`;
    case 3:
      return attempts === 0
        ? '首次执行成功，无重试'
        : `失败 ${attempts} 次后执行成功`;
    case 4:
      return `已失败 ${attempts} 次，等待自动重试`;
    case 5:
      return `连续失败 ${attempts} 次，已停止自动重试`;
    default:
      return '执行状态未知';
  }
}

export function backgroundTaskHasNextRunAt(
  record: API.BackgroundTask,
): boolean {
  return record.status === 1 || record.status === 4;
}
