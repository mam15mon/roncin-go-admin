import { ProFormText } from '@ant-design/pro-components';

export default function CompanyEnglishNameField() {
  return (
    <ProFormText
      name="nameEn"
      label="公司英文名称"
      placeholder="请输入公司正式英文名称"
      extra="用于自签 HBL 的公司抬头；未填写时不能预览、打印或下载 HBL。"
      rules={[
        {
          validator: async (_, value?: string) => {
            const text = value?.trim() ?? '';
            if ([...text].length > 200)
              throw new Error('公司英文名称不能超过200个字符');
            if (/[\p{Cc}\p{Zl}\p{Zp}]/u.test(text))
              throw new Error('公司英文名称不能包含换行或控制字符');
          },
        },
      ]}
    />
  );
}
