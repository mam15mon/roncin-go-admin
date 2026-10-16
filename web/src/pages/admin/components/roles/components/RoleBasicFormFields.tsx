import { ProFormSwitch, ProFormText } from '@ant-design/pro-components';
import { Col, Row } from 'antd';

type RoleBasicFormFieldsProps = {
  editing?: API.AdminRole;
};

/** 角色基础信息表单分区：角色名称与角色状态。 */
export default function RoleBasicFormFields({
  editing,
}: RoleBasicFormFieldsProps) {
  return (
    <Row gutter={16}>
      <Col span={12}>
        <ProFormText
          name="name"
          label="角色名称"
          placeholder="例如：操作主管 / 财务专员"
          rules={[{ required: true, message: '请输入角色名称' }]}
        />
      </Col>
      <Col span={8}>
        {editing && (
          <ProFormSwitch
            name="enabled"
            label="角色状态"
            extra="停用后关联用户将失去此角色权限"
          />
        )}
      </Col>
    </Row>
  );
}
