import type { ProColumns } from '@ant-design/pro-components';
import { useQuery } from '@tanstack/react-query';
import { Alert, Tag } from 'antd';
import React from 'react';
import { useAccess } from '@/app/access';
import { SettingTableTemplate } from '@/components/ui';
import { MasterDataKind } from '@/enums.generated';
import { getCurrencies } from '@/features/master-data/currencies';
import {
  feeCatalogServiceCreateFeeSettingTemplate,
  feeCatalogServiceListBillingUnits,
  feeCatalogServiceListFeeSettingTemplates,
  feeCatalogServiceUpdateFeeSettingTemplate,
} from '@/services/roncin/feeCatalogService';
import { masterDataServiceListOptions } from '@/services/roncin/masterDataService';
import { unwrapList } from '@/utils/api';
import FeeSettingFields from './FeeSettingFields';

type TemplateRow = API.FeeSettingTemplateInput & { id?: string };

export default function FeeTemplatesPanel() {
  const access = useAccess();
  const { data: options } = useQuery({
    queryKey: ['fee-template-options'],
    queryFn: async () => {
      const [units, currencies, items] = await Promise.all([
        feeCatalogServiceListBillingUnits({ page: 1, pageSize: 200 }),
        getCurrencies(),
        masterDataServiceListOptions(),
      ]);
      return {
        billingUnits: unwrapList(units).filter((item) => item.enabled),
        currencies: currencies.filter((item) => item.enabled),
        items: unwrapList(items).filter((item) => item.enabled),
      };
    },
  });
  const columns: ProColumns<TemplateRow>[] = [
    { title: '费用代码', dataIndex: 'feeCode', width: 140 },
    { title: '费用名称', dataIndex: 'nameZh', width: 180 },
    { title: '英文名称', dataIndex: 'nameEn', width: 180 },
    { title: '默认币种', dataIndex: 'defaultCurrency', width: 100 },
    { title: '默认开票项目', dataIndex: 'taxableServiceName', width: 180 },
    { title: '税率（%）', dataIndex: 'taxRate', width: 100 },
    {
      title: '状态',
      dataIndex: 'enabled',
      width: 80,
      render: (_, row) =>
        row.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>,
    },
  ];
  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        title="新公司创建时使用此目录初始化费用科目与开票项目；后续修改不影响已有公司配置。"
      />
      <SettingTableTemplate<TemplateRow, API.FeeSettingTemplateInput>
        entityName="初始费用目录"
        columns={columns}
        pagination={{ pageSize: 20 }}
        modalWidth={840}
        grid
        scroll={{ x: 1100 }}
        canCreate={access.isSystemWorkspace && access.canCreateFeeSettings}
        canUpdate={access.isSystemWorkspace && access.canUpdateFeeSettings}
        query={async (params) => {
          const response = await feeCatalogServiceListFeeSettingTemplates({
            page: Number(params?.current ?? 1),
            pageSize: Number(params?.pageSize ?? 20),
          });
          return {
            ...response,
            data: unwrapList(response).map((row) => ({
              ...row.input,
              id: row.id,
            })),
          };
        }}
        createItem={(input) =>
          feeCatalogServiceCreateFeeSettingTemplate({ input })
        }
        updateItem={(row, input) => {
          if (!row.id) throw new Error('费用目录缺少标识');
          return feeCatalogServiceUpdateFeeSettingTemplate(
            { id: row.id },
            { id: row.id, input },
          );
        }}
        initialValues={(row) =>
          row ?? {
            enabled: true,
            sortOrder: 100,
            defaultCurrency: 'CNY',
            taxRate: '0',
            taxableServiceDefaultTaxRate: '0',
          }
        }
        renderFormItems={(row) => (
          <FeeSettingFields
            template
            editing={Boolean(row)}
            billingUnits={options?.billingUnits ?? []}
            currencies={options?.currencies ?? []}
            serviceTypes={
              options?.items.filter(
                (item) =>
                  item.kind === MasterDataKind.MASTER_DATA_KIND_CHARGE_CATEGORY,
              ) ?? []
            }
            abnormalCases={
              options?.items.filter(
                (item) =>
                  item.kind === MasterDataKind.MASTER_DATA_KIND_ABNORMAL_CASE,
              ) ?? []
            }
          />
        )}
      />
    </>
  );
}
