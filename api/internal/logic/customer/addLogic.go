package customer

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

func (l *AddLogic) Add(req *types.CustomerRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}
	//1.客户是否存在
	var name = strings.TrimSpace(req.Name)
	//i 表示不区分大小写
	filter := bson.M{
		"$or": []bson.M{
			{"name": name},
			{"code": strings.TrimSpace(req.Code)},
			{"unified_social_credit_identifier": strings.TrimSpace(req.UnifiedSocialCreditIdentifier)},
		},
		"is_deleted": bson.M{"$ne": true},
		"status":     bson.M{"$ne": "删除"},
	}
	singleRes := l.svcCtx.CustomerModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Customer
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复客户:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "客户名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "客户编号已占用"
		case one.UnifiedSocialCreditIdentifier == strings.TrimSpace(req.UnifiedSocialCreditIdentifier):
			resp.Msg = "客户统一社会信用代码已占用"
		default:
			resp.Msg = "客户未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //客户未占用
	default:
		fmt.Printf("[Error]查询重复客户:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]客户[%s]入库：创建事务失败:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]客户[%s]入库：开启事务失败:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.添加客户
	var customer = model.Customer{
		Id:                            primitive.NewObjectID(),
		Type:                          req.Type,
		Code:                          strings.TrimSpace(req.Code),
		Image:                         strings.TrimSpace(req.Image),
		LegalRepresentative:           strings.TrimSpace(req.LegalRepresentative),
		UnifiedSocialCreditIdentifier: strings.TrimSpace(req.UnifiedSocialCreditIdentifier),
		Name:                          strings.TrimSpace(req.Name),
		NameFingerprint:               financeCode.CustomerNameFingerprint(req.Name),
		Address:                       strings.TrimSpace(req.Address),
		Contact:                       strings.TrimSpace(req.Contact),
		Manager:                       strings.TrimSpace(req.Manager),
		Status:                        "潜在", //默认:潜在
		StatusCode:                    "",
		IsDeleted:                     false,
		Email:                         req.Email,
		Remark:                        strings.TrimSpace(req.Remark),
		ReceivableBalance:             0,
		CreditBalance:                 0,
		Creator:                       uObjectID,
		CreatedAt:                     time.Now().Unix(),
		UpdatedAt:                     time.Now().Unix(),
	}
	_, err = l.svcCtx.CustomerModel.InsertOne(dbCtx, &customer)
	if err != nil {
		fmt.Printf("[Error]客户[%s]入库:%s\n", req.Name, err.Error())
		_ = session.AbortTransaction(dbCtx)
		if mongo.IsDuplicateKeyError(err) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "客户编号、名称或统一社会信用代码已占用"
			return resp, nil
		}
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if req.ReceivableBalance > 0 {
		now := time.Now().Unix()
		if _, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
			TransactionType: financeCode.TransactionTypeOpeningAR,
			Status:          financeCode.TransactionStatusConfirmed,
			Code:            fmt.Sprintf("CT-%s-%d", time.Now().Format("20060102-15-04-05"), time.Now().UnixMilli()%1000),
			OrderCode:       customer.Code,
			SourceType:      financeCode.SourceTypeOpening,
			SourceId:        customer.Id.Hex(),
			SourceCode:      customer.Code,
			IdempotencyKey:  financeCode.IdempotencyKey(financeCode.TransactionTypeOpeningAR, customer.Id.Hex()),
			CustomerId:      customer.Id.Hex(),
			CustomerName:    customer.Name,
			Amount:          req.ReceivableBalance,
			Remark:          "期初应收",
			Time:            now,
			Creator:         l.ctx.Value("uid").(string),
			CreatorName:     l.ctx.Value("name").(string),
			CreatedAt:       now,
		}); err != nil {
			fmt.Printf("[Error]客户[%s]期初应收流水入库:%s\n", req.Name, err.Error())
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
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		fmt.Printf("[Error]客户[%s]入库：提交事务失败:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
