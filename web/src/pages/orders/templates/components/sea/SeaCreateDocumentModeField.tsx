import {
  DownloadOutlined,
  SaveOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import {
  ProFormDigit,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Form,
  Modal,
  Radio,
  Row,
  Segmented,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { createStyles } from 'antd-style';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useAccess } from '@/app/access';
import {
  FormRow,
  PackageCountInput,
  ProFormSearchableSelect,
} from '@/components/ui';

import {
  OrderBusinessType,
  OrderReleasePodStatus,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import { searchPartnerOptions } from '@/features/partners';
import { orderReleasePodServiceListReleasePods } from '@/services/roncin/orderReleasePodService';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import {
  seaDocumentServiceExecuteChangeSeaDocumentMode,
  seaDocumentServiceGetSeaOrderDocuments,
  seaDocumentServicePreviewChangeSeaDocumentMode,
  seaDocumentServiceUpdateSeaHouseBill,
  seaDocumentServiceUpdateSeaMasterBillContent,
} from '@/services/roncin/seaDocumentService';
import { generateUUID } from '@/utils/uuid';
import { RELEASE_PODS_CHANGED_EVENT } from '../../../release-pod-events';
import type { TemplateProps, TemplateSection } from '../../types';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';
import SeaExternalConfirmationFields, {
  buildSeaExternalConfirmation,
  type SeaExternalConfirmationFormValues,
} from './SeaExternalConfirmationFields';

const { Text } = Typography;

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
  return (
    <Space orientation="vertical" size={12} style={{ width: '100%' }}>
      <Alert
        showIcon
        type={preview.executable ? 'success' : 'error'}
        title={preview.executable ? '预览通过，可以执行' : '当前变更不可执行'}
        description="离港时间、财务和放货事实只作为影响提示；执行不会自动改写这些下游事实。"
      />
      <Table<API.SeaDocumentFieldDifference>
        size="small"
        rowKey={(row) => `${row.field ?? ''}-${row.label ?? ''}`}
        pagination={false}
        dataSource={preview.differences ?? []}
        columns={[
          { title: '字段', dataIndex: 'label', width: 140 },
          { title: '变更前', dataIndex: 'beforeValue' },
          { title: '变更后', dataIndex: 'afterValue' },
        ]}
      />
      {(preview.impacts?.length ?? 0) > 0 ? (
        <Table<API.SeaDocumentDownstreamImpact>
          size="small"
          rowKey={(row) =>
            `${row.factType ?? ''}-${row.referenceId ?? ''}-${row.referenceNo ?? ''}`
          }
          pagination={false}
          dataSource={preview.impacts ?? []}
          columns={[
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
          ]}
        />
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
      form.setFieldValue('seaHouseBill', {
        content: {
          transportTerms: DEFAULT_TRANSPORT_TERMS,
          freightTerms: DEFAULT_FREIGHT_TERMS,
          ...cargoDefaults,
        },
      });
    } else {
      form.setFieldValue('seaHouseBill', undefined);
    }
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
