import { PlusOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Row,
  Select,
  Space,
  Typography,
} from 'antd';
import Decimal from 'decimal.js';
import { useEffect } from 'react';
import { useAccess } from '@/app/access';
import { settlementServiceListInvoiceProfilesForParty } from '@/services/roncin/settlementService';
import { formatAmount } from '@/utils/format';
import {
  applyBillSelection,
  applyTotalChange,
  autoAllocate,
  hasManualAllocation,
  newInvoiceRecord,
  recordGaps,
} from './autofill';
import InvoiceAllocationFields, {
  GapText,
  type RecordUpdater,
} from './InvoiceAllocationFields';
import type {
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
} from './types';

type Name = (string | number)[];
type Props = {
  name: Name;
  bills: InvoiceRegistrationBillSource[];
  organizationId: string;
  settlementPartyId: string;
  profiles?: API.FinanceInvoiceProfileOption[];
  allowAdd?: boolean;
};
const required = { required: true, message: '请明确填写本项' };
const amountRule = {
  validator: async (_: unknown, value: string) => {
    if (
      !/^(?:0|[1-9]\d{0,19})(?:\.\d{1,8})?$/.test(value ?? '') ||
      !new Decimal(value).gt(0)
    )
      throw new Error('请输入大于0的金额，最多8位小数');
  },
};
const listRule = {
  validator: async (_: unknown, value: unknown[]) => {
    if (!value?.length) throw new Error('至少添加一项明确分配');
  },
};

/** 发票层汇总：总额、已分配与待分配差额。 */
function RecordSummary({ record }: { record?: InvoiceRegistrationDraft }) {
  const gaps = recordGaps(record);
  return (
    <Space size="large" wrap style={{ marginBottom: 8 }}>
      <Typography.Text>
        发票总额 {formatAmount(record?.totalAmount || undefined)}
      </Typography.Text>
      <Typography.Text>
        已分配 {formatAmount(gaps.allocated.toFixed())}
      </Typography.Text>
      <GapText label="待分配" gap={gaps.invoiceGap} />
    </Space>
  );
}

