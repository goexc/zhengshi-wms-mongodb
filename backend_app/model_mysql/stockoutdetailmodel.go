package model_mysql

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ StockOutDetailModel = (*customStockOutDetailModel)(nil)

type (
	// StockOutDetailModel is an interface to be customized, add more methods here,
	// and implement the added methods in customStockOutDetailModel.
	StockOutDetailModel interface {
		stockOutDetailModel
	}

	customStockOutDetailModel struct {
		*defaultStockOutDetailModel
	}
)

// NewStockOutDetailModel returns a model for the database table.
func NewStockOutDetailModel(conn sqlx.SqlConn, c cache.CacheConf) StockOutDetailModel {
	return &customStockOutDetailModel{
		defaultStockOutDetailModel: newStockOutDetailModel(conn, c),
	}
}
