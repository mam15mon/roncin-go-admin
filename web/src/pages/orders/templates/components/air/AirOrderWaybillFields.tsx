import { DownloadOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { App, Button, Form, Input, Segmented, Tag } from 'antd';
import { type ReactNode, useEffect, useId, useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import { FormRow } from '@/components/ui';
import { partnerServiceGetPartner } from '@/services/roncin/partnerService';
import { onSeaFormErrorReveal } from '../../sea-form-error-reveal';

type Path = (string | number)[];
type CommonField =
  | 'shipperText'
  | 'consigneeText'
  | 'notifyPartyText'
  | 'secondNotifyPartyText'
  | 'agentText'
  | 'marks'
  | 'goodsDescription';
export type AirDocumentImport = (
  prefix: Path,
  field: 'shipperText' | 'agentText' | 'goodsDescription',
  value: string,
) => void;

const headings: Record<CommonField, string> = {
  shipperText: '发货人 (Shipper)',
  consigneeText: '收货人 (Consignee)',
  notifyPartyText: '通知人 (Notify Party)',
  secondNotifyPartyText: '第二通知人 (Second Notify Party)',
  agentText: '代理 (Agent)',
  marks: '唛头 (Marks & Numbers)',
  goodsDescription: '英文品名 / 运单货描',
};
const labels: Record<CommonField, string> = {
  shipperText: '发货人名称与地址',
  consigneeText: '收货人名称与地址',
  notifyPartyText: '通知人',
  secondNotifyPartyText: '第二通知人',
  agentText: '代理名称与地址',
  marks: '唛头',
  goodsDescription: '运单品名',
};
// 与海运提单录入同口径的占位提示；空运单不可转让，收货人须具名，不提供 TO ORDER 快捷。
const placeholders: Record<CommonField, string> = {
  shipperText: '请输入发货人英文名称与详细地址',
  consigneeText: '请输入收货人名称、详细地址与联系方式',
  notifyPartyText: '请输入通知人名称与详细地址 (例如：SAME AS CONSIGNEE)',
  secondNotifyPartyText: '请输入第二通知人名称与详细地址 (选填)',
  agentText: '请输入代理名称、地址与联系方式',
  marks: '请输入唛头信息 (例如：N/M)',
  goodsDescription: '请输入运单打印品名与货物描述',
};
const headingStyle = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  flexWrap: 'wrap',
  gap: 6,
  marginBottom: 6,
  minHeight: 24,
  fontSize: 13,
  fontWeight: 500,
} as const;

/** 编辑和只读共用字段编排；只读渲染原生受控文本域，不注册父表单字段。 */
export function AirOrderWaybillLayout({
  content,
  prefix,
  renderField,
  actions,
  onSameAsConsignee,
}: {
  content?: API.AirWaybillContent;
  prefix?: Path;
  renderField?: (field: CommonField, id: string, rows: number) => ReactNode;
  actions?: Partial<
    Record<'shipperText' | 'agentText' | 'goodsDescription', ReactNode>
  >;
  /** 可编辑时提供，把通知人填为 SAME AS CONSIGNEE。 */
  onSameAsConsignee?: () => void;
}) {
  const uniqueId = useId();
  const [notify, setNotify] = useState<
    'notifyPartyText' | 'secondNotifyPartyText'
  >('notifyPartyText');
  const prefixKey = JSON.stringify(prefix);
  const fieldId = (name: CommonField) =>
    prefix ? [...prefix, name].join('_') : `${uniqueId}-${name}`;
  useEffect(
    () =>
      onSeaFormErrorReveal((fields) => {
        if (!prefixKey) return;
        const path = JSON.parse(prefixKey) as Path;
        const name = fields[0]?.name;
        if (!name || !path.every((item, index) => item === name[index])) return;
        const field = name[path.length];
        if (field === 'notifyPartyText' || field === 'secondNotifyPartyText')
          setNotify(field);
      }),
    [prefixKey],
  );
  const field = (name: CommonField, rows = 4) => {
    const id = fieldId(name);
    return renderField ? (
      renderField(name, id, rows)
    ) : (
      <Input.TextArea
        id={id}
        aria-label={labels[name]}
        readOnly
        disabled={false}
        value={content?.[name] ?? ''}
        rows={rows}
        style={{ height: 'auto' }}
      />
    );
  };
  const block = (name: CommonField, rows = 4) => (
    <div style={{ minWidth: 0 }}>
      <div style={headingStyle}>
        <label htmlFor={fieldId(name)}>{headings[name]}</label>
        {name !== 'goodsDescription' && actions?.[name as keyof typeof actions]}
      </div>
      {field(name, rows)}
    </div>
  );
  return (
    <div style={{ width: '100%' }}>
      <FormRow cols={2}>
        {block('shipperText')}
        {block('consigneeText')}
        <div style={{ minWidth: 0 }}>
          <div style={headingStyle}>
            <Segmented
              aria-label="通知人切换"
              size="small"
              value={notify}
              disabled={false}
              onChange={(value) => setNotify(value as typeof notify)}
              options={[
                { value: 'notifyPartyText', label: '通知人 (Notify Party)' },
                {
                  value: 'secondNotifyPartyText',
                  label: content?.secondNotifyPartyText
                    ? '第二通知人 (Second Notify Party) · 已填写'
                    : '第二通知人 (Second Notify Party)',
                },
              ]}
            />
            {onSameAsConsignee && notify === 'notifyPartyText' ? (
              <Tag
                style={{
                  cursor: 'pointer',
                  userSelect: 'none',
                  margin: 0,
                  fontSize: 11,
                  padding: '0 4px',
                }}
                onClick={onSameAsConsignee}
              >
                + SAME AS CONSIGNEE
              </Tag>
            ) : null}
          </div>
          <div hidden={notify !== 'notifyPartyText'}>
            {field('notifyPartyText')}
          </div>
          <div hidden={notify !== 'secondNotifyPartyText'}>
            {field('secondNotifyPartyText')}
          </div>
        </div>
        {block('agentText')}
      </FormRow>
      <div style={{ ...headingStyle, margin: '20px 0 12px', fontWeight: 600 }}>
        <span>运单货物描述 (AWB Marks & Cargo)</span>
        {actions?.goodsDescription}
      </div>
      <FormRow cols={2}>
        {block('marks', 5)}
        {block('goodsDescription', 5)}
      </FormRow>
    </div>
  );
}

function partnerText(partner: API.Partner): string {
  const contact = partner.contacts?.[0];
  const contactText = contact
    ? [contact.name, contact.phone, contact.email].filter(Boolean).join(' ')
    : '';
  return [
    partner.profile?.nameEn || partner.legalName,
    partner.profile?.addressEn ||
      partner.registeredAddress ||
      partner.profile?.addressDetail,
    contactText ? `TEL/CONTACT: ${contactText}` : undefined,
  ]
    .filter(Boolean)
    .join('\n');
}

/** 用户显式带入仅在原公司/订单/正文和目标输入均未变化时采纳。 */
export function AirOrderWaybillFields({
  prefix,
  disabled,
  onImport,
}: {
  prefix: Path;
  disabled?: boolean;
  onImport?: AirDocumentImport;
}) {
  const form = Form.useFormInstance();
  const { message, modal } = App.useApp();
  const { initialState } = useInitialState();
  const organizationId = initialState?.currentUser?.currentOrganization?.id;
  const client = useQueryClient();
  const content = Form.useWatch(prefix, { form, preserve: true }) as
    | API.AirWaybillContent
    | undefined;
  const watched = Form.useWatch(
    (values) => [
      values.id,
      values.businessType,
      values.customerId,
      values.foreignAgentId,
      values.goodsDescription,
      values.airDocumentMode,
      values.airMasterMode,
      values.airMasterId,
      values.airMasterSelection?.id,
      values.airExport?.master?.id,
    ],
    { form, preserve: true },
  );
  const fingerprint = JSON.stringify([
    organizationId,
    prefix,
    watched,
    content,
    disabled,
  ]);
  const state = useRef({ fingerprint, revision: 0, active: true });
  if (state.current.fingerprint !== fingerprint) {
    state.current.fingerprint = fingerprint;
    state.current.revision += 1;
  }
  useEffect(() => {
    state.current.active = true;
    return () => {
      state.current.active = false;
    };
  }, []);
  const actionCounter = useRef(0);
  const [loading, setLoading] = useState<string>();
  const callback = useRef(onImport);
  callback.current = onImport;

  const bring = async (
    field: 'shipperText' | 'agentText' | 'goodsDescription',
  ) => {
    if (disabled || !callback.current) return;
    const revision = state.current.revision;
    const action = ++actionCounter.current;
    const currentText = form.getFieldValue([...prefix, field]);
    const sourceKey =
      field === 'shipperText'
        ? 'customerId'
        : field === 'agentText'
          ? 'foreignAgentId'
          : 'goodsDescription';
    const source = form.getFieldValue(sourceKey) as string | undefined;
    if (!source || (field !== 'goodsDescription' && !organizationId)) {
      message.warning(
        field === 'goodsDescription'
          ? '请先填写订单货物描述'
          : field === 'shipperText'
            ? '请先选择委托客户'
            : '请先选择国外代理',
      );
      return;
    }
    const stillCurrent = () =>
      state.current.active &&
      action === actionCounter.current &&
      revision === state.current.revision &&
      form.getFieldValue([...prefix, field]) === currentText &&
      form.getFieldValue(sourceKey) === source;
    setLoading(field);
    try {
      let text = source;
      if (field !== 'goodsDescription') {
        const response = await client.fetchQuery({
          queryKey: [
            'air-document-import',
            { organizationId, partnerId: source },
          ],
          queryFn: ({ signal }) =>
            partnerServiceGetPartner(
              { id: source },
              { signal, skipErrorHandler: true },
            ),
          staleTime: 0,
          meta: { silent: true },
        });
        if (!stillCurrent()) return;
        if (!response || response.success === false || !response.data) {
          message.error('获取客商资料失败，请重试');
          return;
        }
        if (
          response.data.id !== source ||
          response.data.organizationId !== organizationId
        ) {
          message.error('客商资料身份不匹配，请重新选择');
          return;
        }
        text = partnerText(response.data);
      }
      if (!stillCurrent()) return;
      if (!text) {
        message.warning('所选资料没有可带入的名称或地址');
        return;
      }
      if (currentText && currentText !== text) {
        const accepted = await modal.confirm({
          title: '覆盖已有内容？',
          content: '当前文本已有内容，带入将替换该字段。',
          okText: '确认覆盖',
          cancelText: '保留原内容',
        });
        if (!accepted || !stillCurrent()) return;
      }
      if (!callback.current) return;
      callback.current(prefix, field, text);
      message.success('已带入，请保存当前订单');
    } catch (error) {
      if (stillCurrent())
        message.error(
          error instanceof Error ? error.message : '带入资料失败，请重试',
        );
    } finally {
      if (state.current.active && action === actionCounter.current)
        setLoading(undefined);
    }
  };
  const action = (
    field: 'shipperText' | 'agentText' | 'goodsDescription',
    text: string,
  ) =>
    !disabled &&
    Boolean(onImport) && (
      <Button
        type="link"
        size="small"
        icon={<DownloadOutlined />}
        loading={loading === field}
        onClick={() => void bring(field)}
        style={{ padding: 0, height: 'auto', fontSize: 12 }}
      >
        {text}
      </Button>
    );
  return (
    <AirOrderWaybillLayout
      onSameAsConsignee={
        disabled
          ? undefined
          : () =>
              form.setFieldValue(
                [...prefix, 'notifyPartyText'],
                'SAME AS CONSIGNEE',
              )
      }
      content={content}
      prefix={prefix}
      actions={{
        shipperText: action('shipperText', '从委托客户带入'),
        agentText: action('agentText', '从国外代理带入'),
        goodsDescription: action('goodsDescription', '从订单货物信息带入'),
      }}
      renderField={(field, id, rows) => (
        <Form.Item
          name={[...prefix, field]}
          style={{ marginBottom: 0 }}
          rules={[
            {
              validator: async (_, value) => {
                const max =
                  field === 'marks' || field === 'goodsDescription'
                    ? 10000
                    : 5000;
                if (typeof value === 'string' && Array.from(value).length > max)
                  throw new Error(`${labels[field]}最多 ${max} 字`);
              },
            },
          ]}
        >
          <Input.TextArea
            id={id}
            aria-label={labels[field]}
            placeholder={placeholders[field]}
            disabled={disabled}
            rows={rows}
            style={{ height: 'auto' }}
            maxLength={
              field === 'marks' || field === 'goodsDescription' ? 10000 : 5000
            }
          />
        </Form.Item>
      )}
    />
  );
}
