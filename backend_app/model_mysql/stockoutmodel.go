package model_mysql

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ StockOutModel = (*customStockOutModel)(nil)

type (
	// StockOutModel is an interface to be customized, add more methods here,
	// and implement the added methods in customStockOutModel.
	StockOutModel interface {
		stockOutModel
	}

	customStockOutModel struct {
		*defaultStockOutModel
	}
)

// NewStockOutModel returns a model for the database table.
func NewStockOutModel(conn sqlx.SqlConn, c cache.CacheConf) StockOutModel {
	return &customStockOutModel{
		defaultStockOutModel: newStockOutModel(conn, c),
	}
}
