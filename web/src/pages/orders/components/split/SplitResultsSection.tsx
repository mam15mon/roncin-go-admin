import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Col,
  Input,
  Radio,
  Row,
  Select,
  Space,
  Tag,
  Typography,
} from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { DefaultOptionType } from 'antd/es/select';
import dayjs from 'dayjs';
import type { Dispatch, SetStateAction } from 'react';
import { SectionCard } from '@/components/ui';
import { searchShippingLineOptions } from '@/features/master-data/shipping-lines';
import { orderServiceMatchSeaMasterBillCandidate } from '@/services/roncin/orderService';
import { getErrorMessage, type ResultConfig } from '../../splitUtils';

const { Text } = Typography;

interface SplitResultsSectionProps {
  results: ResultConfig[];
  setResults: Dispatch<SetStateAction<ResultConfig[]>>;
  splitContext: API.SeaOrderSplitContextData | null;
  canReassign: boolean;
  carrierOptions: DefaultOptionType[];
  setCarrierOptions: Dispatch<SetStateAction<DefaultOptionType[]>>;
  message: MessageInstance;
  triggerPreview: (results: ResultConfig[]) => void;
  onAddResult: () => void;
  onRemoveResult: (key: string) => void;
}

/** 拆票页区块 2：拆票目标集规划（每票母单策略、候选匹配与新母单录入）。 */
export default function SplitResultsSection({
  results,
  setResults,
  splitContext,
  canReassign,
  carrierOptions,
  setCarrierOptions,
  message,
  triggerPreview,
  onAddResult,
  onRemoveResult,
}: SplitResultsSectionProps) {
  return (
    <SectionCard
      title={`拆票目标集规划（共 ${results.length} 票）`}
      extra={
        <Button
          type="dashed"
          size="small"
          icon={<PlusOutlined />}
          onClick={onAddResult}
        >
          添加拆出新票
        </Button>
      }
    >
      <Row gutter={[16, 16]}>
        {results.map((res, index) => (
          <Col span={24} key={res.key}>
            <Card
              size="small"
              style={{
                borderColor: res.role === 'ORIGINAL' ? '#d9d9d9' : '#91caff',
                background: res.role === 'ORIGINAL' ? '#fafafa' : '#f0f7ff',
              }}
              title={
                <Space>
                  <Tag color={res.role === 'ORIGINAL' ? 'default' : 'geekblue'}>
                    {res.role === 'ORIGINAL' ? '保留原票' : `新票 ${index}`}
                  </Tag>
                  <Text strong>{res.title}</Text>
                </Space>
              }
              extra={
                res.role === 'CREATED' && (
                  <Button
                    type="text"
                    danger
                    size="small"
                    icon={<DeleteOutlined />}
                    onClick={() => onRemoveResult(res.key)}
                  >
                    删除此新票
                  </Button>
                )
              }
            >
              <Row gutter={16}>
                <Col span={6}>
                  <div style={{ marginBottom: 4 }}>
                    <Text type="secondary">母单配载策略：</Text>
                  </div>
                  <Radio.Group
                    value={res.targetType}
                    onChange={(e) => {
                      const val = e.target.value;
                      const updated = [...results];
                      let newAllocNotes = res.allocationNotes;
                      if (val === 'CURRENT') {
                        newAllocNotes = splitContext?.allocationNotes || '';
                      } else if (res.targetType === 'CURRENT') {
                        newAllocNotes = '';
                      }
                      updated[index] = {
                        ...res,
                        targetType: val,
                        shippingLineId:
                          val === 'CURRENT'
                            ? undefined
                            : res.shippingLineId ||
                              splitContext?.currentMasterBill?.shippingLineId,
                        allocationNotes: newAllocNotes,
                        candidateId: undefined,
                        candidateVersion: undefined,
                        candidateTeId: undefined,
                        candidateTeVersion: undefined,
                      };
                      setResults(updated);
                    }}
                  >
                    <Radio value="CURRENT">沿用当前母单</Radio>
                    {canReassign && <Radio value="NEW">录入新母单</Radio>}
                    {canReassign && (
                      <Radio value="CANDIDATE">选择已有母单</Radio>
                    )}
                  </Radio.Group>
                </Col>
                <Col span={6}>
                  <div style={{ marginBottom: 4 }}>
                    <Text type="secondary">内部单号：</Text>
                  </div>
                  <Input
                    placeholder="内部单号（可留空）"
                    value={res.internalReferenceNo}
                    onChange={(e) => {
                      const updated = [...results];
                      updated[index] = {
                        ...res,
                        internalReferenceNo: e.target.value,
                      };
                      setResults(updated);
                    }}
                  />
                </Col>
                <Col span={6}>
                  <div style={{ marginBottom: 4 }}>
                    <Text type="secondary">订舱 / 排载备注：</Text>
                  </div>
                  <Input
                    placeholder="订舱备注"
                    value={res.bookingNotes}
                    onChange={(e) => {
                      const updated = [...results];
                      updated[index] = {
                        ...res,
                        bookingNotes: e.target.value,
                      };
                      setResults(updated);
                    }}
                  />
                </Col>
                <Col span={6}>
                  <div style={{ marginBottom: 4 }}>
                    <Text type="secondary">配载 / 分配备注：</Text>
                  </div>
                  <Input
                    placeholder="配载备注"
                    value={res.allocationNotes}
                    onChange={(e) => {
                      const updated = [...results];
                      updated[index] = {
                        ...res,
                        allocationNotes: e.target.value,
                      };
                      setResults(updated);
                    }}
                  />
                </Col>
              </Row>

              <Row gutter={16} style={{ marginTop: 12 }}>
                <Col span={24}>
                  <div style={{ marginBottom: 4 }}>
                    <Text type="secondary">操作备注：</Text>
                  </div>
                  <Input
                    placeholder="操作备注（可修改或清空）"
                    value={res.operationNotes}
                    onChange={(e) => {
                      const updated = [...results];
                      updated[index] = {
                        ...res,
                        operationNotes: e.target.value,
                      };
                      setResults(updated);
                    }}
                  />
                </Col>
              </Row>

              {res.role === 'ORIGINAL' ? (
                <div
                  style={{
                    marginTop: 12,
                    padding: 8,
                    background: '#f5f5f5',
                    borderRadius: 4,
                  }}
                >
                  <Text type="secondary">保留单分单号 (HBL No)：</Text>
                  <Text strong style={{ marginLeft: 8 }}>
                    {splitContext?.currentHouseBill?.houseNo ||
                      '无分单（直单模式）'}
                  </Text>
                </div>
              ) : (
                splitContext?.documentStructure === 'HOUSE' && (
                  <div
                    style={{
                      marginTop: 12,
                      padding: 12,
                      background: '#ffffff',
                      borderRadius: 4,
                      border: '1px dashed #91caff',
                    }}
                  >
                    <Row gutter={12}>
                      <Col span={8}>
                        <div style={{ marginBottom: 4 }}>
                          <Text strong>新分单号 (HBL No) *</Text>
                        </div>
                        <Input
                          placeholder="新分单号"
                          value={res.houseNo}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              houseNo: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                      <Col span={8}>
                        <div style={{ marginBottom: 4 }}>
                          <Text strong>签发主体</Text>
                        </div>
                        <Select
                          value={res.issuerSource || 'SELF_ORGANIZATION'}
                          onChange={(val) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              issuerSource: val,
                            };
                            setResults(updated);
                          }}
                          style={{ width: '100%' }}
                          options={[
                            {
                              label: '组织自签 (SELF_ORGANIZATION)',
                              value: 'SELF_ORGANIZATION',
                            },
                            {
                              label: '客户代签 (CUSTOMER_PARTNER)',
                              value: 'CUSTOMER_PARTNER',
                            },
                            {
                              label: '第三方代签 (OTHER_PARTNER)',
                              value: 'OTHER_PARTNER',
                            },
                          ]}
                        />
                      </Col>
                      <Col span={8}>
                        <div style={{ marginBottom: 4 }}>
                          <Text strong>分单备注</Text>
                        </div>
                        <Input
                          placeholder="可选分单备注"
                          value={res.houseBillNote}
                          onChange={(e) => {
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              houseBillNote: e.target.value,
                            };
                            setResults(updated);
                          }}
                        />
                      </Col>
                    </Row>
                  </div>
                )
              )}

              {res.targetType === 'CANDIDATE' && (
                <div
                  style={{
                    marginTop: 12,
                    padding: 12,
                    background: '#ffffff',
                    borderRadius: 4,
                  }}
                >
                  <Space style={{ width: '100%' }}>
                    <Select
                      showSearch={{
                        filterOption: false,
                        onSearch: async (keyword) => {
                          const options =
                            await searchShippingLineOptions(keyword);
                          setCarrierOptions(options);
                        },
                      }}
                      placeholder="选择船公司"
                      style={{ width: 220 }}
                      value={res.shippingLineId}
                      options={carrierOptions}
                      onChange={(value) => {
                        const updated = [...results];
                        updated[index] = {
                          ...res,
                          shippingLineId: value,
                          candidateId: undefined,
                          candidateVersion: undefined,
                          candidateTeId: undefined,
                          candidateTeVersion: undefined,
                        };
                        setResults(updated);
                      }}
                    />
                    <Input
                      placeholder="输入已有草稿提单号 (MBL No)"
                      style={{ width: 260 }}
                      value={res.masterNo}
                      onChange={(e) => {
                        const updated = [...results];
                        updated[index] = {
                          ...res,
                          masterNo: e.target.value,
                          candidateId: undefined,
                          candidateVersion: undefined,
                          candidateTeId: undefined,
                          candidateTeVersion: undefined,
                        };
                        setResults(updated);
                      }}
                    />
                    <Button
                      onClick={async () => {
                        if (!res.masterNo) {
                          message.warning('请先输入提单号');
                          return;
                        }
                        if (!/^[A-Za-z0-9]+$/.test(res.masterNo)) {
                          message.warning(
                            '提单号只能包含英文字母和阿拉伯数字，不能包含空格或符号',
                          );
                          return;
                        }
                        if (!res.shippingLineId) {
                          message.warning('请先选择船公司');
                          return;
                        }
                        try {
                          const resp =
                            await orderServiceMatchSeaMasterBillCandidate({
                              masterNo: res.masterNo,
                              shippingLineId: res.shippingLineId,
                            });
                          if (resp?.matched && resp.candidate) {
                            const c = resp.candidate;
                            const transportExecutions =
                              c.transportExecutions ?? [];
                            if (transportExecutions.length !== 1) {
                              message.error(
                                '该 MBL 存在多个实际航次，请改用整票改配明确选择目标航次',
                              );
                              return;
                            }
                            const te = transportExecutions[0];
                            if (!c.id || !c.version || !te?.id || !te.version) {
                              message.error(
                                '候选母单或运输执行缺少版本信息，无法选择！',
                              );
                              return;
                            }
                            const updated = [...results];
                            updated[index] = {
                              ...res,
                              masterNo: c.masterNo,
                              candidateId: c.id,
                              candidateVersion: String(c.version),
                              candidateTeId: te.id,
                              candidateTeVersion: String(te.version),
                              shippingLineId: te?.shippingLineId,
                              vesselName: te?.vesselName,
                              voyageNo: te?.voyageNo,
                              originLocationId: te?.originLocationId,
                              dischargeLocationId: te?.dischargeLocationId,
                              transitLocationId: te?.transitLocationId,
                              etd: te?.etd
                                ? dayjs(te.etd).format('YYYY-MM-DD HH:mm:ss')
                                : undefined,
                              eta: te?.eta
                                ? dayjs(te.eta).format('YYYY-MM-DD HH:mm:ss')
                                : undefined,
                            };
                            setResults(updated);
                            message.success(
                              `成功匹配到共享母单 [${c.masterNo}]，版本: ${c.version}`,
                            );
                            triggerPreview(updated);
                          } else {
                            message.warning('未找到匹配的草稿候选母单');
                          }
                        } catch (error: unknown) {
                          message.error(
                            getErrorMessage(error, '匹配候选母单失败'),
                          );
                        }
                      }}
                    >
                      匹配已有母单
                    </Button>
                    {res.candidateId && (
                      <Tag color="success">
                        已匹配 ID: {res.candidateId.slice(0, 8)} (v
                        {res.candidateVersion}) {res.vesselName} {res.voyageNo}
                      </Tag>
                    )}
                  </Space>
                </div>
              )}

              {res.targetType === 'NEW' && (
                <div
                  style={{
                    marginTop: 12,
                    padding: 12,
                    background: '#ffffff',
                    borderRadius: 4,
                  }}
                >
                  <Row gutter={12}>
                    <Col span={6}>
                      <Input
                        placeholder="新母单号 (MBL No)"
                        value={res.masterNo}
                        onChange={(e) => {
                          const updated = [...results];
                          updated[index] = {
                            ...res,
                            masterNo: e.target.value,
                          };
                          setResults(updated);
                        }}
                      />
                    </Col>
                    <Col span={6}>
                      <Select
                        showSearch={{
                          filterOption: false,
                          onSearch: async (keyword) => {
                            const opts =
                              await searchShippingLineOptions(keyword);
                            setCarrierOptions(opts);
                          },
                        }}
                        placeholder="选择船公司"
                        style={{ width: '100%' }}
                        value={res.shippingLineId}
                        options={carrierOptions}
                        onChange={(val) => {
                          const updated = [...results];
                          updated[index] = {
                            ...res,
                            shippingLineId: val,
                          };
                          setResults(updated);
                        }}
                      />
                    </Col>
                    <Col span={6}>
                      <Input
                        placeholder="船名"
                        value={res.vesselName}
                        onChange={(e) => {
                          const updated = [...results];
                          updated[index] = {
                            ...res,
                            vesselName: e.target.value,
                          };
                          setResults(updated);
                        }}
                      />
                    </Col>
                    <Col span={6}>
                      <Input
                        placeholder="航次"
                        value={res.voyageNo}
                        onChange={(e) => {
                          const updated = [...results];
                          updated[index] = {
                            ...res,
                            voyageNo: e.target.value,
                          };
                          setResults(updated);
                        }}
                      />
                    </Col>
                  </Row>
                </div>
              )}
            </Card>
          </Col>
        ))}
      </Row>
    </SectionCard>
  );
}
