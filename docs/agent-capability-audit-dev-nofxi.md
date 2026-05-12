# NOFXi Agent 功能与知识点审计（dev-nofxi）

日期：2026-05-12

## 审计范围

这份文档只审计当前 `dev-nofxi` 分支里的 agent 能力、产品知识、前后端字段映射和容易导致对话混乱的代码结构，不包含本次代码修复。

重点检查对象：

- `agent/skills/*.json`
- `agent/central_brain.go`
- `agent/skill_domain_context.go`
- `agent/llm_flow_extractor.go`
- `agent/skill_management_handlers.go`
- `agent/skill_dispatcher.go`
- `agent/tools.go`
- `agent/workflow.go`
- `agent/planner_runtime.go`
- 前端真实字段：`web/src/components/trader/*`、`web/src/components/strategy/*`、`web/src/types/strategy.ts`

## 总结结论

当前 agent 已经覆盖了模型、交易所、策略、交易员、诊断、行情、账户、交易执行等主要能力，但还没有达到“比开发更熟悉产品”的状态。主要问题不是缺少某一个 prompt，而是产品知识分散在多层代码里：skill JSON、Go handler、中央 brain prompt、字段 catalog、tool schema、legacy workflow、前端表单各维护一份，导致同一个能力可能出现不同说法、不同字段、不同必填逻辑。

需要优先解决的问题：

1. **路由和引导链路太多**：中央 brain、planner fallback、legacy workflow、skill session、tool schema 都会参与决策，容易互相打架。
2. **产品字段知识不统一**：前端字段、skill JSON、Go 校验、agent 引导文案之间存在漂移。
3. **新手引导仍像表单**：很多回复是在列字段，而不是先解释当前步骤、给少量选择、再把用户选择写入草稿。
4. **交易所 DEX 字段提取不完整**：Aster/Lighter/Hyperliquid 等字段在 skill schema 里有，但部分 LLM extraction 合并逻辑没有完整接收。
5. **诊断能力没有完全对齐页面能力**：能查日志、决策、配置，但策略预览、试跑、交易所引导、模型钱包解释等还没有统一成产品级诊断流程。
6. **旧逻辑和硬规则仍多**：大量 `containsAny`、legacy fallback、deterministic fallback、领域 prompt 特例仍存在，容易造成“修一个坏一个”。

## 当前 Agent 能力地图

### 1. 模型管理

已具备：

- 创建模型配置
- 修改模型名称、provider、model name、base URL、API key、启用状态
- 查询模型列表和详情
- 删除模型配置
- 模型诊断，包括 API key、provider、模型名、Claw402/Blockrun 钱包类问题

不清晰或缺失：

- 前端模型选择是两步：选择 provider/model，再填 API 或钱包配置；agent 现在更像槽位收集。
- `model_management.json` 支持 `blockrun-base` / `blockrun-sol`，但前端 `AI_PROVIDER_CONFIG` 和 Blockrun 模型列表的关系没有在 agent 里统一描述。
- Claw402、Blockrun 这类链上支付模型的余额、主钱包、代理钱包、API key 之间的边界需要结构化知识，不应该靠临时 prompt。
- 创建模型时到底允许“草稿无 key”还是必须有 key，agent JSON 和 tool 行为描述不完全一致。

建议修改：

- 建立统一的模型产品目录，字段包括 provider、展示名、是否需要 API key、是否需要钱包、是否可直接测试、默认模型、前端是否展示。
- 模型创建改成新手流程：先选供应商，再解释需要什么凭证，最后确认创建。
- 模型诊断统一走“读取配置 -> 检查凭证/钱包 -> 解释原因 -> 给下一步”的证据包，不靠关键词猜测。

### 2. 交易所管理

已具备：

- 创建交易所账户
- 修改账户名、交易所类型、API key、secret、passphrase、testnet、enabled
- 查询交易所配置
- 删除交易所配置
- 交易所诊断，包括连接失败、签名、时间戳、权限、IP 白名单、余额、下单、持仓模式等

不清晰或缺失：

- Aster、Lighter、Hyperliquid 的字段和 CEX 字段不同，但部分旧 extraction 逻辑仍按 `api_key + secret_key` 收集。
- `exchange_management.json` 里已经有 Aster/Lighter/Hyperliquid 字段，但 `applyLLMExtractionToSkillSession` 没有完整合并这些字段。
- Aster 的 `aster_user` 容易被误解成用户名；它实际对应前端的 Aster 主钱包地址。
- `entity_field_catalog.go` 里 Aster 字段包含泛化关键词 `user`，可能在上下文不足时误映射。
- 前端有交易所注册链接、服务器 IP 指引、WebCrypto 检查等信息，agent 没有形成完整产品知识。

建议修改：

- 交易所字段必须按 `exchange_type` 动态决定，不能在通用 missing fields 里固定要 `api_key/secret_key`。
- exchange extraction 接收所有前端字段：Aster、Lighter、Hyperliquid 的专属字段都要能从用户文本写入草稿。
- 把 Aster 字段解释放到字段目录里，而不是散落在 prompt 特例中。
- 新手流程：先选交易所 -> 账户显示名 -> 展示该交易所专属凭证说明 -> 用户填完后做安全确认。

