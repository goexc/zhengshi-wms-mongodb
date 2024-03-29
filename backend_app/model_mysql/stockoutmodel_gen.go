// Code generated by goctl. DO NOT EDIT.

package model_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/builder"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/stringx"
)

var (
	stockOutFieldNames          = builder.RawFieldNames(&StockOut{})
	stockOutRows                = strings.Join(stockOutFieldNames, ",")
	stockOutRowsExpectAutoSet   = strings.Join(stringx.Remove(stockOutFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), ",")
	stockOutRowsWithPlaceHolder = strings.Join(stringx.Remove(stockOutFieldNames, "`id`", "`create_at`", "`create_time`", "`created_at`", "`update_at`", "`update_time`", "`updated_at`"), "=?,") + "=?"

	cacheStockOutIdPrefix = "cache:stockOut:id:"
)

type (
	stockOutModel interface {
		Insert(ctx context.Context, data *StockOut) (sql.Result, error)
		FindOne(ctx context.Context, id int64) (*StockOut, error)
		FindByPage(ctx context.Context, query string) ([]StockOut, error)
		Count(ctx context.Context, query string) (int64, error)
		Update(ctx context.Context, data *StockOut) error
		Delete(ctx context.Context, id int64) error
	}

	defaultStockOutModel struct {
		sqlc.CachedConn
		table string
	}

	StockOut struct {
		Id         int64     `db:"id"`
		ClientId   int64     `db:"client_id"`   // 客户id
		ClientName string    `db:"client_name"` // 客户名称
		Numbering  string    `db:"numbering"`   // 编号
		HasTax     bool      `db:"has_tax"`     // 是否含税
		Tax        int64     `db:"tax"`         // 税率(%)
		Total      float64   `db:"total"`       // 总金额(元)
		Date       int64     `db:"date"`        // 出库日期
		CreatedAt  time.Time `db:"created_at"`
		UpdatedAt  time.Time `db:"updated_at"`
	}
)

func newStockOutModel(conn sqlx.SqlConn, c cache.CacheConf) *defaultStockOutModel {
	return &defaultStockOutModel{
		CachedConn: sqlc.NewConn(conn, c),
		table:      "`stock_out`",
	}
}

func (m *defaultStockOutModel) Delete(ctx context.Context, id int64) error {
	stockOutIdKey := fmt.Sprintf("%s%v", cacheStockOutIdPrefix, id)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("delete from %s where `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, id)
	}, stockOutIdKey)
	return err
}

func (m *defaultStockOutModel) FindOne(ctx context.Context, id int64) (*StockOut, error) {
	stockOutIdKey := fmt.Sprintf("%s%v", cacheStockOutIdPrefix, id)
	var resp StockOut
	err := m.QueryRowCtx(ctx, &resp, stockOutIdKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
		query := fmt.Sprintf("select %s from %s where `id` = ? limit 1", stockOutRows, m.table)
		return conn.QueryRowCtx(ctx, v, query, id)
	})
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultStockOutModel) FindByPage(ctx context.Context, query string) ([]StockOut, error) {
	query = fmt.Sprintf("select %s from %s", stockOutRows, m.table) + query
	var resp []StockOut
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &resp, query)
	switch err {
	case nil:
		return resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

func (m *defaultStockOutModel) Count(ctx context.Context, query string) (int64, error) {
	query = fmt.Sprintf("select %s from %s", stockOutRows, m.table) + query
	var resp []StockOut
	err := m.CachedConn.QueryRowsNoCacheCtx(ctx, &resp, query)
	switch err {
	case nil:
		return int64(len(resp)), nil
	case sqlc.ErrNotFound:
		return 0, ErrNotFound
	default:
		return 0, err
	}
}

func (m *defaultStockOutModel) Insert(ctx context.Context, data *StockOut) (sql.Result, error) {
	stockOutIdKey := fmt.Sprintf("%s%v", cacheStockOutIdPrefix, data.Id)
	ret, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?, ?, ?)", m.table, stockOutRowsExpectAutoSet)
		return conn.ExecCtx(ctx, query, data.ClientId, data.ClientName, data.Numbering, data.HasTax, data.Tax, data.Total, data.Date)
	}, stockOutIdKey)
	return ret, err
}

func (m *defaultStockOutModel) Update(ctx context.Context, data *StockOut) error {
	stockOutIdKey := fmt.Sprintf("%s%v", cacheStockOutIdPrefix, data.Id)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("update %s set %s where `id` = ?", m.table, stockOutRowsWithPlaceHolder)
		return conn.ExecCtx(ctx, query, data.ClientId, data.ClientName, data.Numbering, data.HasTax, data.Tax, data.Total, data.Date, data.Id)
	}, stockOutIdKey)
	return err
}

func (m *defaultStockOutModel) formatPrimary(primary interface{}) string {
	return fmt.Sprintf("%s%v", cacheStockOutIdPrefix, primary)
}

func (m *defaultStockOutModel) queryPrimary(ctx context.Context, conn sqlx.SqlConn, v, primary interface{}) error {
	query := fmt.Sprintf("select %s from %s where `id` = ? limit 1", stockOutRows, m.table)
	return conn.QueryRowCtx(ctx, v, query, primary)
}

func (m *defaultStockOutModel) tableName() string {
	return m.table
}
