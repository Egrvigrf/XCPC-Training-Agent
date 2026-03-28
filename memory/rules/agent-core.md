---
paths:
  - internal/logic/agent/**
---
修改 agent 核心时遵守：

- controller 负责事件循环，不承载过多配置拼装细节。
- prompt 只描述角色、输出约束和必要上下文，不重复工具 schema。
- session snapshot 只保留目标、确认事实、已完成/未完成事项和 artifacts。
- 除非必须，不把完整历史消息重新注入模型。
