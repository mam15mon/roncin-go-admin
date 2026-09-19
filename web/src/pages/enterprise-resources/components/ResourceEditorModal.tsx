import { UploadOutlined } from '@ant-design/icons';
import type { FormInstance, UploadFile } from 'antd';
import {
  Checkbox,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Upload,
} from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import type { RcFile } from 'antd/es/upload';
import type { Dispatch, SetStateAction } from 'react';
import { isRequestTimeoutError } from '@/requestErrorConfig';
import { enterpriseResourceServicePrepareEnterpriseResourceImageUpload } from '@/services/roncin/enterpriseResourceService';
import { LONG_REQUEST_TIMEOUT } from '@/utils/requestTimeout';
import {
  addressTypes,
  type EditorValues,
  imageChecksum,
  partyTypes,
  type ResourceTab,
  remarkTypes,
} from '../resourceConstants';

type ResourceEditorModalProps = {
  active: ResourceTab;
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  editing?: API.EnterpriseResource;
  saving: boolean;
  form: FormInstance<EditorValues>;
  onSave: () => void;
  imageFiles: UploadFile[];
  setImageFiles: Dispatch<SetStateAction<UploadFile[]>>;
  capabilities?: API.GetEnterpriseResourceCapabilitiesResponse;
  partnerOptions: { label: string; value: string }[];
  searchPartners: (keyword?: string) => Promise<void>;
  assigneeOptions: { label: string; value: string }[];
  searchAssignees: (keyword?: string) => Promise<void>;
  provinceOptions: { label: string; value: string }[];
  cityOptions: { label: string; value: string }[];
  districtOptions: { label: string; value: string }[];
  countryCode?: string;
  provinceCode?: string;
  cityCode?: string;
  tagGroups: API.EnterpriseTagGroup[];
  message: MessageInstance;
};

