# 现场订单显式补链

## 授权与预检

用户已明确授权修复SE2026110800002，恢复原主单和原航程，保留旧记录；不直接SQL写入。

默认预检已通过，applied=false：
- orderID=01a1bbd3-18fe-73e5-b185-9c6b5ae7f788，公司01a0c1b1-a30b-7747-8fae-2dd785dfdac3，version=7。
- endedLink=01a1bbd3-1906-7d3c-b37f-212871a37440，原MBL=01a1bbd3-1904-7db1-85cb-24d6136cfd71，原TE=01a1bbd3-1903-7772-b3ed-53faed041775。
- 完成退关事件01a1bbd6-ade5-7991-ae64-8910f7c49082，旧恢复事件01a1bbf5-2df6-728c-994b-b93a51076c7c。
- 操作者为现有系统管理员01a13400-0da3-76a9-a83b-509e8fcf41af；命令已实际校验同公司read/transition权限，未仅凭actorID跳过授权。
- 默认预检未执行apply，实际写入须等独立review和门禁通过后。

## 执行与复核

独立review及完整门禁通过后，最终重新预检仍为version7/applied=false，随即按已授权精确目标--apply成功。

- apply后version=8，新ACTIVE Link=01a1bc0e-f08b-7968-a98a-82635a7f0b9c、初版1，原MBL/TE保持不变。
- 正式HTTP API读取订单详情、海运单证、主单版本列表及事件均成功；原主单CODEXHISTORY1108、原航程、HBL均存在，版本列表[2,1]，事件总数1。
- 只读SQL核对旧Link仍ENDED/version3、ended_at和reason未改；新Link唯一ACTIVE/version1。
- 独立修复审计order.sea_restore_link.repair恰好1条；TERMINATION事件仍3条，没有伪造ACTIVE→ACTIVE或再次退关事件。
- 未修改费用/提成或其他订单；未复活旧关联、未直接SQL写业务数据。
