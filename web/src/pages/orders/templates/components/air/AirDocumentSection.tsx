import { ProFormText, ProFormTextArea } from '@ant-design/pro-components';
import {
  Alert,
  Collapse,
  Descriptions,
  Form,
  Space,
  Tabs,
  Typography,
} from 'antd';
import { useEffect, useState } from 'react';
import { FormCell, FormRow, ProFormSearchableSelect } from '@/components/ui';
import { onSeaFormErrorReveal } from '../../sea-form-error-reveal';
import type { TemplateProps } from '../../types';
import { AirWaybillFields } from './AirWaybillFields';

/** 主单正文为轻量只读快照，不注册字段、不进入本票整单提交。 */
function MasterWaybillSnapshot({ master }: { master?: API.AirMasterWaybill }) {
  const content = master?.content?.waybillContent;
  const fields = [
    ['shipperText', '发货人名称与地址'],
    ['consigneeText', '收货人名称与地址'],
    ['notifyPartyText', '通知人'],
    ['secondNotifyPartyText', '第二通知人'],
    ['agentText', '代理名称与地址'],
    ['marks', '唛头'],
    ['goodsDescription', '运单品名'],
  ] as const;
  return (
    <Space
      orientation="vertical"
      size={12}
      style={{ width: '100%', alignItems: 'stretch' }}
    >
      <Descriptions
        size="small"
        layout="vertical"
        column={{ xs: 1, md: 2 }}
        items={fields.map(([key, label]) => ({
          key,
          label,
          children: (
            <span style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>
              {content?.[key] || '-'}
            </span>
          ),
        }))}
      />
      <Collapse
        size="small"
        items={[
          {
            key: 'additional',
            label: '其他主单资料',
            children: (
              <Descriptions
                size="small"
                column={{ xs: 1, md: 3 }}
                items={(
                  [
                    ['departureText', '起运机场印字'],
                    ['destinationText', '目的机场印字'],
                    ['flightText', '航班印字'],
                    ['currency', '运单币种'],
                    ['accountingInformation', '账务信息'],
                    ['handlingInformation', '处理信息'],
                    ['remarks', '单证备注'],
                  ] as const
                ).map(([key, label]) => ({
                  key,
                  label,
                  children: (
                    <span
                      style={{
                        whiteSpace: 'pre-wrap',
                        overflowWrap: 'anywhere',
                      }}
                    >
                      {content?.[key] || '-'}
                    </span>
                  ),
                }))}
              />
            ),
          },
        ]}
      />
    </Space>
  );
}

/** 主分单在同一业务卡内切换；隐藏页签常驻注册以保留输入和校验。 */
export function AirDocumentSection({ props }: { props: TemplateProps }) {
  const form = Form.useFormInstance();
  const watchedMode = Form.useWatch('airDocumentMode', {
    form,
    preserve: true,
  });
  const masterMode = Form.useWatch('airMasterMode', { form, preserve: true });
  const air = Form.useWatch('airExport', { form, preserve: true }) as
    | API.AirOrderInfo
    | undefined;
  const selectedMaster = Form.useWatch('airMasterSelection', {
    form,
    preserve: true,
  }) as API.AirMasterWaybill | undefined;
  const direct =
    (props.isDetail ? air?.documentMode : watchedMode) === 'DIRECT';
  const [activeTab, setActiveTab] = useState('HAWB');
  const master = props.isDetail ? air?.master : selectedMaster;
  const masterEditable = !props.isDetail && masterMode !== 'EXISTING';

  useEffect(
    () =>
      onSeaFormErrorReveal((fields) => {
        const first = fields[0]?.name[0];
        if (first === 'airMaster' && fields[0]?.name[1] === 'waybillContent')
          setActiveTab('MAWB');
        if (
          !direct &&
          (first === 'airHouseContent' ||
            first === 'airHouseNo' ||
            first === 'airHouseStatus' ||
            first === 'airHouseChangeReason')
        )
          setActiveTab('HAWB');
      }),
    [direct],
  );

  return (
    <>
      {direct && (
        <Alert
          type="info"
          showIcon
          title="直单：本票使用主运单，无 HAWB"
          style={{ width: '100%', marginBottom: 12 }}
        />
      )}
      <Tabs
        style={{ width: '100%' }}
        key={direct ? 'DIRECT' : 'HOUSE'}
        activeKey={direct ? 'MAWB' : activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'MAWB',
            label: '主运单 MAWB',
            forceRender: true,
            children: (
              <Space
                orientation="vertical"
                size={12}
                style={{ width: '100%', alignItems: 'stretch' }}
              >
                <Space wrap>
                  <Typography.Text strong>主运单正文</Typography.Text>
                  {props.airDocumentActions?.MAWB}
                </Space>
                {!masterEditable && (
                  <Typography.Paragraph type="secondary">
                    主单正文为共享资料，请通过主单制作入口维护；本票保存不修改主单正文。
                  </Typography.Paragraph>
                )}
                {masterEditable ? (
                  <AirWaybillFields
                    prefix={['airMaster', 'waybillContent']}
                    disabled={props.readonly}
                    variant="order"
                  />
                ) : (
                  <MasterWaybillSnapshot master={master} />
                )}
              </Space>
            ),
          },
          ...(!direct
            ? [
                {
                  key: 'HAWB',
                  label: '分运单 HAWB',
                  forceRender: true,
                  children: (
                    <Space
                      orientation="vertical"
                      size={12}
                      style={{ width: '100%', alignItems: 'stretch' }}
                    >
                      <Space wrap>
                        <Typography.Text strong>本票单证资料</Typography.Text>
                        {props.airDocumentActions?.HAWB}
                      </Space>
                      <FormRow cols={6} style={{ rowGap: 4 }}>
                        <FormCell span={2}>
                          <ProFormText
                            name="airHouseNo"
                            label="本票分运单号（HAWB）"
                            rules={[
                              {
                                required: true,
                                whitespace: true,
                                message: '请输入本票分运单号',
                              },
                            ]}
                            disabled={props.readonly || props.isDetail}
                            fieldProps={{ maxLength: 64 }}
                            tooltip="同一主单内不允许重号，不同主单可重复；号码会统一大写。"
                          />
                        </FormCell>
                        <ProFormSearchableSelect
                          name="airHouseStatus"
                          label="本票分单状态"
                          initialValue={props.isDetail ? undefined : 'DRAFT'}
                          options={[
                            { label: '资料待完善', value: 'DRAFT' },
                            { label: '已确认', value: 'CONFIRMED' },
                            { label: '已签发', value: 'RELEASED' },
                          ]}
                          disabled={props.readonly}
                        />
                      </FormRow>
                      <AirWaybillFields
                        prefix={['airHouseContent']}
                        disabled={props.readonly}
                        variant="order"
                      />
                      {props.isDetail && air?.house?.status === 'RELEASED' && (
                        <ProFormTextArea
                          name="airHouseChangeReason"
                          label="已签发分单修改原因"
                          tooltip="修改已签发分单内容或状态时必填；仅保存本票客户、货量等字段时可空。分单号保存后不在普通编辑中修改。"
                          fieldProps={{ maxLength: 500 }}
                          disabled={props.readonly}
                        />
                      )}
                    </Space>
                  ),
                },
              ]
            : []),
        ]}
      />
    </>
  );
}
