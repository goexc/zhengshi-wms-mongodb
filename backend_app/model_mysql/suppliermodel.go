package model_mysql

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SupplierModel = (*customSupplierModel)(nil)

type (
	// SupplierModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSupplierModel.
	SupplierModel interface {
		supplierModel
	}

	customSupplierModel struct {
		*defaultSupplierModel
	}
)

// NewSupplierModel returns a model for the database table.
func NewSupplierModel(conn sqlx.SqlConn, c cache.CacheConf) SupplierModel {
	return &customSupplierModel{
		defaultSupplierModel: newSupplierModel(conn, c),
	}
}
