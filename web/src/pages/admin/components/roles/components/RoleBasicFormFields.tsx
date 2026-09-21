import { ProFormSwitch, ProFormText } from '@ant-design/pro-components';
import { Col, Row } from 'antd';
import { ProFormSearchableSelect } from '@/components/ui';
import { AdminDataScope } from '@/enums.generated';
import { dataScopeMap, dataScopeOptions } from '../roleConstants';

type RoleBasicFormFieldsProps = {
  editing?: API.AdminRole;
};

/** 角色基础信息表单分区：角色名称、数据访问范围与角色状态。 */
export default function RoleBasicFormFields({
  editing,
}: RoleBasicFormFieldsProps) {
  const retiredScope = dataScopeMap.get(AdminDataScope.DATA_SCOPE_SELF);
  return (
    <>
      <Row gutter={16}>
        <Col span={12}>
          <ProFormText
            name="name"
            label="角色名称"
            placeholder="例如：操作主管 / 财务专员"
            rules={[{ required: true, message: '请输入角色名称' }]}
          />
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={16}>
          <ProFormSearchableSelect
            name="dataScope"
            label="数据访问范围"
            extra={
              editing?.dataScope === AdminDataScope.DATA_SCOPE_SELF
                ? dataScopeMap.get(AdminDataScope.DATA_SCOPE_SELF)?.description
                : undefined
            }
            options={[
              ...dataScopeOptions,
              ...(editing?.dataScope === AdminDataScope.DATA_SCOPE_SELF &&
              retiredScope
                ? [retiredScope]
                : []),
            ].map((opt) => ({
              disabled: opt.value === AdminDataScope.DATA_SCOPE_SELF,
              label: `${opt.label} —— ${opt.description}`,
              value: opt.value,
            }))}
            rules={[
              { required: true, message: '请选择数据范围' },
              {
                validator: async (_, value) => {
                  if (value === AdminDataScope.DATA_SCOPE_SELF)
                    throw new Error('仅本人范围已停用，请明确选择新的数据范围');
                },
              },
            ]}
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
    </>
  );
}