/** 企业资源新建 / 编辑弹窗：按资源类型渲染地址、备注、往来单位、标签、图片分区表单。 */
export default function ResourceEditorModal({
  active,
  open,
  setOpen,
  editing,
  saving,
  form,
  onSave,
  imageFiles,
  setImageFiles,
  capabilities,
  partnerOptions,
  searchPartners,
  assigneeOptions,
  searchAssignees,
  provinceOptions,
  cityOptions,
  districtOptions,
  countryCode,
  provinceCode,
  cityCode,
  tagGroups,
  message,
}: ResourceEditorModalProps) {
  return (
    <Modal
      title={editing ? `编辑${active.label}` : `新建${active.label}`}
      open={open}
      width={760}
      confirmLoading={saving}
      onOk={onSave}
      onCancel={() => setOpen(false)}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Space align="start" size={16} style={{ width: '100%' }}>
          <Form.Item
            name="shortName"
            label="简称/名称"
            rules={[{ required: true }]}
          >
            <Input style={{ width: 320 }} maxLength={200} />
          </Form.Item>
          <Form.Item name="sortOrder" label="排序">
            <InputNumber min={0} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Space>
        <Form.Item name="partnerIds" label="关联企业">
          <Select
            mode="multiple"
            showSearch={{
              filterOption: false,
              onSearch: (value) => void searchPartners(value),
            }}
            onFocus={() => void searchPartners()}
            options={partnerOptions}
            placeholder="可不选择，保存为独立资源"
          />
        </Form.Item>
        {active.type === 1 && (
          <>
            <Space size={16}>
              <Form.Item name="contactName" label="联系人">
                <Input />
              </Form.Item>
              <Form.Item name="contactPhone" label="联系电话">
                <Input />
              </Form.Item>
              <Form.Item
                name="countryCode"
                label="国家/地区"
                rules={[{ required: true }]}
              >
                <Input maxLength={2} />
              </Form.Item>
            </Space>
            {countryCode === 'CN' && (
              <Space size={16}>
                <Form.Item name="provinceCode" label="省">
                  <Select
                    style={{ width: 180 }}
                    allowClear
                    options={provinceOptions}
                    onChange={() =>
                      form.setFieldsValue({
                        cityCode: undefined,
                        districtCode: undefined,
                      })
                    }
                  />
                </Form.Item>
                <Form.Item name="cityCode" label="市">
                  <Select
                    style={{ width: 180 }}
                    allowClear
                    disabled={!provinceCode}
                    options={cityOptions}
                    onChange={() =>
                      form.setFieldValue('districtCode', undefined)
                    }
                  />
                </Form.Item>
                <Form.Item name="districtCode" label="区县">
                  <Select
                    style={{ width: 180 }}
                    allowClear
                    disabled={!cityCode}
                    options={districtOptions}
                  />
                </Form.Item>
              </Space>
            )}
            <Form.Item
              name="addressDetail"
              label="详细地址"
              rules={[{ required: true }]}
            >
              <Input.TextArea rows={3} maxLength={1000} />
            </Form.Item>
            <Form.Item name="addressTypes" label="地址类型">
              <Checkbox.Group options={addressTypes} />
            </Form.Item>
            <Form.Item name="assigneeIds" label="关联人员">
              <Select
                mode="multiple"
                showSearch={{
                  filterOption: false,
                  onSearch: (value) => void searchAssignees(value),
                }}
                onFocus={() => void searchAssignees()}
                options={assigneeOptions}
                placeholder="搜索当前组织人员"
              />
            </Form.Item>
            <Form.Item name="addressRemark" label="备注">
              <Input.TextArea rows={2} maxLength={500} />
            </Form.Item>
          </>
        )}
        {active.type === 2 && (
          <>
            <Form.Item
              name="remarkType"
              label="备注类型"
              rules={[{ required: true }]}
            >
              <Select options={remarkTypes} />
            </Form.Item>
            <Form.Item
              name="content"
              label="备注内容"
              rules={[{ required: true }]}
            >
              <Input.TextArea rows={8} maxLength={4000} showCount />
            </Form.Item>
          </>
        )}
        {partyTypes.has(active.type) && (
          <>
            <Space size={16}>
              <Form.Item
                name="companyName"
                label="企业/单证主体名称"
                rules={[{ required: true }]}
              >
                <Input style={{ width: 320 }} />
              </Form.Item>
              <Form.Item name="businessCode" label="企业代码">
                <Input />
              </Form.Item>
              <Form.Item
                name="countryCode"
                label="国家代码"
                rules={[{ required: true }]}
              >
                <Input maxLength={2} />
              </Form.Item>
            </Space>
            <Form.Item name="partyAddress" label="企业地址">
              <Input.TextArea rows={3} />
            </Form.Item>
            <Space size={16}>
              <Form.Item name="contactName" label="联系人">
                <Input />
              </Form.Item>
              <Form.Item name="contactPhone" label="电话">
                <Input />
              </Form.Item>
              <Form.Item name="email" label="邮箱">
                <Input />
              </Form.Item>
            </Space>
            <Space size={16}>
              <Form.Item name="taxIdentifier" label="税号">
                <Input />
              </Form.Item>
              <Form.Item name="aeoCode" label="AEO 代码">
                <Input />
              </Form.Item>
              <Form.Item
                name="customDisplay"
                label="自定义页面展示"
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
            </Space>
            <Form.Item
              noStyle
              shouldUpdate={(previous, current) =>
                previous.customDisplay !== current.customDisplay
              }
            >
              {({ getFieldValue }) =>
                getFieldValue('customDisplay') && (
                  <Form.Item name="displayContent" label="页面展示内容">
                    <Input.TextArea rows={5} maxLength={4000} />
                  </Form.Item>
                )
              }
            </Form.Item>
            <Form.Item name="partyRemark" label="内部备注">
              <Input.TextArea rows={2} />
            </Form.Item>
          </>
        )}
        {active.type === 4 && (
          <Form.Item name="groupId" label="标签组" rules={[{ required: true }]}>
            <Select
              options={tagGroups.flatMap((group) =>
                group.id ? [{ value: group.id, label: group.name }] : [],
              )}
            />
          </Form.Item>
        )}
        {active.type === 3 && (
          <Form.Item label="图片" required>
            <Upload.Dragger
              accept="image/jpeg,image/png,image/bmp,image/gif"
              maxCount={1}
              fileList={imageFiles}
              beforeUpload={(file) => {
                if (file.size > Number(capabilities?.imageMaxFileSize)) {
                  message.error(
                    `图片不能超过 ${Number(capabilities?.imageMaxFileSize) / 1024 / 1024} MiB`,
                  );
                  return Upload.LIST_IGNORE;
                }
                return true;
              }}
              customRequest={async ({ file, onError, onSuccess }) => {
                try {
                  const source = file as RcFile;
                  const checksum = await imageChecksum(source);
                  const prepared =
                    await enterpriseResourceServicePrepareEnterpriseResourceImageUpload(
                      {
                        fileName: source.name,
                        mimeType: source.type,
                        fileSize: String(source.size),
                        checksum,
                      },
                    );
                  if (!prepared.uploadUrl || !prepared.objectKey)
                    throw new Error('未获得上传凭证');
                  const response = await fetch(prepared.uploadUrl, {
                    method: 'PUT',
                    headers: prepared.headers,
                    body: source,
                    signal: AbortSignal.timeout(LONG_REQUEST_TIMEOUT),
                  });
                  if (!response.ok)
                    throw new Error(`对象存储上传失败：${response.status}`);
                  const uploadResponse = {
                    objectKey: prepared.objectKey,
                    checksum,
                  };
                  setImageFiles([
                    {
                      uid: source.uid,
                      name: source.name,
                      status: 'done',
                      originFileObj: source,
                      response: uploadResponse,
                    },
                  ]);
                  onSuccess?.(uploadResponse);
                } catch (error) {
                  onError?.(error as Error);
                  message.error(
                    isRequestTimeoutError(error)
                      ? '图片上传超时，请确认文件状态后重试'
                      : error instanceof Error
                        ? error.message
                        : '图片上传失败',
                  );
                }
              }}
              onChange={({ fileList }) => setImageFiles(fileList)}
              onRemove={() => {
                setImageFiles([]);
                return true;
              }}
            >
              <p>
                <UploadOutlined /> 点击或拖拽图片上传
              </p>
              <p>
                支持 JPG、PNG、BMP、GIF，最大{' '}
                {Number(capabilities?.imageMaxFileSize) / 1024 / 1024} MiB
              </p>
            </Upload.Dragger>
          </Form.Item>
        )}
      </Form>
    </Modal>
  );
}
