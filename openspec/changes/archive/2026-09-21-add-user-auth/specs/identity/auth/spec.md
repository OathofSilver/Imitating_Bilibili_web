# Spec Delta

## ADDED Requirements

### Requirement: 用户注册

系统 SHALL 提供基于用户名与密码的注册能力。用户名 SHALL 全局唯一且大小写不敏感地判定重复，重复注册 MUST 返回明确的冲突错误码，MUST NOT 静默覆盖既有账号。密码 MUST 以不可逆哈希形式存储，MUST NOT 以明文或可逆加密形式落库。注册成功后系统 SHALL 直接签发登录凭证，MUST NOT 要求用户二次登录。

#### Scenario: 注册成功

- **WHEN** 用户提交未被占用的用户名与符合强度要求的密码
- **THEN** 系统 MUST 创建用户并直接返回登录凭证，且响应 MUST NOT 包含密码或密码哈希

#### Scenario: 用户名已被占用

- **WHEN** 用户提交注册请求而该用户名已被占用（含仅大小写不同的情况）
- **THEN** 系统 MUST 返回用户名冲突错误码，MUST NOT 创建新用户

#### Scenario: 密码不符合强度要求

- **WHEN** 用户提交的密码不满足最低长度或字符构成约束
- **THEN** 系统 MUST 返回参数非法错误码，并在 message 中指明密码字段

### Requirement: 登录凭据校验

系统 SHALL 通过用户名与密码校验用户身份。校验失败 MUST 返回统一的不认证错误码，MUST NOT 通过不同的错误码或文案区分「用户名不存在」与「密码错误」，以避免账号枚举。

#### Scenario: 凭据正确

- **WHEN** 用户提交正确的用户名与密码
- **THEN** 系统 MUST 签发带有效期的登录凭证并返回

#### Scenario: 用户名不存在与密码错误返回一致

- **WHEN** 请求凭据中的用户名不存在，或密码与已存哈希不匹配
- **THEN** 两种情形 MUST 返回完全相同的不认证错误码与文案，MUST NOT 泄露用户名是否存在

### Requirement: 登录态撤销

系统 SHALL 支持退出登录，且撤销 MUST 在服务端生效：凭证被撤销后，携带该凭证访问任意受保护接口 MUST 被拒绝，MUST NOT 仅依赖客户端清除本地存储。

#### Scenario: 退出登录后原凭证立即失效

- **WHEN** 用户登出后再次携带原凭证请求受保护接口
- **THEN** 服务端 MUST 返回未认证错误码，MUST NOT 返回业务数据

#### Scenario: 撤销记录的生命周期

- **WHEN** 服务端记录一条被撤销的凭证
- **THEN** 该记录 MUST 至少覆盖该凭证的剩余有效期，且 MUST 在凭证自然过期后自动清理，MUST NOT 无限期累积

### Requirement: 凭证的携带方式与跨域一致性

受保护接口的登录凭证 SHALL 通过请求头 `Authorization: Bearer <token>` 携带。凭证 SHALL 采用服务端签名的自校验形式，使任一业务域服务 MUST 能独立完成有效性校验，MUST NOT 要求每个受保护请求同步调用用户域才能完成鉴权。各域校验所用的签名密钥 SHALL 来自统一配置注入，MUST NOT 硬编码在代码中。

#### Scenario: 任一域可独立校验凭证

- **WHEN** 客户端携带有效凭证请求视频域、互动域、评论域或搜索域的受保护接口
- **THEN** 该域服务 MUST 能独立完成签名与有效期校验并放行，MUST NOT 依赖用户域服务在线

#### Scenario: 缺少或格式错误的凭证

- **WHEN** 请求未携带 Authorization 头，或其格式不是 `Bearer` 加凭证
- **THEN** 服务端 MUST 返回未认证错误码，MUST NOT 继续执行受保护逻辑
