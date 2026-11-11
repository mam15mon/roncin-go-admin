# 资源覆盖导入权限边界

## 目标与需求

对应父任务 R2：仅具备新建权限的用户能无冲突导入新记录，但不能覆盖任何已有记录；覆盖需要新建与编辑权限。服务端拒绝伪造覆盖请求且整批不部分落库；前端明确呈现是否允许覆盖。

## 问题证据

server/internal/service/enterprise_resource.go:347；server/api/enterprise_resource/v1/enterprise_resource.proto:84；server/internal/data/enterprise_resource.go:554；web/src/pages/enterprise-resources/index.tsx:603、710。

## 验收

- [ ] create-only 新增成功。
- [ ] create-only 冲突预览不可覆盖且伪造覆盖 403。
- [ ] create+update 覆盖成功。
- [ ] update-only 不能导入。
- [ ] 混合新建/覆盖无权限整批零写入。
- [ ] 公司隔离。。

## 交付边界

独立交付；web/src/pages/enterprise-resources/index.tsx 同 R3 相交，主协调者串行整合该文件或由同一负责人修改。
不重置数据库、不自动纠正已有业务金额；保留无关工作。按父任务授权完成独立 review、合并主分支和 push。
