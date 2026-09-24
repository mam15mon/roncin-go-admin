import { Alert, Form, Radio, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import React, { useEffect, useRef } from 'react';

import { useColumnSettings } from '@/components/ui/column-settings';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import type { SeaExternalConfirmationFormValues } from './SeaExternalConfirmationFields';

import {
  DEFAULT_FREIGHT_TERMS,
  DEFAULT_TRANSPORT_TERMS,
} from './seaDocumentSectionConstants';

export type ModeChangeFormValues = SeaExternalConfirmationFormValues & {
  reason?: string;
  newHouseBill?: Partial<API.SeaHouseBillInput>;
};

export function buildHouseBillInput(
  values?: Partial<API.SeaHouseBillInput>,
): API.SeaHouseBillInput | undefined {
  if (!values) return undefined;
  return {
    id: values.id,
    houseNo: values.houseNo?.trim() ?? '',
    issuerSource:
      values.issuerSource ??
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_UNSPECIFIED,
    issuerPartnerId:
      values.issuerSource ===
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_OTHER_PARTNER
        ? values.issuerPartnerId
        : undefined,
    note: values.note?.trim() || undefined,
    content: values.content,
    expectedVersion: values.expectedVersion,
  };
}

export function isDocumentStructure(
  value: number | undefined,
): value is SeaDocumentStructure {
  return (
    value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE ||
    value === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
  );
}

export function ModeChangePreviewResult({
  preview,
}: {
  preview: API.SeaDocumentModeChangePreview;
}) {
  const differenceColumns: ColumnsType<API.SeaDocumentFieldDifference> = [
    { title: '字段', dataIndex: 'label', width: 140 },
    { title: '变更前', dataIndex: 'beforeValue' },
    { title: '变更后', dataIndex: 'afterValue' },
  ];
  const differenceSettings =
    useColumnSettings<ColumnsType<API.SeaDocumentFieldDifference>[number]>({
      tableKey: 'orders:doc-field-differences',
      columns: differenceColumns,
    });

  const impactColumns: ColumnsType<API.SeaDocumentDownstreamImpact> = [
    { title: '事实类型', dataIndex: 'factType', width: 130 },
    { title: '编号', dataIndex: 'referenceNo', width: 150 },
    { title: '影响', dataIndex: 'message' },
    {
      title: '结论',
      dataIndex: 'blocksExecution',
      width: 80,
      render: (blocked: boolean) => (
        <Tag color={blocked ? 'error' : 'success'}>
          {blocked ? '阻断' : '提示'}
        </Tag>
      ),
    },
  ];
  const impactSettings =
    useColumnSettings<ColumnsType<API.SeaDocumentDownstreamImpact>[number]>({
      tableKey: 'orders:doc-downstream-impacts',
      columns: impactColumns,
    });

  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Alert
        showIcon
        type={preview.executable ? 'success' : 'error'}
        title={preview.executable ? '预览通过，可以执行' : '当前变更不可执行'}
        description="离港时间、财务和放货事实只作为影响提示；执行不会自动改写这些下游事实。"
      />
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
        {differenceSettings.entry}
      </div>
      <Table<API.SeaDocumentFieldDifference>
        size="small"
        rowKey={(row) => `${row.field ?? ''}-${row.label ?? ''}`}
        pagination={false}
        dataSource={preview.differences ?? []}
        columns={differenceSettings.columns}
      />
      {differenceSettings.modal}
      {(preview.impacts?.length ?? 0) > 0 ? (
        <>
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
            {impactSettings.entry}
          </div>
          <Table<API.SeaDocumentDownstreamImpact>
            size="small"
            rowKey={(row) =>
              `${row.factType ?? ''}-${row.referenceId ?? ''}-${row.referenceNo ?? ''}`
            }
            pagination={false}
            dataSource={preview.impacts ?? []}
            columns={impactSettings.columns}
          />
          {impactSettings.modal}
        </>
      ) : null}
    </Space>
  );
}

