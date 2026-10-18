package svc

import (
	"api/model"
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureFinanceCatalog adds only missing catalog entries. It does not assign
// roles, replace administrator customizations, or modify existing business data.
func EnsureFinanceCatalog(s *ServiceContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	root := ""
	var rootMenu model.Menu
	err := s.MenuModel.FindOne(ctx, bson.M{"type": 1, "parent_id": "", "path": "", "hidden": true}).Decode(&rootMenu)
	if err == nil {
		root = rootMenu.Id.Hex()
	} else if err != mongo.ErrNoDocuments {
		return err
	}
	parent, err := ensureFinanceMenu(ctx, s, model.Menu{Type: 1, Name: "Finance", Title: "客户财务", Path: "/finance", ParentId: root, SortId: 60, Perms: "finance", Icon: "Wallet"})
	if err != nil {
		return err
	}
	for _, module := range []struct {
		key, label, icon string
		actions          []string
	}{
		{"acceptance", "客户承兑", "Tickets", []string{"list", "add", "edit", "redeem", "reverse", "void"}},
		{"invoice", "销售开票记录", "Document", []string{"list", "add", "edit", "void"}},
	} {
		id, e := ensureFinanceMenu(ctx, s, model.Menu{Type: 1, Name: "Finance_" + module.key, Title: module.label, Path: "/finance/" + module.key, ParentId: parent, SortId: 1, Component: "/finance/" + module.key + "/index", Perms: "finance:" + module.key, Icon: module.icon})
		if e != nil {
			return e
		}
		labels := map[string]string{"list": "查看", "add": "新增", "edit": "编辑", "redeem": "确认到账", "reverse": "撤销到账", "void": "作废登记"}
		for i, action := range module.actions {
			_, e = ensureFinanceMenu(ctx, s, model.Menu{Type: 2, Name: labels[action] + module.label, Title: labels[action] + module.label, ParentId: id, SortId: int64(i), Perms: "finance:" + module.key + ":" + action})
			if e != nil {
				return e
			}
		}
	}
	for _, a := range financeAPIEntries() {
		a.Id = financeCatalogID("api:" + a.Method + ":" + a.Uri)
		a.Type = 2
		a.CreatedAt = time.Now().Unix()
		_, err = s.ApiModel.UpdateOne(ctx, bson.M{"uri": a.Uri, "method": a.Method}, bson.M{"$setOnInsert": a}, options.Update().SetUpsert(true))
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("register finance API: %w", err)
		}
	}
	return nil
}

func financeCatalogID(key string) primitive.ObjectID {
	sum := sha256.Sum256([]byte("zhengshi:customer-documents:v1:" + key))
	var id primitive.ObjectID
	copy(id[:], sum[:12])
	return id
}

func ensureFinanceMenu(ctx context.Context, s *ServiceContext, menu model.Menu) (string, error) {
	menu.Id = financeCatalogID("menu:" + menu.Perms)
	menu.CreatedAt = time.Now().Unix()
	filter := bson.M{"type": menu.Type, "perms": menu.Perms}
	if menu.Type == 1 {
		filter = bson.M{"type": 1, "path": menu.Path}
	}
	_, err := s.MenuModel.UpdateOne(ctx, filter, bson.M{"$setOnInsert": menu}, options.Update().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return "", err
	}
	var current model.Menu
	if err = s.MenuModel.FindOne(ctx, filter).Decode(&current); err != nil {
		return "", err
	}
	return current.Id.Hex(), nil
}

func financeAPIEntries() []model.Api {
	return []model.Api{
		{Name: "财务客户选择查询", Uri: "/customer", Method: "GET"},
		{Name: "财务客户列表查询", Uri: "/customer/list", Method: "GET"},
		{Name: "客户承兑查询", Uri: "/customer/acceptance", Method: "GET"},
		{Name: "客户承兑详情", Uri: "/customer/acceptance/detail", Method: "GET"},
		{Name: "客户承兑新增", Uri: "/customer/acceptance", Method: "POST"},
		{Name: "客户承兑修改", Uri: "/customer/acceptance", Method: "PUT"},
		{Name: "客户承兑到账", Uri: "/customer/acceptance/redeem", Method: "POST"},
		{Name: "客户承兑撤销到账", Uri: "/customer/acceptance/reverse", Method: "POST"},
		{Name: "承兑关联回款查询", Uri: "/customer/acceptance/payments", Method: "GET"},
		{Name: "客户承兑作废", Uri: "/customer/acceptance/void", Method: "POST"},
		{Name: "销售开票查询", Uri: "/customer/invoice", Method: "GET"},
		{Name: "销售开票详情", Uri: "/customer/invoice/detail", Method: "GET"},
		{Name: "销售开票新增", Uri: "/customer/invoice", Method: "POST"},
		{Name: "销售开票修改", Uri: "/customer/invoice", Method: "PUT"},
		{Name: "销售开票作废", Uri: "/customer/invoice/void", Method: "POST"},
		{Name: "附件上传", Uri: "/images", Method: "POST"},
		{Name: "附件素材查询", Uri: "/images", Method: "GET"},
	}
}
