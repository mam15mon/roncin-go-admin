# ADR 0001: 契约单向流——proto 是接口唯一真相源

- 状态：已采纳
- 日期：2026-08-20（单仓库工程骨架建立时）
- 主题：契约与生成物

## 背景

服务端是 Go + Kratos，前端是 React + Ant Design Pro，两端靠 HTTP 接口协作，
业务字段多且迭代快。如果 Go 绑定代码、OpenAPI 文档、前端请求客户端各自
手工维护，任何一次字段变更都会产生三处同步点，漂移只会在运行时暴露。

## 决策

- `server/api/**/*.proto` 是 HTTP/gRPC 契约的唯一真相源。
- 契约变更固定走单向流程：
  1. 修改 `.proto`；
  2. `make -C server api` 生成 Go 绑定，检查生成差异；
  3. 仓库根目录 `pnpm run generate:web-client` 更新 `server/openapi.yaml`
     与前端客户端（`web/src/services/roncin/`、`web/types/`）；
  4. 源文件与生成物放在同一提交中审阅。
- 禁止手改任何生成物：`*.pb.go`、`*_grpc.pb.go`、`*_http.pb.go`、
  `openapi.yaml`、`web/src/services/roncin/`、`web/types/`。
- 前端所有请求经过统一请求配置或生成客户端，页面不得自行拼接后端主机地址。

## 理由

- 双端契约漂移是最大的腐化源；生成链把「字段类型不一致」从运行期提前到
  编译期（前端 tsc 直接报错）。
- 备选方案各有缺陷：代码优先 + 事后导出 OpenAPI 缺少稳定真相源；手写 TS
  客户端等于复制第二套契约。项目明确「无历史包袱、不建兼容层」（见
  [../../../AGENTS.md](../../../AGENTS.md) 交付阶段约定），单向生成是同步
  成本最低的方案。
- 生成差异本身就是变更评审材料：reviewer 看 `make api` 的 diff 即可知道
  契约变了什么。

## 后果

- 每次接口变更的成本固定为「改 proto + 两条生成命令」，不可跳过。
- 前端出现契约疑问时只查 proto，不读生成的 TS 反推。
- 生成结果异常必须回头修 proto 或生成配置，不得手工修补生成物。
- 后续派生约定都建立在此流之上：枚举消费用生成常量（见
  [../web/frontend/type-safety.md](../web/frontend/type-safety.md)）、
  权限键编译期生成（ADR 0002）。
