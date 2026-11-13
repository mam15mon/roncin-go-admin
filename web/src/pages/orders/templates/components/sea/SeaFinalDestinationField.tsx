import { ProFormText } from '@ant-design/pro-components';
import { Button, Form } from 'antd';
import { useEffect, useId } from 'react';
import { ProFormSearchableSelect } from '@/components/ui';
import type { CreateOrderFormValues } from '../../../order-kinds/form-values';
import type { TemplateProps } from '../../types';
import { useSharedSeaVoyage } from './useSeaMasterBillCandidate';

const DESTINATION_HINT =
  '本票独立维护，不随共同航程同步：默认同卸货港，有后续转运时再修改。';

function DestinationModeButton({
  target,
  label,
  disabled,
  onChange,
}: {
  target: 'manual' | 'discharge';
  label?: string;
  disabled: boolean;
  value?: CreateOrderFormValues['seaDestinationDefault'];
  onChange?: (value: CreateOrderFormValues['seaDestinationDefault']) => void;
}) {
  const form = Form.useFormInstance();
  return (
    <Button
      type="link"
      size="small"
      disabled={disabled}
      onClick={() => {
        const value =
          form.getFieldValue(
            target === 'manual'
              ? 'destinationLocationId'
              : 'dischargeLocationId',
          ) ?? null;
        if (target === 'discharge')
          form.setFieldValue('destinationLocationId', value);
        // 走真实表单 change 协议，单独更改默认意图也要标脏并写入草稿。
        onChange?.({ mode: target, value, label });
      }}
    >
      {target === 'manual' ? '更改最终目的港' : '恢复与卸货港一致'}
    </Button>
  );
}

/** 默认只属于新建意图；人工选择、清空和客户明确预填均与共同段解绑。 */
export function SeaFinalDestinationField({
  locationOptions,
  searchLocations,
  isDetail = false,
  readonly = false,
}: Pick<
  TemplateProps,
  'locationOptions' | 'searchLocations' | 'isDetail' | 'readonly'
>) {
  const form = Form.useFormInstance();
  const outputId = useId();
  const { voyage } = useSharedSeaVoyage();
  const values = Form.useWatch(
    (store) => ({
      destination: store.destinationLocationId as string | null | undefined,
      discharge: store.dischargeLocationId as string | null | undefined,
      state:
        store.seaDestinationDefault as CreateOrderFormValues['seaDestinationDefault'],
      option:
        store.seaDischargePortOption as CreateOrderFormValues['seaDischargePortOption'],
    }),
    { form, preserve: true },
  );
  const discharge = values?.discharge;
  const destination = values?.destination;
  const state = values?.state;
  const label =
    locationOptions.find((option) => option.value === discharge)?.label ??
    (voyage?.dischargeLocationId === discharge
      ? voyage?.dischargeLocationName
      : undefined) ??
    (values?.option?.value === discharge ? values?.option?.label : undefined) ??
    (state?.value === discharge ? state?.label : undefined);

  useEffect(() => {
    if (isDetail) return;
    // 模板在 layout effect 恢复草稿；此处读完整 store，不能把主动清空的 null 当未填。
    const current = form.getFieldsValue(true) as CreateOrderFormValues;
    const existing = current.seaDestinationDefault;
    const currentDestination = current.destinationLocationId;
    if (!existing && currentDestination !== undefined) {
      form.setFieldValue('seaDestinationDefault', { mode: 'manual' });
      return;
    }
    if (existing?.mode === 'manual') return;
    if (existing && currentDestination !== existing.value) {
      // 客户预设等明确带入独立目的港，后续共同段重读不应覆盖它。
      form.setFieldValue('seaDestinationDefault', { mode: 'manual' });
      return;
    }
    const target = current.dischargeLocationId ?? null;
    const next = { mode: 'discharge' as const, value: target, label };
    if (
      JSON.stringify(existing) === JSON.stringify(next) &&
      currentDestination === target
    )
      return;
    form.setFieldsValue({
      destinationLocationId: target,
      seaDestinationDefault: next,
    });
  }, [form, isDetail, discharge, destination, state, label]);

  const defaultMode = !isDetail && state?.mode !== 'manual';
  const options = [...locationOptions];
  if (destination && !options.some((option) => option.value === destination)) {
    const destinationLabel = destination === discharge ? label : state?.label;
    if (destinationLabel)
      options.push({ value: destination, label: destinationLabel });
  }
  return (
    <div>
      {defaultMode ? (
        <>
          <ProFormText name="destinationLocationId" hidden />
          <Form.Item
            label="最终目的港"
            tooltip={DESTINATION_HINT}
            htmlFor={outputId}
          >
            {/* 与同行下拉框等高的只读框，链接按钮作为后缀，避免额外换行。 */}
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                height: 32,
                boxSizing: 'border-box',
                padding: '0 4px 0 11px',
                border: '1px solid #c3cbd6',
                borderRadius: 6,
                background: '#ffffff',
              }}
            >
              <output
                id={outputId}
                style={{
                  flex: 1,
                  minWidth: 0,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  lineHeight: '30px',
                  color: destination ? undefined : '#5b6b82',
                }}
              >
                {destination
                  ? `${label || '当前卸货港'}（同卸货港）`
                  : '选择共同航程卸货港后自动带入'}
              </output>
              <Form.Item name="seaDestinationDefault" noStyle>
                <DestinationModeButton
                  target="manual"
                  label={label}
                  disabled={readonly}
                />
              </Form.Item>
            </div>
          </Form.Item>
        </>
      ) : (
        <ProFormSearchableSelect
          name="destinationLocationId"
          label="最终目的港"
          tooltip={DESTINATION_HINT}
          options={options}
          disabled={readonly || undefined}
          request={
            readonly
              ? undefined
              : async ({ keyWords }) => searchLocations(keyWords)
          }
          fieldProps={{
            filterOption: false,
            onChange: (value, option) => {
              if (!isDetail) {
                const selected = Array.isArray(option) ? option[0] : option;
                form.setFieldValue('seaDestinationDefault', {
                  mode: 'manual',
                  value: value ?? null,
                  label: selected?.label ? String(selected.label) : undefined,
                });
              }
            },
          }}
          placeholder="请选择本票最终目的港"
        />
      )}
      {/* 说明收进标签提示；仅新建手动改港时保留恢复入口，常规同港不占额外行。 */}
      {!isDetail && !defaultMode && (
        <div style={{ marginTop: -8 }}>
          <Form.Item name="seaDestinationDefault" noStyle>
            <DestinationModeButton
              target="discharge"
              label={label}
              disabled={readonly}
            />
          </Form.Item>
        </div>
      )}
    </div>
  );
}
