package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
	"zhengshi-wms-windowsapp/internal/securestore"
)

var ErrNoCachedSession = errors.New("没有缓存登录状态")

type RestoreNetworkError struct {
	Err error
}

func (e *RestoreNetworkError) Error() string {
	return "无法在线验证缓存登录状态：" + e.Err.Error()
}

func (e *RestoreNetworkError) Unwrap() error { return e.Err }

func RestoreSession(cfg config.Config) (*Session, error) {
	cached, err := securestore.Load()
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoCachedSession
	}
	if err != nil {
		_ = securestore.Delete()
		return nil, fmt.Errorf("缓存登录状态无法解密：%w", err)
	}
	if cached.APIBaseURL != cfg.APIBaseURL {
		_ = securestore.Delete()
		return nil, errors.New("服务器环境已变化，请重新登录")
	}
	if cached.ExpiresAt <= time.Now().Add(time.Minute).Unix() {
		_ = securestore.Delete()
		return nil, errors.New("登录状态已过期，请重新登录")
	}

	client := api.NewClient(cfg.APIBaseURL)
	client.SetToken(cached.Token)
	profile, err := client.Profile(context.Background())
	if err != nil {
		return nil, classifyRestoreError(err)
	}
	perms, err := client.Permissions(context.Background())
	if err != nil {
		return nil, classifyRestoreError(err)
	}
	return &Session{
		Client: client,
		Login: api.LoginData{
			Name: profile.Name, Mobile: cached.Mobile,
			DepartmentName: profile.DepartmentName,
			Token:          cached.Token, Exp: cached.ExpiresAt,
		},
		Profile: profile,
		Perms:   perms,
	}, nil
}

func classifyRestoreError(err error) error {
	var businessErr *api.BusinessError
	if errors.As(err, &businessErr) {
		switch businessErr.Code {
		case 400, 401, 403:
			_ = securestore.Delete()
			return fmt.Errorf("登录状态已失效：%w", err)
		}
	}
	return &RestoreNetworkError{Err: err}
}
