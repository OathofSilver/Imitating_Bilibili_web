package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"bilibili-web/server/app/user/api/internal/binder"
	"bilibili-web/server/app/user/api/internal/logic"
	"bilibili-web/server/common/middleware"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

// profileWritableFields 是资料修改接口接受的白名单字段。
//
// 与 types.UpdateProfileReq 的 json 标签保持一一对应，
// 由测试固化二者的一致性——两处各写一份是为了让「拒绝什么」
// 在 handler 层就可见，而不是藏在结构体定义里。
var profileWritableFields = []string{
	"nickname",
	"avatar_url",
	"signature",
	"gender",
	"birthday",
}

// identityOf 取出鉴权中间件注入的身份。
//
// 返回 middleware.ErrNoIdentity 时映射为 50001 而非 40100：
// 取不到身份说明路由漏挂了中间件，是服务端配置错误而不是客户端凭证问题。
func identityOf(r *http.Request) (middleware.Identity, error) {
	identity, err := middleware.MustIdentity(r.Context())
	if err != nil {
		return middleware.Identity{}, fmt.Errorf("%w: %v", logic.ErrDependencyUnavailable, err)
	}

	return identity, nil
}

// pathInt64 读取路径参数并转为 int64。
//
// 用 go-zero 的 pathvar.Vars 而非标准库的 r.PathValue：
// go-zero v1.10.3 的 rest 使用自己的路由树，路径参数写入的是
// 请求上下文中的私有键，标准库 API 取不到（会静默返回空串）。
//
// 转换失败返回校验错误而非 404：用户 ID 传了非数字是请求格式问题，
// 与「这个 ID 不存在」不同，混为一谈会让前端难以区分该如何提示。
func pathInt64(r *http.Request, name string) (int64, error) {
	raw := pathvar.Vars(r)[name]
	if raw == "" {
		return 0, binder.NewValidationError(name, "路径参数缺失")
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, binder.NewValidationError(name, "必须是合法的整数 ID")
	}
	if value <= 0 {
		return 0, binder.NewValidationError(name, "必须是正整数")
	}

	return value, nil
}

// decodeFieldsInto 把已读出的顶层字段重新编码后解码到目标结构体。
//
// 之所以要「重新编码」而不是复用原请求体：请求体是一次性的流，
// 上一步的 RawFields 已经把它读完了。改从 map 出发还有个额外好处——
// 到这一步时白名单外的字段已被拒，因此不会出现未知字段错误。
func decodeFieldsInto(fields map[string]any, dst any) error {
	body, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("重新编码请求体失败: %w", err)
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return translateUnmarshalError(err)
	}

	return nil
}

// translateUnmarshalError 把 json 解码错误转成字段级校验错误。
func translateUnmarshalError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return binder.NewValidationError(typeErr.Field, "字段类型不合法")
	}

	return &binder.ValidationError{Reason: "请求体解析失败"}
}
