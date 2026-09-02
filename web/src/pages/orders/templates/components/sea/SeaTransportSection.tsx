import {
  ProFormDateTimePicker,
  ProFormText,
} from '@ant-design/pro-components';
import { Alert, Button, Col, Form, Input, Tooltip } from 'antd';
import dayjs from 'dayjs';
import isoWeek from 'dayjs/plugin/isoWeek';
import React from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import { containerOwnershipOptions } from '../../../common';
import {
  OrderContainerRequestFields,
  OrderShippingDocumentFields,
  type SelectOption,
} from '../../../order-plan-fields';
import { resolveSeaOrderFormPolicy } from '../../../sea-order-policy';

dayjs.extend(isoWeek);

export function SeaContainerPlanFields({
  options,
}: {
  options: SelectOption[];
}) {
  const form = Form.useFormInstance();
  const shipmentType = Form.useWatch('shipmentType');
  const containerRequests = (Form.useWatch('containerRequests') ??
    []) as API.OrderContainerRequestInput[];
  const policy = resolveSeaOrderFormPolicy({ shipmentType });

  if (!policy.showContainerPlan) {
    return (
      <Col span={24}>
        <Alert
          type={containerRequests.length > 0 ? 'warning' : 'info'}
          showIcon
          title="散杂货不使用箱型箱量、箱号或封号配置"
          description={
            containerRequests.length > 0
              ? '切换托运类型前已经录入箱型箱量，请确认后清空，系统不会静默删除已有数据。'
              : '页面已隐藏集装箱专属配置，货物将按件数、毛重、体积和计费吨管理。'
          }
          action={
            containerRequests.length > 0 ? (
              <Button
                danger
                size="small"
                htmlType="button"
                onClick={() => form.setFieldValue('containerRequests', [])}
              >
                清空箱量计划
              </Button>
            ) : undefined
          }
          style={{ marginBottom: 16 }}
        />
      </Col>
    );
  }
  return <OrderContainerRequestFields options={options} />;
}

export function SeaScheduleDateFields() {
  const form = Form.useFormInstance();
  const etd = Form.useWatch('etd', form);
  const weekValue =
    etd && dayjs(etd).isValid() ? `W${dayjs(etd).isoWeek()}` : '';

  return (
    <>
      <Col className="col-5">
        <ProFormDateTimePicker
          name="etd"
          label="ETD (预计开航)"
          fieldProps={{
            style: { width: '100%' },
            onChange: (date) => {
              if (date) {
                const currentEta = form?.getFieldValue('eta');
                if (currentEta?.isBefore(date)) {
                  form?.setFieldValue('eta', undefined);
                }
              }
            },
          }}
        />
      </Col>
      <Col className="col-5">
        <Form.Item label="WEEK" style={{ marginInline: 0 }}>
          <Input
            value={weekValue}
            placeholder="依据 ETD 自动生成"
            disabled
            style={{ width: '100%' }}
          />
        </Form.Item>
      </Col>
      <Col className="col-5">
        <ProFormDateTimePicker
          name="eta"
          label="ETA (预计到达)"
          dependencies={['etd']}
          rules={[
            ({ getFieldValue }) => ({
              validator(_, value) {
                const etdVal = getFieldValue('etd');
                if (
                  !value ||
                  !etdVal ||
                  value.isAfter(etdVal) ||
                  value.isSame(etdVal)
                ) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('ETA 不能早于 ETD'));
              },
            }),
          ]}
          fieldProps={{ style: { width: '100%' } }}
        />
      </Col>
    </>
  );
}

export function buildSeaTransportSection(
  locationOptions: SelectOption[],
  searchLocations: (keyword?: string) => Promise<SelectOption[]>,
  containerSpecOptions: SelectOption[],
) {
  return {
    key: 'transportInfo',
    title: '配舱信息',
    content: (
      <>
        {/* 第 1 行：主单号、分单号、主单单证类型、主单签放方式、分单签放方式 */}
        <OrderShippingDocumentFields transportMode="sea" />

        {/* 第 2 行：箱型箱量 */}
        <SeaContainerPlanFields options={containerSpecOptions} />

        {/* 第 3 行：航线 4 港口（起运港、目的港、卸货港、中转港） */}
        <Col className="col-5">
          <ProFormSearchableSelect
            name="originLocationId"
            label="起运港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择起运港或地点"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="destinationLocationId"
            label="目的港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择目的港或地点"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="dischargeLocationId"
            label="卸货港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择卸货港"
          />
        </Col>
        <Col className="col-5">
          <ProFormSearchableSelect
            name="transitLocationId"
            label="中转港"
            options={locationOptions}
            request={async ({ keyWords }) => searchLocations(keyWords)}
            fieldProps={{ filterOption: false }}
            placeholder="请选择中转港"
          />
        </Col>
        <Col className="col-5" />

        {/* 第 4 行：货主箱标记、船名航次、ETD、WEEK、ETA */}
        <Col className="col-5">
          <ProFormSearchableSelect
            name="containerOwnership"
            label="货主箱标记"
            options={containerOwnershipOptions}
            placeholder="请选择 COC / SOC"
          />
        </Col>
        <Col className="col-5">
          <ProFormText
            name="vesselVoyage"
            label="船名航次"
            placeholder="请输入船名航次"
            fieldProps={{
              suffix: (
                <Tooltip title="船期与船舶实时动态追踪">
                  <a
                    href="https://www.shipxy.com/"
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{ fontSize: 12, color: '#1677ff' }}
                    onClick={(e) => e.stopPropagation()}
                  >
                    船在哪儿
                  </a>
                </Tooltip>
              ),
            }}
          />
        </Col>
        <SeaScheduleDateFields />

        {/* 第 5 行：SI截关时间、单证截关时间(截单时间)、报关截关时间(截关时间)、VGM截关时间 */}
        <Col className="col-5">
          <ProFormDateTimePicker
            name="siCutoff"
            label="SI截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="docCutoff"
            label="单证截关时间"
            tooltip="即截单时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="customsCutoff"
            label="报关截关时间"
            tooltip="即截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5">
          <ProFormDateTimePicker
            name="vgmCutoff"
            label="VGM截关时间"
            fieldProps={{ style: { width: '100%' } }}
          />
        </Col>
        <Col className="col-5" />
      </>
    ),
  };
}
