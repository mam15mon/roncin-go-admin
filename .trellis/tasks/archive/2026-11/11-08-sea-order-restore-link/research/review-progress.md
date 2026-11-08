# 独立审查跟踪

- reviewer已实际开始本期独立审查。
- 首个阻塞：锁内Order未预载Organization，直接orderToBiz会解引用组织边；已交后端改为仅构造运输校验所需字段并补DIRECT/HOUSE真实恢复正例。
- 在独立review与预检通过前，不执行开发库--apply。
