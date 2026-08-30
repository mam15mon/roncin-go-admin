import type {
  AppInstance,
  ConfirmWithReasonOptions,
} from './confirmWithReason';
import { confirmWithReason } from './confirmWithReason';

type Version = number | string;

type VersionedRecord = {
  id?: string;
  version?: Version;
};

type VersionParams<T extends VersionedRecord> = {
  id: string;
  expectedVersion: NonNullable<T['version']>;
};

/** 统一从版本化记录提取接口动作所需的 ID 与乐观锁版本。 */
export function makeVersionActions<T extends VersionedRecord>(
  app: Pick<AppInstance, 'modal' | 'message'>,
) {
  const paramsOf = (record: T): VersionParams<T> | undefined => {
    if (!record.id || !record.version) return undefined;
    return {
      id: record.id,
      expectedVersion: record.version,
    } as VersionParams<T>;
  };

  return {
    run(
      record: T,
      onSubmit: (params: VersionParams<T>, record: T) => Promise<void>,
    ) {
      const params = paramsOf(record);
      if (!params) return Promise.resolve();
      return onSubmit(params, record);
    },
    confirm(
      record: T,
      title: string,
      onSubmit: (
        params: VersionParams<T>,
        reason: string,
        record: T,
      ) => Promise<void>,
      options?: ConfirmWithReasonOptions,
    ) {
      const params = paramsOf(record);
      if (!params) return;
      confirmWithReason(
        app,
        title,
        (reason) => onSubmit(params, reason, record),
        options,
      );
    },
  };
}
