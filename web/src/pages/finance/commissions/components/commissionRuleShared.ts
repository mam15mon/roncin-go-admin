import { unwrapList } from '@/utils/api';

export type EmployeeOption = { label: string; value: string };

export const toEmployeeOptions = (
  response: API.ListCommissionEmployeesResponse,
): EmployeeOption[] =>
  unwrapList(response).flatMap((item) =>
    item.id
      ? [
          {
            value: item.id,
            label: item.displayName ?? item.id,
          },
        ]
      : [],
  );