### 3. 策略管理

已具备：

- 创建 AI 策略和网格策略
- 修改策略名称、prompt、配置字段
- 查询策略列表和详情
- 删除、复制、激活策略
- AI 策略字段和网格策略字段已经有分支隔离逻辑
- 策略诊断包含选币、K 线、AI 决策、风险过滤等方向

不清晰或缺失：

- 策略字段知识散在 `strategy_management.json`、`strategy_field_catalog.go`、`tools.go`、`skill_management_handlers.go`、前端 StrategyStudio 里。
- AI 策略和网格策略有些字段容易串线，虽然已经有局部修复，但架构上仍依赖多处硬判断。
- 前端有策略 prompt 预览和 AI 测试运行能力，agent 当前没有作为明确工具暴露。
- 系统强制字段、用户可改字段、仅解释字段没有统一标记，agent 容易让用户填写页面上不存在或不该填的字段。
- 新手引导仍可能一次列太多字段，或变成“字段表单”，没有自然对话感。

建议修改：

- 建立策略 schema 单一来源，字段必须有：适用模板、是否前端可编辑、是否必填、范围、默认值、解释、是否敏感、是否系统强制。
- AI 策略新手流程按逻辑分组，而不是逐字段：
  1. 策略目标和名称
  2. 风格预设：稳健 / 标准 / 进取 / 自定义
  3. 选币逻辑：AI500 / OI Top / OI Low / 静态币种
  4. 周期和指标：少量解释后给推荐组合
  5. 量化数据和风控边界
  6. prompt 摘要和最终确认
- 网格策略新手流程：
  1. 交易对和资金
  2. 网格数量、杠杆、价格边界
  3. 止损、回撤、日亏损、Maker
  4. 最终确认
- agent 可以推荐，但必须把推荐后的草稿展示给用户确认，并允许用户追问“为什么这样填”。
- 增加策略 prompt 预览和 test-run 工具能力，严格对齐前端。

### 4. 交易员管理

已具备：

- 创建交易员
- 绑定 exchange / model / strategy
- 修改扫描间隔、全仓/逐仓、比赛展示
- 启动、停止、删除交易员
- 查询交易员状态、余额、持仓、决策、日志
- 交易员诊断覆盖不交易、启动失败、模型错误、交易所错误、策略过滤等方向

不清晰或缺失：

- 交易员创建应该是“组装已有模型、交易所、策略”，但 agent 现在有时会把依赖创建和交易员创建混在一起。
- 前端可以选择已有配置，agent 也能查列表，但引导还不够像新手流程。
- 启动交易员是高风险操作，应该有统一确认机制，不应该只靠某一层 skill prompt。
- `toolCreateTrader` 里还有一些看起来偏旧的字段，如 `UseAI500`、`UseOITop`、`TradingSymbols`、`CustomPrompt`、`OverrideBasePrompt` 等，需要确认是否仍被后端使用。

建议修改：

- 交易员创建流程：先解释交易员是“模型 + 交易所 + 策略”的组合，再展示已有可选项。
- 如果缺模型/交易所/策略，agent 应该主动引导先创建缺失依赖，而不是让用户猜。
- 启动、停止、删除统一走风险确认。
- 清理或确认旧字段是否还被运行时使用，不再使用就删除。

### 5. 诊断能力

已具备：

- 模型诊断
- 交易所诊断
- 策略诊断
- 交易员诊断
- 后端日志读取
- 决策记录读取
- 余额、持仓、成交历史、行情、K 线、候选币等工具

不清晰或缺失：

- 诊断 prompt 里有很多固定错误码和关键词规则，容易变成硬规则。
- 诊断流程没有完全产品化成“证据包 -> LLM 判断 -> 用户可执行下一步”。
- 策略页面的 prompt 预览、试跑结果、候选币来源没有完全纳入诊断。
- 交易所页面的服务器 IP、权限、注册链接、测试连接引导没有完全纳入 agent 知识。

建议修改：

- 每个诊断都先收集结构化证据包，再交给 LLM 做判断。
- 固定错误码可以作为知识库条目，但不要写成直接覆盖用户问题的硬分支。
- 诊断输出只回答当前问题，不要顺手展开无关配置。

## 代码结构问题清单