export function SeaCreateDocumentModeField({
  disabled = false,
  onModeChange,
}: {
  disabled?: boolean;
  onModeChange?: (mode: SeaDocumentStructure) => void;
}) {
  const form = Form.useFormInstance();
  const initializedRef = useRef(false);
  const changeCreateMode = (nextMode: SeaDocumentStructure) => {
    onModeChange?.(nextMode);
    form.setFieldValue('seaDocumentStructure', nextMode);

    const goodsDescription = form.getFieldValue('goodsDescription');
    const totalPackages = form.getFieldValue('totalPackages');
    const totalPackageUnit = form.getFieldValue('totalPackageUnit');
    const totalGrossWeightKg = form.getFieldValue('totalGrossWeightKg');
    const totalVolumeCbm = form.getFieldValue('totalVolumeCbm');

    const cargoDefaults = {
      ...(goodsDescription ? { goodsDescriptionText: goodsDescription } : {}),
      ...(totalPackages !== undefined ? { packageCount: totalPackages } : {}),
      ...(totalPackageUnit ? { packageUnit: totalPackageUnit } : {}),
      ...(totalGrossWeightKg !== undefined
        ? { grossWeightKg: totalGrossWeightKg }
        : {}),
      ...(totalVolumeCbm !== undefined ? { volumeCbm: totalVolumeCbm } : {}),
    };

    const currentMbl = (form.getFieldValue('seaMasterBillContent') ??
      {}) as Record<string, unknown>;
    form.setFieldValue('seaMasterBillContent', {
      transportTerms: DEFAULT_TRANSPORT_TERMS,
      freightTerms: DEFAULT_FREIGHT_TERMS,
      ...cargoDefaults,
      ...currentMbl,
    });

    if (nextMode === SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE) {
      // 仅在尚未录入 HBL 时初始化默认内容；DIRECT 切回保留用户已填草稿。
      if (!form.getFieldValue('seaHouseBill')) {
        form.setFieldValue('seaHouseBill', {
          content: {
            transportTerms: DEFAULT_TRANSPORT_TERMS,
            freightTerms: DEFAULT_FREIGHT_TERMS,
            ...cargoDefaults,
          },
        });
      }
    }
    // DIRECT 下不清理 seaHouseBill：页签卸载使校验规则不注册，
    // 提交载荷由 form-adapter 按当前模式过滤，草稿值原样保留待切回。
  };

  // 全员分单制下新建默认 HOUSE：初始渲染不经过 onChange 联动，这里补一次
  // 分单内容默认值初始化（带入委托件重尺与条款默认），仅在分单内容为空时执行。
  // 模式默认值在 effect 中写入 store，而不是用 Form.Item initialValue：
  // 订单带入流程会经 Form initialValues 预置 seaDocumentStructure，字段级
  // initialValue 与之冲突会触发 rc-field-form 覆盖警告。
  useEffect(() => {
    if (initializedRef.current) return;
    initializedRef.current = true;
    if (
      form.getFieldValue('seaDocumentStructure') ===
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT
    ) {
      return;
    }
    if (form.getFieldValue('seaHouseBill')) {
      // 已带入 HBL 数据时内容默认值已就绪，只补缺失的模式标记，不覆盖带入内容。
      if (form.getFieldValue('seaDocumentStructure') === undefined) {
        form.setFieldValue(
          'seaDocumentStructure',
          SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
        );
      }
      return;
    }
    changeCreateMode(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
  }, []);

  return (
    <Form.Item
      name="seaDocumentStructure"
      label="提单模式"
      rules={[{ required: true, message: '请选择 HOUSE 或 DIRECT' }]}
      style={{ marginBottom: 0 }}
    >
      <Radio.Group
        disabled={disabled}
        onChange={(event) =>
          changeCreateMode(event.target.value as SeaDocumentStructure)
        }
      >
        <Radio.Button value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE}>
          有货代分单（HOUSE）
        </Radio.Button>
        <Radio.Button
          value={SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT}
        >
          仅船公司主单（DIRECT）
        </Radio.Button>
      </Radio.Group>
    </Form.Item>
  );
}
