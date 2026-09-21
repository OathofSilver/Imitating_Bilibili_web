package token

import "errors"

// ErrInvalidToken 表示凭证缺失、格式非法、签名不匹配或已过期。
//
// 刻意只暴露这一个哨兵错误：按 identity/auth「凭证过期统一处理」，
// 调用方对上述所有情形都应当返回同一个未认证错误码，不得靠错误细节区分，
// 否则会把「签名错」与「已过期」的区别泄露给攻击者。
var ErrInvalidToken = errors.New("token: 凭证无效")
