package model_mysql

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ProductPriceModel = (*customProductPriceModel)(nil)

type (
	// ProductPriceModel is an interface to be customized, add more methods here,
	// and implement the added methods in customProductPriceModel.
	ProductPriceModel interface {
		productPriceModel
	}

	customProductPriceModel struct {
		*defaultProductPriceModel
	}
)

// NewProductPriceModel returns a model for the database table.
func NewProductPriceModel(conn sqlx.SqlConn, c cache.CacheConf) ProductPriceModel {
	return &customProductPriceModel{
		defaultProductPriceModel: newProductPriceModel(conn, c),
	}
}
