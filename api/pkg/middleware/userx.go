package middleware

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
)

type SetUser struct {
}

func NewSetUser() *SetUser {
	return &SetUser{}
}

// Handle api 应用中间件：写入UserId
func (m *SetUser) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-User")
		logx.Infof("微服务获得UserId:%s", userId)
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", userId)
		next(w, r.WithContext(ctx))
	}
}
