package images

import (
	"api/internal/svc"
	"context"
	"net/http"
)

func scopeAllowed(ctx context.Context, s *svc.ServiceContext, scope string, write bool) bool {
	if scope == "" {
		return true
	}
	if s.Enforcer == nil {
		return false
	}
	uid, _ := ctx.Value("uid").(string)
	if uid == "" {
		return false
	}
	subject := "user_" + uid
	roles, err := s.Enforcer.GetRolesForUser(subject)
	if err != nil {
		return false
	}
	for _, role := range roles {
		if role == "role_"+s.Config.Ids.Role {
			return true
		}
	}
	path := "/customer/acceptance"
	if scope == "invoice" {
		path = "/customer/invoice"
	}
	methods := []string{http.MethodGet}
	if write {
		methods = []string{http.MethodPost, http.MethodPut}
		if scope == "acceptance_redemption" {
			path += "/redeem"
			methods = []string{http.MethodPost}
		}
	}
	for _, method := range methods {
		if allowed, e := s.Enforcer.Enforce(subject, path, method); e == nil && allowed {
			return true
		}
	}
	return false
}
