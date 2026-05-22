package transaction

import (
	customerfinance "api/internal/logic/customer/finance"
	"api/model"
	financeCode "api/pkg/code"
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
		fmt.Printf("[Error]客户id[%s]格式错误：%s\n", req.CustomerId, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	var filter = bson.M{"_id": customerId, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}}
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

	typeInput := req.TransactionType
	if strings.TrimSpace(typeInput) == "" {
		typeInput = req.Type
	}
	transactionType, ok := financeCode.ManualTransactionTypeCode(typeInput)
	if !ok {
		resp.Code = http.StatusBadRequest
		resp.Msg = "交易类型错误"
		return resp, nil
	}
	if transactionType == financeCode.TransactionTypeManualAdjustment {
		resp.Code = http.StatusBadRequest
		resp.Msg = "手工调整需走审批流程，不能直接入账"
		return resp, nil
	}

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]添加客户[%s]交易记录：创建事务失败:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]添加客户[%s]交易记录：开启事务失败:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = financeCode.IdempotencyKey(financeCode.SourceTypeManualAdjustment, primitive.NewObjectID().Hex())
	}

	_, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
		TransactionType: transactionType,
		Status:          financeCode.TransactionStatusConfirmed,
		Code:            fmt.Sprintf("CT-%s-%d", time.Now().Format("20060102-15-04-05"), time.Now().UnixMilli()%1000),
		SourceType:      financeCode.SourceTypeManualAdjustment,
		SourceId:        idempotencyKey,
		IdempotencyKey:  idempotencyKey,
		CustomerId:      customer.Id.Hex(),
		CustomerName:    customer.Name,
		Amount:          req.Amount,
		Annex:           strings.Join(req.Annex, ","),
		Remark:          strings.TrimSpace(req.Remark),
		Time:            req.Time,
		Creator:         l.ctx.Value("uid").(string),
		CreatorName:     l.ctx.Value("name").(string),
	})
	if err != nil {
		fmt.Printf("[Error]添加客户[%s]交易记录:%s\n", customer.Name, err.Error())
		_ = session.AbortTransaction(dbCtx)
		if msg, isBusiness := customerfinance.IsBusinessError(err); isBusiness {
			resp.Code = http.StatusBadRequest
			resp.Msg = msg
			return resp, nil
		}
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		fmt.Printf("[Error]添加客户[%s]交易记录：提交事务失败:%s\n", customer.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
