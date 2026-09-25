# backend/coding Specification

## Purpose
定义后端代码的书写规范，统一命名、注释、错误处理、日志、context 传递与并发控制的约定，使代码在静态检查阶段即可拦截常见缺陷，并保证线上问题可以凭日志与请求标识定位。

## Requirements

### Requirement: 标识符与注释的语言分工

代码标识符（包名、类型、函数、变量、字段）SHALL 使用英文。注释 SHALL 使用简体中文，说明「为什么这样做」而非复述代码。导出的符号 SHALL 具有以其名称开头的注释。

#### Scenario: 新增一个导出函数

- **WHEN** 开发者新增一个导出函数
- **THEN** 该函数 MUST 带有以其名称开头的注释，MUST NOT 出现无注释的导出符号

#### Scenario: 注释复述代码

- **WHEN** 注释内容仅重复代码字面含义而未提供额外信息
- **THEN** 该注释 SHOULD 被删除或改写为说明设计意图与约束

### Requirement: 错误逐层包装且不得吞掉

错误 SHALL 沿调用链逐层包装并携带上下文，MUST 使用 `%w` 保留错误链以便 `errors.Is` 与 `errors.As` 判定。MUST NOT 静默丢弃错误，MUST NOT 以空结果替代失败返回。

哨兵错误与自定义错误类型 SHALL 集中在所属域的 `errs.go` 或 `platform/errcode` 中定义，MUST NOT 在业务代码中散落匿名错误值。

#### Scenario: 底层调用失败

- **WHEN** 数据库或下游依赖调用失败
- **THEN** 错误 MUST 携带关键上下文（操作与关键标识）向上传递并记录，MUST NOT 被忽略或替换为空值返回

#### Scenario: 需要判断错误类型

- **WHEN** 上层需要区分某类错误以决定处理方式
- **THEN** 该判断 MUST 通过 `errors.Is` 或 `errors.As` 完成，MUST NOT 依赖错误字符串匹配

### Requirement: 禁止裸 panic

业务代码 MUST NOT 使用 `panic` 表达可预期的错误，MUST 以 error 返回值向上传递。`panic` MAY 仅出现在初始化阶段的不可恢复配置错误，且中间件 SHALL 统一捕获未预期的 panic 并转换为服务端错误响应，同时记录堆栈。

#### Scenario: 参数非法

- **WHEN** 函数收到非法参数
- **THEN** 该函数 MUST 返回 error，MUST NOT 以 panic 终止进程

#### Scenario: 请求处理过程中发生未预期 panic

- **WHEN** 处理请求时发生未预期的 panic
- **THEN** 中间件 MUST 捕获该 panic，返回服务端错误响应并记录堆栈，MUST NOT 使进程退出

### Requirement: 结构化日志且禁止裸打印

日志 SHALL 使用结构化日志库输出，MUST NOT 使用 `fmt.Println`、`fmt.Printf` 或标准库 `log` 直接输出。日志 SHALL 携带键值化字段，MUST NOT 以字符串拼接承载结构化信息。

每个请求的日志 SHALL 携带请求标识以支持串联排查。日志中 MUST NOT 输出密钥、令牌、密码等敏感信息。

#### Scenario: 记录一次业务失败

- **WHEN** 业务处理失败需要记录日志
- **THEN** 日志 MUST 通过结构化字段携带错误原因与关键标识，MUST NOT 仅输出拼接后的字符串

#### Scenario: 日志级别使用

- **WHEN** 开发者选择日志级别
- **THEN** Error 级别 MUST 只用于需要人工关注的异常，MUST NOT 用于可预期的业务分支（如参数校验失败）

### Requirement: 请求标识贯穿全链路

系统 SHALL 为每个进入的请求生成或透传唯一请求标识，该标识 SHALL 注入到该请求的日志与异步消息中。对外响应 SHALL 回传该标识。

#### Scenario: 排查一次失败请求

- **WHEN** 运维拿到一个失败的响应
- **THEN** 该响应 MUST 带有请求标识，凭该标识 MUST 能检索到本次请求的全部相关日志

#### Scenario: 请求触发异步任务

- **WHEN** 一个请求触发了异步消息生产
- **THEN** 消息体 MUST 携带该请求的标识，消费端日志 MUST 能关联回原始请求

### Requirement: 常量收敛

错误码、状态枚举、缓存过期时间、分页上限、并发度等取值 SHALL 以具名常量或类型化枚举定义，MUST NOT 在业务代码中出现裸字面量。同一语义的取值 MUST 只在一处定义。

#### Scenario: 使用状态值

- **WHEN** 代码需要判断或写入某个状态值
- **THEN** 该值 MUST 引用具名常量或类型化枚举，MUST NOT 直接书写字面量数字或字符串

#### Scenario: 同一取值在多处出现

- **WHEN** 同一个取值被多个文件使用
- **THEN** 该取值 MUST 收敛到单一位置定义并被引用，MUST NOT 重复声明

### Requirement: 接口保持窄小

为便于替换实现与隔离测试，接口 SHALL 保持方法数量少且语义单一。定义接口 SHALL 以调用方需要为准，MUST NOT 为每个具体类型机械地抽取接口。

#### Scenario: 为业务逻辑注入依赖

- **WHEN** 业务逻辑依赖外部存储以便测试时替换
- **THEN** 该依赖 SHALL 以窄接口形式声明，接口方法 MUST 只包含调用方实际使用的能力

### Requirement: 并发安全与 goroutine 生命周期

共享状态的读写 SHALL 使用同步原语或通过通道传递，MUST NOT 存在无保护的并发读写。每个显式启动的 goroutine SHALL 具有明确的退出路径，MUST 通过 context 取消、关闭信号或 errgroup 收敛。MUST NOT 启动无上限的并发任务。

#### Scenario: 启动后台任务

- **WHEN** 代码启动一个后台 goroutine
- **THEN** 该 goroutine MUST 能被 context 或关闭信号终止，MUST NOT 存在无法退出的常驻协程

#### Scenario: 并发处理批量数据

- **WHEN** 需要并发处理一批数据
- **THEN** 并发度 MUST 有明确上限，MUST NOT 按输入规模无界地启动 goroutine

### Requirement: 禁止直接序列化持久化实体

对外响应 MUST 由显式定义的 DTO 承载，MUST NOT 直接序列化数据库实体或持久化结构体。持久化结构体的字段增删 MUST NOT 自动改变对外接口的返回内容。

#### Scenario: 返回用户信息

- **WHEN** 接口需要返回用户信息
- **THEN** 响应 MUST 由专门的 DTO 构造并显式列明字段，MUST NOT 直接输出数据库实体

#### Scenario: 持久化结构体新增内部字段

- **WHEN** 为持久化结构体新增一个仅内部使用的字段
- **THEN** 对外接口的返回结构 MUST 保持不变，MUST NOT 因该字段而外泄内部实现细节
