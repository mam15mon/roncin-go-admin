# 执行计划：空运出口危险品包装类别 PG

复杂任务（跨层契约 + 迁移 + 生成物），按步骤提交，每步独立可验证。

## Step 1：契约 + 服务端 + 迁移

- [ ] `order.proto`：enum + 三消息字段（注释三要素：DGR 适用范围 /
      UNSPECIFIED 双语义 / UN↔PG 查表钩子），`make -C server api`。
- [ ] biz 领域类型与输入结构；service FromAPI/ToAPI + create/update 接线；
      data 写入（Clear 语义）与读侧投影。
- [ ] ent schema + 迁移 SQL（`character varying NULL`，禁 IF EXISTS）。
- [ ] `go -C server generate`（如生成 ent 代码需要）；补充 service 转换
      单测（FromAPI/ToAPI 各值 + nil 分支）。
- [ ] 验证：`go -C server test -p 32 ./...`、`go -C server vet ./...`、
      `pnpm run migrate:dev`。
- [ ] 提交 `feat: 订单契约新增危险品包装类别枚举字段`（proto 源 + pb 生成物
      + 服务端 + 迁移同组）。

## Step 2：web client 生成 + 前端

- [ ] `pnpm run generate:web-client`（openapi + client + enums.generated）。
- [ ] `SeaDangerousGoodsFields`：`showPackingGroup` prop + PG 字段 +
      清值列表；`SeaDangerousGoodsFields.test.tsx` 补 prop 行为
      （传 prop 渲染第四字段、不传无、清值含 PG）。
- [ ] air-template 传 `showPackingGroup`；air adapter 序列化 + 回填。
- [ ] 测试：air form-adapter（payload 携带/不携带/回填）、air-create-layout
      （空运卡内存在 PG、海运卡无 PG 由组件测试锁定）。
- [ ] 验证：`pnpm --dir web tsc`、Biome、定向 vitest（air 两个测试文件 +
      SeaDangerousGoodsFields.test.tsx + new.test.tsx）。
- [ ] 提交 `feat: 空运出口危险品区接入包装类别字段`。

## Step 3：全量门禁

- [ ] `pnpm run check:server`（或按风险以 Step1/2 已跑的 go test/vet 为准）
- [ ] `pnpm run check:architecture`
- [ ] `git diff --stat` 总览：海运 adapter/模板零改动（共享组件可选 prop
      除外），生成物与源同组。

## 回滚点

Step 1、Step 2 各自独立提交；异常单独 revert，列 nullable 遗留无副作用。

## 明确不做

UN↔PG 查表校验、闪点、海洋污染物、海运渲染变化、权限码/审计/列表列。
