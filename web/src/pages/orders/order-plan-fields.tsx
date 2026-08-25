import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import {
  ProFormDigit,
  ProFormList,
  ProFormSelect,
  ProFormText,
} from '@ant-design/pro-components';
import { Button, Col, Form, Input, Row, Tooltip } from 'antd';
import React from 'react';

type SelectOption = { label: string; value: string | number };

export function OrderShippingDocumentFields({
  disabled = false,
}: {
  disabled?: boolean;
} = {}) {
  const form = Form.useFormInstance();
  const [shippingDocuments, setShippingDocuments] = React.useState<
    API.OrderShippingDocumentInput[]
  >(() => {
    const initial = form?.getFieldValue('shippingDocuments');
    return Array.isArray(initial) && initial.length > 0
      ? initial.map((item) => ({
          ...item,
          id: item.id || `doc-${Math.random().toString(36).slice(2, 9)}`,
        }))
      : [{ id: `doc-${Math.random().toString(36).slice(2, 9)}`, masterNo: '', houseNo: '' }];
  });

  React.useEffect(() => {
    const current = form?.getFieldValue('shippingDocuments');
    if (Array.isArray(current) && current.length > 0) {
      setShippingDocuments(
        current.map((item) => ({
          ...item,
          id: item.id || `doc-${Math.random().toString(36).slice(2, 9)}`,
        })),
      );
    }
  }, [form]);

  const primaryDoc = shippingDocuments[0] || { masterNo: '', houseNo: '' };
  const appendedDocs = shippingDocuments.slice(1);

  const updateDocs = (next: API.OrderShippingDocumentInput[]) => {
    setShippingDocuments(next);
    form?.setFieldValue('shippingDocuments', next);
  };

  const handlePrimaryMasterChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    const next = [...shippingDocuments];
    next[0] = { ...next[0], masterNo: val };
    updateDocs(next);
  };

  const handlePrimaryHouseChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    const next = [...shippingDocuments];
    next[0] = { ...next[0], houseNo: val };
    updateDocs(next);
  };

  const handleAppend = () => {
    const next = [
      ...shippingDocuments,
      { id: `doc-${Math.random().toString(36).slice(2, 9)}`, masterNo: '', houseNo: '' },
    ];
    updateDocs(next);
  };

  const handleAppendedMasterChange = (index: number, val: string) => {
    const next = [...shippingDocuments];
    const targetIndex = index + 1;
    if (next[targetIndex]) {
      next[targetIndex] = { ...next[targetIndex], masterNo: val };
      updateDocs(next);
    }
  };

  const handleAppendedHouseChange = (index: number, val: string) => {
    const next = [...shippingDocuments];
    const targetIndex = index + 1;
    if (next[targetIndex]) {
      next[targetIndex] = { ...next[targetIndex], houseNo: val };
      updateDocs(next);
    }
  };

  const handleRemoveAppended = (index: number) => {
    const targetIndex = index + 1;
    const next = shippingDocuments.filter((_, i) => i !== targetIndex);
    updateDocs(next.length > 0 ? next : [{ masterNo: '', houseNo: '' }]);
  };

  return (
    <Col span={24}>
      <Row gutter={16} align="top">
        {/* 主单号列 */}
        <Col className="col-5">
          <Form.Item
            label="主单号"
            style={{ marginInline: 8, marginBottom: 16 }}
          >
            <Input
              value={primaryDoc.masterNo}
              onChange={handlePrimaryMasterChange}
              placeholder="请输入主单号"
              maxLength={64}
              disabled={disabled}
              suffix={
                !disabled ? (
                  <Tooltip title="加拼主单号">
                    <Button
                      type="text"
                      size="small"
                      aria-label="加拼主单号"
                      icon={<PlusOutlined style={{ fontSize: 12, color: '#1677ff' }} />}
                      onClick={handleAppend}
                      style={{
                        height: 22,
                        width: 22,
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: 0,
                      }}
                    />
                  </Tooltip>
                ) : undefined
              }
            />
            {appendedDocs.length > 0 && (
              <div
                style={{
                  marginTop: 8,
                  padding: '8px 10px',
                  background: '#fafafa',
                  border: '1px dashed #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <div
                  style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: '#8c8c8c',
                    marginBottom: 6,
                  }}
                >
                  加拼主单号
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {appendedDocs.map((doc, idx) => (
                    <div
                      key={doc.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                      }}
                    >
                      <span
                        style={{
                          fontSize: 11,
                          fontWeight: 'bold',
                          color: '#bfbfbf',
                          width: 14,
                          textAlign: 'center',
                          flexShrink: 0,
                        }}
                      >
                        {idx + 1}
                      </span>
                      <Input
                        size="small"
                        value={doc.masterNo}
                        onChange={(e) => handleAppendedMasterChange(idx, e.target.value)}
                        placeholder="请输入主单号"
                        maxLength={64}
                        disabled={disabled}
                        style={{ flex: 1 }}
                      />
                      {!disabled && (
                        <Button
                          type="text"
                          size="small"
                          danger
                          aria-label={`删除加拼主单号${idx + 1}`}
                          icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                          onClick={() => handleRemoveAppended(idx)}
                          style={{
                            height: 24,
                            width: 24,
                            display: 'inline-flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            padding: 0,
                            flexShrink: 0,
                          }}
                        />
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </Form.Item>
        </Col>

        {/* 分单号列 */}
        <Col className="col-5">
          <Form.Item
            label="分单号"
            style={{ marginInline: 8, marginBottom: 16 }}
          >
            <Input
              value={primaryDoc.houseNo}
              onChange={handlePrimaryHouseChange}
              placeholder="请输入分单号"
              maxLength={64}
              disabled={disabled}
              suffix={
                !disabled ? (
                  <Tooltip title="加拼分单号">
                    <Button
                      type="text"
                      size="small"
                      aria-label="加拼分单号"
                      icon={<PlusOutlined style={{ fontSize: 12, color: '#1677ff' }} />}
                      onClick={handleAppend}
                      style={{
                        height: 22,
                        width: 22,
                        display: 'inline-flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        padding: 0,
                      }}
                    />
                  </Tooltip>
                ) : undefined
              }
            />
            {appendedDocs.length > 0 && (
              <div
                style={{
                  marginTop: 8,
                  padding: '8px 10px',
                  background: '#fafafa',
                  border: '1px dashed #d9d9d9',
                  borderRadius: 4,
                }}
              >
                <div
                  style={{
                    fontSize: 11,
                    fontWeight: 500,
                    color: '#8c8c8c',
                    marginBottom: 6,
                  }}
                >
                  加拼分单号
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                  {appendedDocs.map((doc, idx) => (
                    <div
                      key={doc.id}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 6,
                      }}
                    >
                      <span
                        style={{
                          fontSize: 11,
                          fontWeight: 'bold',
                          color: '#bfbfbf',
                          width: 14,
                          textAlign: 'center',
                          flexShrink: 0,
                        }}
                      >
                        {idx + 1}
                      </span>
                      <Input
                        size="small"
                        value={doc.houseNo}
                        onChange={(e) => handleAppendedHouseChange(idx, e.target.value)}
                        placeholder="请输入分单号"
                        maxLength={64}
                        disabled={disabled}
                        style={{ flex: 1 }}
                      />
                      {!disabled && (
                        <Button
                          type="text"
                          size="small"
                          danger
                          aria-label={`删除加拼分单号${idx + 1}`}
                          icon={<DeleteOutlined style={{ fontSize: 12 }} />}
                          onClick={() => handleRemoveAppended(idx)}
                          style={{
                            height: 24,
                            width: 24,
                            display: 'inline-flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            padding: 0,
                            flexShrink: 0,
                          }}
                        />
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </Form.Item>
        </Col>
      </Row>
    </Col>
  );
}

export function OrderContainerRequestFields({
  options,
}: {
  options: SelectOption[];
}) {
  return (
    <Col span={24}>
      <ProFormList
        name="containerRequests"
        label="箱型箱量"
        creatorButtonProps={{ creatorButtonText: '新增箱型箱量' }}
        creatorRecord={{ containerSpecId: '', quantity: 1 }}
        copyIconProps={false}
        deleteIconProps={{ tooltipText: '删除该箱型箱量' }}
        itemContainerRender={(doms) => <div style={{ width: '100%' }}>{doms}</div>}
      >
        <ProFormText name="id" hidden />
        <ProFormSelect
          name="containerSpecId"
          label="箱型"
          options={options}
          placeholder="请选择箱型"
          rules={[{ required: true, message: '请选择箱型' }]}
          fieldProps={{ showSearch: true, optionFilterProp: 'label' }}
          width="md"
        />
        <ProFormDigit
          name="quantity"
          label="箱量"
          min={1}
          max={999}
          fieldProps={{ precision: 0 }}
          rules={[{ required: true, message: '请输入箱量' }]}
          width="sm"
        />
      </ProFormList>
    </Col>
  );
}
