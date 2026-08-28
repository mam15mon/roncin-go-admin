import {
  AuditOutlined,
  BankOutlined,
  FileTextOutlined,
  PaperClipOutlined,
} from '@ant-design/icons';
import { Drawer, Space, Tabs, Tag } from 'antd';
import React from 'react';
import AccountsPanel from './components/secondary/AccountsPanel';
import AttachmentsPanel from './components/secondary/AttachmentsPanel';
import ContractsPanel from './components/secondary/ContractsPanel';
import InvoiceProfilesPanel from './components/secondary/InvoiceProfilesPanel';
import SettlementRulesPanel from './components/secondary/SettlementRulesPanel';

type PartnerSecondaryProps = {
  partner?: API.Partner;
  open: boolean;
  canManage: boolean;
  onClose: () => void;
};

export default function PartnerSecondary({
  partner,
  open,
  canManage,
  onClose,
}: PartnerSecondaryProps) {
  return (
    <Drawer
      title={
        <Space size={8}>
          <BankOutlined style={{ color: '#1677ff' }} />
          <span>往来商务档案：{partner?.legalName ?? ''}</span>
          {partner?.code && (
            <Tag variant="filled" style={{ fontFamily: 'monospace' }}>
              {partner.code}
            </Tag>
          )}
        </Space>
      }
      open={open}
      onClose={onClose}
      size={960}
      destroyOnHidden
    >
      <Tabs
        items={[
          {
            key: 'invoice-profile',
            label: (
              <Space size={4}>
                <FileTextOutlined />
                <span>开票资料</span>
              </Space>
            ),
            children: (
              <InvoiceProfilesPanel partner={partner} canManage={canManage} />
            ),
          },
          {
            key: 'accounts',
            label: (
              <Space size={4}>
                <BankOutlined />
                <span>结算账户</span>
              </Space>
            ),
            children: (
              <AccountsPanel partner={partner} canManage={canManage} />
            ),
          },
          {
            key: 'contracts',
            label: (
              <Space size={4}>
                <FileTextOutlined />
                <span>商务合同</span>
              </Space>
            ),
            children: (
              <ContractsPanel partner={partner} canManage={canManage} />
            ),
          },
          {
            key: 'settlement-rules',
            label: (
              <Space size={4}>
                <AuditOutlined />
                <span>结算规则</span>
              </Space>
            ),
            children: (
              <SettlementRulesPanel partner={partner} canManage={canManage} />
            ),
          },
          {
            key: 'attachments',
            label: (
              <Space size={4}>
                <PaperClipOutlined />
                <span>证照附件</span>
              </Space>
            ),
            children: (
              <AttachmentsPanel partner={partner} canManage={canManage} />
            ),
          },
        ]}
      />
    </Drawer>
  );
}
