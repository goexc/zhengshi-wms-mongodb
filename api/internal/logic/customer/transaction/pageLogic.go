package transaction

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
)

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.CustomerTransactionPageRequest) (resp *types.CustomerTransactionsResponse, err error) {
	resp = new(types.CustomerTransactionsResponse)

	//1.客户是否存在
	var customer model.Customer

	customerId, err := primitive.ObjectIDFromHex(req.CustomerId)
	if err != nil {
		fmt.Printf("[Error]客户id[%s]格式错误：%s\n", req.CustomerId)
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	var filter = bson.M{"_id": customerId, "status": bson.M{"$ne": "删除"}}
	singleRes := l.svcCtx.CustomerModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		if err = singleRes.Decode(&customer); err != nil {
			fmt.Printf("[Error]解析客户信息:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

	case mongo.ErrNoDocuments: //客户不存在
		fmt.Printf("[Error]客户[%s]不存在\n", req.CustomerId)
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.CustomerId, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//2.查询客户交易流水分页
	filter = bson.M{"customer_id": req.CustomerId}
	var option = options.Find().SetSort(bson.M{"time": -1}).SetSkip((req.Page - 1) * req.Size).SetLimit(req.Size)
	cur, err := l.svcCtx.CustomerTransactionModel.Find(l.ctx, filter, option)
	if err != nil {
		fmt.Printf("[Error]查询客户交易流水分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var transactions []model.CustomerTransaction
	if err = cur.All(l.ctx, &transactions); err != nil {
		fmt.Printf("[Error]解析客户交易流水分页:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	for _, one := range transactions {
		resp.Data.List = append(resp.Data.List, types.CustomerTransaction{
			Type:   one.Type,
			Time:   one.Time,
			Amount: one.Amount,
		})
	}

	//3.查询客户交易流水条数
	resp.Data.Total, err = l.svcCtx.CustomerTransactionModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询客户[%s]交易流水条数:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
