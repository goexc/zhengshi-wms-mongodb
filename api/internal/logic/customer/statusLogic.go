package customer

import (
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"math"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatusLogic {
	return &StatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StatusLogic) Status(req *types.CustomerStatusRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]客户[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户参数错误"
		return resp, nil
	}

	if req.Status != "删除" {
		update := bson.M{
			"$set": bson.M{
				"status":      req.Status,
				"status_code": financeCode.CustomerStatusCodeFromLabel(req.Status),
				"is_deleted":  false,
			},
		}
		updateRes, e := l.svcCtx.CustomerModel.UpdateOne(l.ctx, bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}}, &update)
		if e != nil {
			fmt.Printf("[Error]修改客户[%s]状态：%s\n", req.Id, e.Error())
			resp.Msg = "服务器内部错误"
			resp.Code = http.StatusInternalServerError
			return resp, nil
		}
		if updateRes.MatchedCount != 1 {
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户不存在"
			return resp, nil
		}

		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]删除客户[%s]创建事务:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]删除客户[%s]开启事务:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	filter := bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}}
	var customer model.Customer
	singleRes := l.svcCtx.CustomerModel.FindOne(dbCtx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&customer); err != nil {
			_ = session.AbortTransaction(dbCtx)
			fmt.Printf("[Error]解析客户[%s]:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	case mongo.ErrNoDocuments:
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	default:
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.Id, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	{
		if math.Abs(customer.ReceivableBalance) > 0.000001 || math.Abs(customer.CreditBalance) > 0.000001 {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户存在应收账款或贷项余额，不能删除"
			return resp, nil
		}

		count, e := l.svcCtx.CustomerTransactionModel.CountDocuments(dbCtx, bson.M{"customer_id": customer.Id.Hex()})
		if e != nil {
			_ = session.AbortTransaction(dbCtx)
			fmt.Printf("[Error]查询客户[%s]交易流水:%s\n", req.Id, e.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		if count > 0 {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户存在交易流水，不能删除"
			return resp, nil
		}
	}

	update := bson.M{
		"$set": bson.M{
			"status":      req.Status,
			"status_code": financeCode.CustomerStatusCodeFromLabel(req.Status),
			"is_deleted":  true,
		},
	}
	updateFilter := bson.M{
		"_id":        id,
		"is_deleted": bson.M{"$ne": true},
		"status":     bson.M{"$ne": "删除"},
		"$and": []bson.M{
			{"$or": []bson.M{
				{"receivable_balance": bson.M{"$gte": -0.000001, "$lte": 0.000001}},
				{"receivable_balance": bson.M{"$exists": false}},
			}},
			{"$or": []bson.M{
				{"credit_balance": bson.M{"$gte": -0.000001, "$lte": 0.000001}},
				{"credit_balance": bson.M{"$exists": false}},
			}},
		},
	}
	updateRes, err := l.svcCtx.CustomerModel.UpdateOne(dbCtx, updateFilter, &update)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]修改客户[%s]状态：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	if updateRes.MatchedCount != 1 {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户状态或余额已变化，请刷新后重试"
		return resp, nil
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		fmt.Printf("[Error]删除客户[%s]提交事务:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
