export interface BackgroundTaskPresentation {
  label: string;
  description: string;
  color: string;
}

const genericTaskPresentation: Record<number, BackgroundTaskPresentation> = {
  [BackgroundTaskKind.BACKGROUND_TASK_KIND_MASTER_DATA_IMPORT]: {
    label: '主数据导入',
    description: '导入并处理主数据文件',
    color: 'blue',
  },
  [BackgroundTaskKind.BACKGROUND_TASK_KIND_UNLOCODE_IMPORT]: {
    label: 'UNLOCODE 导入',
    description: '同步联合国口岸位置代码',
    color: 'cyan',
  },
  [BackgroundTaskKind.BACKGROUND_TASK_KIND_ORDER_REMINDER]: {
    label: '订单提醒',
    description: '按计划发送订单业务提醒',
    color: 'orange',
  },
  [BackgroundTaskKind.BACKGROUND_TASK_KIND_INTEGRATION]: {
    label: '外部系统集成',
    description: '与外部系统交换业务数据',
    color: 'purple',
  },
  [BackgroundTaskKind.BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION]: {
    label: '钉钉业务通知',
    description: '通过钉钉企业机器人发送消息',
    color: 'geekblue',
  },
};

export function backgroundTaskPresentation(
  record: API.BackgroundTask,
): BackgroundTaskPresentation {
  if (
    record.kind ===
    BackgroundTaskKind.BACKGROUND_TASK_KIND_DINGTALK_NOTIFICATION
  ) {
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
    case BackgroundTaskStatus.BACKGROUND_TASK_STATUS_PENDING:
      return attempts === 0 ? '等待首次执行' : `等待第 ${attempts + 1} 次执行`;
    case BackgroundTaskStatus.BACKGROUND_TASK_STATUS_RUNNING:
      return attempts === 0 ? '首次执行中' : `第 ${attempts + 1} 次执行中`;
    case BackgroundTaskStatus.BACKGROUND_TASK_STATUS_SUCCEEDED:
      return attempts === 0
        ? '首次执行成功，无重试'
        : `失败 ${attempts} 次后执行成功`;
    case BackgroundTaskStatus.BACKGROUND_TASK_STATUS_FAILED:
      return `已失败 ${attempts} 次，等待自动重试`;
    case BackgroundTaskStatus.BACKGROUND_TASK_STATUS_DEAD_LETTER:
      return `连续失败 ${attempts} 次，已停止自动重试`;
    default:
      return '执行状态未知';
  }
}

export function backgroundTaskHasNextRunAt(
  record: API.BackgroundTask,
): boolean {
  return (
    record.status === BackgroundTaskStatus.BACKGROUND_TASK_STATUS_PENDING ||
    record.status === BackgroundTaskStatus.BACKGROUND_TASK_STATUS_FAILED
  );
}
import { BackgroundTaskKind, BackgroundTaskStatus } from '@/enums.generated';