/** 已有票据多条登记编辑器：独立登记与建账组共用，提交由所属事务入口处理。 */
export default function InvoiceRegistrationEditor({
  name,
  bills,
  organizationId,
  settlementPartyId,
  profiles,
  allowAdd = true,
}: Props) {
  const form = Form.useFormInstance();
  const records = Form.useWatch(name, { form, preserve: true }) as
    | InvoiceRegistrationDraft[]
    | undefined;
  const access = useAccess();
  const { modal } = App.useApp();
  const update: RecordUpdater = (index, change) => {
    const all = (form.getFieldValue(name) ?? []) as InvoiceRegistrationDraft[];
    if (!all[index]) return;
    form.setFieldValue([...name, index], change(all[index], all));
  };
  const allocateByBalance = (index: number) => {
    const run = () =>
      update(index, (current, all) => autoAllocate(current, bills, all, index));
    const current = (form.getFieldValue(name) ?? [])[index];
    if (hasManualAllocation(current) || current?.allocations?.length)
      modal.confirm({
        title: '按可登记余额重新分配？',
        content: '将覆盖本张发票现有的账单分配、费用金额和税额（含手动修改）。',
        onOk: run,
      });
    else run();
  };
  const canQuery =
    access.canCreateFinanceInvoices &&
    access.canOperateOrganization(organizationId);
  const query = useQuery({
    queryKey: [
      'finance',
      'invoice-registration-profiles',
      organizationId,
      settlementPartyId,
    ],
    enabled:
      canQuery &&
      (profiles === undefined ||
        Boolean(records?.some((record) => !record.sourceInvoiceId))) &&
      Boolean(settlementPartyId),
    meta: { errorMessage: '加载登记抬头失败' },
    queryFn: async ({ signal }) => {
      const response = await settlementServiceListInvoiceProfilesForParty(
        { settlementPartyId },
        { signal, skipErrorHandler: true },
      );
      if (
        !response?.data ||
        response.success === false ||
        response.data.organizationId !== organizationId ||
        response.data.settlementPartyId !== settlementPartyId
      )
        throw new Error('登记抬头来源不正确');
      return response.data.data ?? [];
    },
  });
  // 结算单位仅一个有效抬头时自动选中新增发票的抬头；多个抬头仍由财务明确选择。
  const onlyProfile = query.data?.length === 1 ? query.data[0].id : undefined;
  const recordCount = records?.length ?? 0;
  useEffect(() => {
    if (!onlyProfile || !recordCount) return;
    const all = (form.getFieldValue(name) ?? []) as InvoiceRegistrationDraft[];
    all.forEach((record, index) => {
      if (record && !record.sourceInvoiceId && !record.invoiceProfileId)
        form.setFieldValue([...name, index, 'invoiceProfileId'], onlyProfile);
    });
  }, [onlyProfile, recordCount, form, name]);
  const profileOptions = (items: API.FinanceInvoiceProfileOption[]) =>
    items.map((profile) => ({
      value: profile.id,
      label: `${profile.invoiceTitle ?? ''} · ${profile.taxpayerIdentificationNo ?? ''}`,
    }));
  return (
    <>
      <Alert
        type="info"
        showIcon
        title="登记已有发票。发票总额默认带出所选账单未登记余额并自动分配，请按实际发票修改；单账单单费用时改总额会同步分配。税额按来源税率给出建议值，请以实际发票为准。手动修改过的金额和税额不会被自动覆盖，分配不平时显示差额。"
        style={{ marginBottom: 12 }}
      />
      <Form.List
        name={name}
        rules={[
          {
            validator: async (_, rows: InvoiceRegistrationDraft[]) => {
              if ((rows?.length ?? 0) > 100)
                throw new Error('一次最多登记100张发票');
            },
          },
        ]}
      >
        {(fields, { add, remove }, errors) => (
          <>
            {fields.map((field) => (
              <Card
                key={field.key}
                size="small"
                title={`第${field.name + 1}张发票`}
                style={{ marginBottom: 12 }}
                extra={
                  <Space>
                    <Button
                      size="small"
                      onClick={() => allocateByBalance(field.name)}
                    >
                      按可登记余额分配
                    </Button>
                    <Button
                      type="text"
                      danger
                      onClick={() => remove(field.name)}
                    >
                      移除发票记录
                    </Button>
                  </Space>
                }
              >
                <Form.Item name={[field.name, 'sourceInvoiceId']} hidden>
                  <Input />
                </Form.Item>
                <Row gutter={12}>
                  <Col xs={24} md={12}>
                    <Form.Item
                      name={[field.name, 'invoiceProfileId']}
                      label="开票抬头"
                      rules={[required]}
                    >
                      <Select
                        options={profileOptions(
                          records?.[field.name]?.sourceInvoiceId
                            ? (profiles ?? [])
                            : (query.data ?? []),
                        )}
                        loading={query.isFetching}
                        disabled={Boolean(
                          records?.[field.name]?.sourceInvoiceId,
                        )}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item
                      name={[field.name, 'invoiceType']}
                      label="发票类型"
                      rules={[required]}
                    >
                      <Select
                        options={[
                          { value: 'NORMAL', label: '普通发票' },
                          { value: 'SPECIAL', label: '专用发票' },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'taxInvoiceNo']}
                      label="已有发票号码"
                      rules={[required, { max: 100, whitespace: true }]}
                    >
                      <Input />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'invoiceDate']}
                      label="发票日期"
                      rules={[required]}
                    >
                      <DatePicker style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item
                      name={[field.name, 'totalAmount']}
                      label="发票含税总额"
                      rules={[required, amountRule]}
                      extra={
                        records?.[field.name]?.totalSource === 'auto' &&
                        records?.[field.name]?.totalAmount
                          ? '默认为所选账单未登记余额，请按实际发票金额修改'
                          : undefined
                      }
                    >
                      <Input
                        onChange={() =>
                          update(field.name, (current) =>
                            applyTotalChange(
                              { ...current, totalSource: 'manual' },
                              bills,
                            ),
                          )
                        }
                      />
                    </Form.Item>
                  </Col>
                </Row>
                <RecordSummary record={records?.[field.name]} />
                <Form.List
                  name={[field.name, 'allocations']}
                  rules={[listRule]}
                >
                  {(allocations, actions, allocationErrors) => (
                    <>
                      {allocations.map((allocation) => (
                        <InvoiceAllocationFields
                          key={allocation.key}
                          name={allocation.name}
                          listName={name}
                          recordIndex={field.name}
                          bills={bills}
                          update={update}
                          remove={() => actions.remove(allocation.name)}
                        />
                      ))}
                      <Form.ErrorList errors={allocationErrors.errors} />
                      <Button
                        size="small"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          actions.add({
                            billId:
                              bills.length === 1 ? bills[0].id : undefined,
                            amountSource: 'auto',
                            fees: [],
                          });
                          if (bills.length === 1)
                            update(field.name, (current, all) =>
                              applyBillSelection(
                                current,
                                current.allocations.length - 1,
                                bills,
                                all,
                                field.name,
                              ),
                            );
                        }}
                      >
                        添加账单分配
                      </Button>
                    </>
                  )}
                </Form.List>
                <Form.Item
                  name={[field.name, 'note']}
                  label="备注"
                  rules={[{ max: 500 }]}
                  style={{ marginTop: 12 }}
                >
                  <Input />
                </Form.Item>
              </Card>
            ))}
            <Form.ErrorList errors={errors.errors} />
            {allowAdd && (
              <Button
                icon={<PlusOutlined />}
                onClick={() => add(newInvoiceRecord(bills, records ?? []))}
              >
                继续添加发票
              </Button>
            )}
          </>
        )}
      </Form.List>
      <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
        来源税率仅用于给出建议税额；登记税额以财务确认的发票实际值为准，不改写原费用。任一登记失败，当前提交全部回滚并保留输入。
      </Typography.Paragraph>
    </>
  );
}
