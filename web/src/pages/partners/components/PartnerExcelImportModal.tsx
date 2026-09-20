import {
  DownloadOutlined,
  FileExcelOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Modal,
  Radio,
  Space,
  Table,
  Tag,
  Typography,
  Upload,
} from 'antd';
import type { UploadFile } from 'antd/es/upload/interface';
import React, { useState } from 'react';
import * as XLSX from 'xlsx';
import { PartnerImportMode, PartnerRoleType } from '@/enums.generated';
import { partnerServiceImportPartners } from '@/services/roncin/partnerService';
import { getErrorMessage } from '@/utils/errorMessage';
import { longRequestOptions } from '@/utils/requestTimeout';

const { Text } = Typography;

interface PartnerExcelImportModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
  currentRoleType: PartnerRoleType;
  currentRoleLabel: string;
}

const ROLE_NAME_TO_TYPE: Record<string, number> = {
  客户: PartnerRoleType.PARTNER_ROLE_TYPE_CUSTOMER,
  供应商: PartnerRoleType.PARTNER_ROLE_TYPE_SUPPLIER,
  国外代理: PartnerRoleType.PARTNER_ROLE_TYPE_FOREIGN_AGENT,
};

export default function PartnerExcelImportModal({
  open,
  onClose,
  onSuccess,
  currentRoleType,
  currentRoleLabel,
}: PartnerExcelImportModalProps) {
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importMode, setImportMode] = useState<PartnerImportMode>(
    PartnerImportMode.PARTNER_IMPORT_MODE_UPSERT,
  );
  const [parsedItems, setParsedItems] = useState<API.PartnerImportItemInput[]>(
    [],
  );
  const [parseErrors, setParseErrors] = useState<string[]>([]);

  const reset = () => {
    setFileList([]);
    setParsedItems([]);
    setParseErrors([]);
    setImporting(false);
    setImportMode(PartnerImportMode.PARTNER_IMPORT_MODE_UPSERT);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleDownloadTemplate = () => {
    const headers = [
      '单位编码*',
      '企业名称*',
      '统一社会信用代码',
      '业务角色(如: 客户;供应商)*',
      '注册地址',
      '主要联系人',
      '联系电话',
      '电子邮箱',
    ];

    const sampleRows = [
      [
        'CUST001',
        '上海示例供应链管理有限公司',
        '91310000XXXXXXXXXX',
        '客户;供应商',
        '上海市黄浦区中山东一路1号',
        '张经理',
        '13800000000',
        'contact@example.com',
      ],
      [
        'AGT001',
        'Pacific Logistics Global Ltd.',
        '',
        '国外代理',
        '100 Wall Street, New York, NY',
        'John Smith',
        '+1-212-555-0188',
        'john@pacificlogistics.com',
      ],
    ];

    const ws = XLSX.utils.aoa_to_sheet([headers, ...sampleRows]);
    // 设置列宽
    ws['!cols'] = [
      { wch: 14 },
      { wch: 30 },
      { wch: 24 },
      { wch: 28 },
      { wch: 35 },
      { wch: 14 },
      { wch: 18 },
      { wch: 26 },
    ];

    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, '往来单位导入模板');
    XLSX.writeFile(wb, '往来单位导入模板.xlsx');
    message.success('导入模板下载成功');
  };

  const parseExcelFile = async (file: File) => {
    try {
      const buffer = await file.arrayBuffer();
      const wb = XLSX.read(buffer, { type: 'array' });
      const firstSheetName = wb.SheetNames[0];
      if (!firstSheetName) {
        setParseErrors(['Excel 工作表为空']);
        return;
      }

      const ws = wb.Sheets[firstSheetName];
      const rawRows = XLSX.utils.sheet_to_json<Record<string, unknown>>(ws, {
        defval: '',
      });

      if (rawRows.length === 0) {
        setParseErrors(['未在 Excel 中解析到任何有效数据行']);
        return;
      }

      const errors: string[] = [];
      const items: API.PartnerImportItemInput[] = [];

      rawRows.forEach((row, index) => {
        const rowNum = index + 2; // 第一行为表头

        // 容错匹配列名
        const getVal = (keys: string[]): string => {
          for (const key of Object.keys(row)) {
            const cleanKey = key.replace(/[*（）()]/g, '').trim();
            if (
              keys.some(
                (k) =>
                  cleanKey.includes(k) || key.toLowerCase() === k.toLowerCase(),
              )
            ) {
              return String(row[key] ?? '').trim();
            }
          }
          return '';
        };

        const code = getVal(['单位编码', '编码', 'code']);
        const legalName = getVal([
          '企业名称',
          '名称',
          'legalName',
          'legal_name',
        ]);
        const uscc = getVal(['统一社会信用代码', '信用代码', '税号', 'uscc']);
        const rolesStr = getVal(['业务角色', '角色', 'roles']);
        const address = getVal(['注册地址', '地址', 'address']);
        const contactName = getVal(['联系人', '主要联系人', 'contact']);
        const phone = getVal(['联系电话', '电话', '手机', 'phone']);
        const email = getVal(['电子邮箱', '邮箱', 'email']);

        if (!code) {
          errors.push(`第 ${rowNum} 行: 单位编码缺失`);
          return;
        }
        if (!legalName) {
          errors.push(`第 ${rowNum} 行: 企业名称缺失`);
          return;
        }

        // 解析角色
        const parsedRoles: API.PartnerRoleInput[] = [];
        if (rolesStr) {
          const roleParts = rolesStr.split(/[,;；，]/).map((s) => s.trim());
          roleParts.forEach((part) => {
            const roleVal = ROLE_NAME_TO_TYPE[part];
            if (roleVal && !parsedRoles.some((r) => r.type === roleVal)) {
              parsedRoles.push({ type: roleVal, enabled: true });
            }
          });
        }

        // 如果用户未填角色或无法识别，默认赋予当前视图角色
        if (parsedRoles.length === 0) {
          parsedRoles.push({ type: currentRoleType, enabled: true });
        }

        // 联系人
        const contacts: API.PartnerContactInput[] = [];
        if (contactName || phone || email) {
          contacts.push({
            name: contactName || '主要联系人',
            phone,
            email,
            isPrimary: true,
          });
        }

        items.push({
          code,
          legalName,
          unifiedSocialCreditCode: uscc || undefined,
          registeredAddress: address || undefined,
          roles: parsedRoles,
          contacts,
        });
      });

      setParsedItems(items);
      setParseErrors(errors);
      if (items.length === 0 && errors.length > 0) {
        message.error(`解析失败，共发现 ${errors.length} 处错误`);
      } else if (errors.length > 0) {
        message.warning(
          `成功解析 ${items.length} 条，但有 ${errors.length} 行数据有误已跳过`,
        );
      } else {
        message.success(`成功解析 ${items.length} 条数据`);
      }
    } catch (e) {
      setParseErrors([getErrorMessage(e, '读取 Excel 文件发生错误')]);
      message.error('解析 Excel 文件失败，请确认文件格式为有效 .xlsx');
    }
  };

  const handleImport = async () => {
    if (parsedItems.length === 0) {
      message.warning('当前没有可导入的有效数据');
      return;
    }
    if (parsedItems.length > 500) {
      message.error(
        `单次最多支持导入 500 条数据，当前共 ${parsedItems.length} 条`,
      );
      return;
    }

    setImporting(true);
    try {
      const modeNumber =
        importMode === PartnerImportMode.PARTNER_IMPORT_MODE_UPSERT ? 2 : 1;
      const res = await partnerServiceImportPartners(
        {
          source: 'EXCEL',
          mode: modeNumber,
          items: parsedItems,
        },
        longRequestOptions,
      );

      message.success(
        `导入完成: 新增 ${res.createdCount ?? 0} 条，更新 ${res.updatedCount ?? 0} 条`,
      );
      onSuccess();
      handleClose();
    } catch (e) {
      message.error(getErrorMessage(e, '导入数据失败'));
    } finally {
      setImporting(false);
    }
  };

  const previewColumns = [
    {
      title: '单位编码',
      dataIndex: 'code',
      width: 120,
      render: (v: string) => (
        <Text style={{ fontFamily: 'monospace' }}>{v}</Text>
      ),
    },
    {
      title: '企业名称',
      dataIndex: 'legalName',
      width: 220,
      ellipsis: true,
    },
    {
      title: '统一信用代码',
      dataIndex: 'unifiedSocialCreditCode',
      width: 180,
      render: (v?: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '业务角色',
      dataIndex: 'roles',
      width: 160,
      render: (roles?: API.PartnerRoleInput[]) => (
        <Space size={4} wrap>
          {(roles ?? []).map((r) => {
            const label =
              r.type === 1 ? '客户' : r.type === 2 ? '供应商' : '国外代理';
            return (
              <Tag key={r.type} color="blue">
                {label}
              </Tag>
            );
          })}
        </Space>
      ),
    },
    {
      title: '注册地址',
      dataIndex: 'registeredAddress',
      ellipsis: true,
      render: (v?: string) => v || <Text type="secondary">-</Text>,
    },
  ];

  return (
    <Modal
      title={
        <Space size={8}>
          <FileExcelOutlined style={{ color: '#52c41a' }} />
          <span>导入往来单位 Excel ({currentRoleLabel})</span>
        </Space>
      }
      open={open}
      onCancel={handleClose}
      width={780}
      destroyOnHidden
      footer={[
        <Button key="cancel" onClick={handleClose} disabled={importing}>
          取消
        </Button>,
        <Button
          key="submit"
          type="primary"
          onClick={handleImport}
          loading={importing}
          disabled={parsedItems.length === 0}
        >
          {importing
            ? '正在导入...'
            : `确认导入 (${parsedItems.length} 条数据)`}
        </Button>,
      ]}
    >
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          marginTop: 16,
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '12px 16px',
            backgroundColor: '#f5f7fa',
            borderRadius: 6,
          }}
        >
          <div>
            <Text strong>导入格式说明：</Text>
            <div style={{ fontSize: 12, color: '#8c8c8c', marginTop: 4 }}>
              支持 .xlsx 格式，单次最多导入 500
              条数据。未填写的业务角色将默认归入「{currentRoleLabel}」。
            </div>
          </div>
          <Button icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>
            下载 Excel 导入模板
          </Button>
        </div>

        <div>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>
            <span>导入模式：</span>
          </div>
          <Radio.Group
            value={importMode}
            onChange={(e) => setImportMode(e.target.value)}
          >
            <Radio value={PartnerImportMode.PARTNER_IMPORT_MODE_UPSERT}>
              存在则更新 (按单位编码匹配，已存在则覆盖更新)
            </Radio>
            <Radio value={PartnerImportMode.PARTNER_IMPORT_MODE_CREATE_ONLY}>
              仅新增 (若已存在则忽略跳过)
            </Radio>
          </Radio.Group>
        </div>

        <Upload.Dragger
          accept=".xlsx, .xls"
          fileList={fileList}
          onChange={(info) => {
            setFileList(info.fileList);
          }}
          beforeUpload={(file) => {
            if (!file.name.endsWith('.xlsx') && !file.name.endsWith('.xls')) {
              message.error('只支持上传 .xlsx 或 .xls 格式的 Excel 文件');
              return false;
            }
            setFileList([file]);
            void parseExcelFile(file);
            return false;
          }}
          onRemove={() => {
            setFileList([]);
            setParsedItems([]);
            setParseErrors([]);
          }}
          maxCount={1}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined style={{ color: '#1677ff' }} />
          </p>
          <p className="ant-upload-text">点击或将 Excel 文件拖拽至此区域</p>
          <p className="ant-upload-hint">仅支持标准 .xlsx 格式文件</p>
        </Upload.Dragger>

        {parseErrors.length > 0 && (
          <Alert
            type="warning"
            showIcon
            title={`解析提示 (${parseErrors.length} 条)`}
            description={
              <div style={{ maxHeight: 80, overflowY: 'auto' }}>
                {parseErrors.slice(0, 10).map((err) => (
                  <div key={err}>{err}</div>
                ))}
                {parseErrors.length > 10 && (
                  <div>...以及更多 {parseErrors.length - 10} 处</div>
                )}
              </div>
            }
          />
        )}

        {parsedItems.length > 0 && (
          <div>
            <div style={{ marginBottom: 8 }}>
              <Text strong>
                数据预览 (前 5 条 / 共 {parsedItems.length} 条)：
              </Text>
            </div>
            <Table
              dataSource={parsedItems.slice(0, 5).map((item, i) => ({
                key: item.code || i,
                ...item,
              }))}
              columns={previewColumns}
              pagination={false}
              size="small"
              bordered
            />
          </div>
        )}
      </div>
    </Modal>
  );
}
