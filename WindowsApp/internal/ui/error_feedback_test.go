package ui

import (
	"errors"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestRequestFailureTextDistinguishesPermissionAndBusinessErrors(t *testing.T) {
	if got := requestFailureText(&api.BusinessError{Code: 403, Msg: "无权操作"}); got != "权限不足：无权操作" {
		t.Fatalf("permission text = %q", got)
	}
	if got := requestFailureText(&api.BusinessError{Code: 400, Msg: "状态错误"}); got != "业务校验未通过：状态错误" {
		t.Fatalf("business text = %q", got)
	}
	if got := requestFailureText(errors.New("broken")); got != "请求失败：broken" {
		t.Fatalf("fallback text = %q", got)
	}
}
