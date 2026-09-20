package handler

import (
	"net/http"

	"bilibili-web/server/app/user/api/internal/logic"
	"bilibili-web/server/app/user/api/internal/svc"
	"bilibili-web/server/common/response"
)

func HealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.WriteOK(w, logic.Health(svcCtx))
	}
}
