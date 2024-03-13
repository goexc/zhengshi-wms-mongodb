package transaction

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddLogic) Add(req *types.CustomerTransactionAddRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

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

	//2.添加客户交易记录
	var transaction = model.CustomerTransaction{
		Type:         req.Type,
		Code:         fmt.Sprintf("O-%s-%d", time.Now().Format("20060102-15-04-05"), time.Now().UnixMilli()%1000), //YYYY-MM-DD-HH-mm-ss-SSS
		OrderCode:    "",
		CustomerId:   customer.Id.Hex(),
		CustomerName: customer.Name,
		Amount:       req.Amount,
		Annex:        strings.Join(req.Annex, ","),
		Remark:       strings.TrimSpace(req.Remark),
		Time:         req.Time,
		Creator:      l.ctx.Value("uid").(string),
		CreatorName:  l.ctx.Value("name").(string),
		CreatedAt:    time.Now().Unix(),
	}
	_, err = l.svcCtx.CustomerTransactionModel.InsertOne(l.ctx, &transaction)
	if err != nil {
		fmt.Printf("[Error]添加客户[%s]交易记录:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.扣减客户应收款项
	var update = bson.M{
		"$inc": bson.M{
			"receivable_balance": req.Amount,
		},
	}

	_, err = l.svcCtx.CustomerModel.UpdateByID(l.ctx, customer.Id, &update)
	if err != nil {
		fmt.Printf("[Error]扣减客户[%s]应收款项：%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