| 优先级 | 区域 | 当前问题 | 影响 | 建议 |
| --- | --- | --- | --- | --- |
| P0 | `llm_flow_extractor.go` | 交易所 missing fields 和 extraction 对 DEX 字段支持不完整 | Aster/Lighter/Hyperliquid 创建容易问错字段或丢字段 | 按 exchange_type 动态生成必填字段，完整接收前端所有字段 |
| P0 | `central_brain.go` + `workflow.go` + `planner_runtime.go` | 多套路由和 fallback 并存 | 同一句话可能被不同层解释，导致 topic 污染和循环 | 收敛到中央 brain + active session，旧 workflow 分阶段删除 |
| P0 | 策略字段体系 | 字段散在多处，AI/grid 分支依赖多处判断 | 容易串模板、编造字段、遗漏前端能力 | 建立单一产品字段目录，并由 handler/tool/prompt 共用 |
| P1 | 新手引导文案 | 多处输出字段清单 | 用户像填表，不像被引导 | 改为“解释一句 + 少量选择 + 草稿确认”的分组流程 |
| P1 | `skill_domain_context.go` | 产品知识写在长 prompt 里 | 难维护，容易变成特殊规则 | 产品事实移入结构化 catalog，prompt 只引用 |
| P1 | `tools.go` | tool schema 和 handler 有重复校验 | 同一字段可能多个默认值/范围 | 工具参数、字段目录、前端范围统一 |
| P1 | `skill_dag.go` | DAG 太浅，真实步骤在别处 | 流程不可见，难测试 | 用 DAG 表达真实新手流程和依赖 |
| P2 | `brain.go` | 新闻情绪有硬关键词 | 不符合“用户信息交给 LLM 判断” | 改为 LLM 分类或关闭该启发式 |
| P2 | `onboard.go` | 有 TODO，尚未真正创建 exchange/trader | onboarding 能力不完整 | 要么补齐，要么从 agent 能力里移除 |
| P2 | 测试 | 部分测试固定旧的字段清单回复 | 会阻碍新引导体验 | 改成流程状态、字段草稿、风险确认的 golden tests |

## 推荐改造方案

### 第一步：建立产品能力目录

新增一个结构化目录，作为 agent 理解产品的唯一来源。

目录应该覆盖：

- 模型 provider、字段、凭证类型、默认模型、前端可见性
- 交易所类型、每种交易所必填字段、字段解释、敏感字段
- 策略模板、字段、默认值、范围、依赖、是否前端可编辑
- 交易员字段、依赖关系、启动/停止/删除风险等级
- 页面操作能力：预览、测试、复制、激活、删除、诊断、导入导出等

目标：skill JSON、tool schema、handler 校验、LLM prompt 都从这里读取，不再各写各的。

### 第二步：重做四个创建流程的新手引导

不是一个字段问一次，也不是一次列全部字段，而是一个逻辑组问一次。

模型：

1. 选择供应商和模型
2. 解释需要的凭证
3. 填写并确认

交易所：

1. 选择交易所
2. 账户显示名
3. 该交易所专属凭证
4. 安全提醒和确认

策略：

1. 策略类型和目标
2. 风格预设或用户意图
3. 核心参数组
4. 风控/执行组
5. 草稿总结和确认

交易员：

1. 解释交易员由模型、交易所、策略组成
2. 选择或创建缺失依赖
3. 设置运行参数
4. 确认是否启动

### 第三步：清理旧分支和硬规则

需要重点清理：

- `workflow.go` 的关键词路由
- `planner_runtime.go` 的 legacy fallback / hard_skill fallback
- `central_brain.go` 中过多的策略 create 特例 guard
- `skill_management_handlers.go` 中分散的字段解析
- `entity_field_catalog.go` 中过泛化的关键词

保留的应该是：

- 安全校验
- 字段范围校验
- 敏感信息保护
- 高风险动作确认
- 工具调用前后的结构化验证

### 第四步：补齐前端页面能力

agent 应严格参考前端能力，至少补齐：

- 策略 prompt 预览
- 策略 AI 试跑
- 交易所专属字段和引导链接
- 模型钱包类说明和余额诊断
- 交易员依赖选择和运行状态诊断
- 页面能做但 agent 不能做的操作，需要明确是补工具还是声明不支持

### 第五步：测试改成产品流程测试

新增或改造测试：

- Aster 创建不会问用户名，只问主钱包地址、API Pro 代理钱包地址、API Pro 私钥
- Lighter/Hyperliquid 字段能被 LLM extraction 写入草稿
- AI 策略不会出现网格字段，网格策略不会出现 AI 排行字段
- “让 LLM 推荐”会生成草稿，并允许用户追问和修改
- 新手流程不会一次输出大量字段
- topic 切换后不会污染上一轮上下文
- 启动/删除/真实交易必须确认
- 不允许编造前端不存在字段

## 需要你判断的点

1. 是否允许我把旧 `workflow.go` / legacy fallback 分阶段下线，而不是继续小修小补。
2. agent 聊天页是否保留“可点击选择 UI”，如果保留，它只作为用户回复的可视化，不承担复杂表单。
3. 策略 import/export 是否也要进入 agent 能力范围。
4. 新闻/主动提醒里的关键词情绪判断是否要保留；如果严格按“都交给 LLM 判断”，建议改掉。
5. 创建流程是否按“新手引导优先”，牺牲一点速度，换取更少误填和更清晰解释。

## 建议执行顺序

1. 先做产品能力目录，统一字段、范围、解释和前端可见性。
2. 修 DEX 交易所字段 extraction 和动态必填字段。
3. 重做四个创建流程的新手引导。
4. 补齐策略 preview/test-run 和诊断证据包。
5. 清理旧 workflow/fallback。
6. 更新测试。

这套顺序的原因是：先统一知识，再改对话；否则继续改 prompt 或单个 handler，很容易继续出现“修一个问题，另一个入口又错”的情况。
