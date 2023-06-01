package customer

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

type UpdateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLogic) Update(req *types.CustomerRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//todo:用于操作记录
	//uid := l.ctx.Value("uid").(string)
	//uObjectID, err := primitive.ObjectIDFromHex(uid)
	//if err != nil {
	//	fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
	//	resp.Code = http.StatusBadRequest
	//	resp.Msg = "参数错误"
	//	return resp, nil
	//}
	//1.客户是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]客户[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户参数错误"
		return resp, nil
	}
	//排除已删除的客户
	filter := bson.M{
		"_id":    id,
		"status": bson.M{"$ne": 100},
	}
	count, err := l.svcCtx.CustomerModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询客户[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	//2.客户名称是否重复
	filter = bson.M{
		"_id":    bson.M{"$ne": id},
		"status": bson.M{"$ne": 100},
		"$or": []bson.M{
			{"name": strings.TrimSpace(req.Name)},
			{"code": strings.TrimSpace(req.Code)},
			{"unified_social_credit_identifier": strings.TrimSpace(req.UnifiedSocialCreditIdentifier)},
		},
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
			resp.Msg = "客户未知问题无法注册"
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

	//3.更新客户信息
	var update = bson.M{
		"$set": bson.M{
			"type":                             req.Type,
			"name":                             strings.TrimSpace(req.Name),
			"code":                             strings.TrimSpace(req.Code),
			"legal_representative":             strings.TrimSpace(req.LegalRepresentative),
			"unified_social_credit_identifier": strings.TrimSpace(req.UnifiedSocialCreditIdentifier),
			"address":                          req.Address,
			"contact":                          req.Contact,
			"manager":                          req.Manager,
			"level":                            req.Level,
			"email":                            req.Email,
			"remark":                           req.Remark,
			"updated_at":                       time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.CustomerModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新客户[%s]信息：%s\n", req.Id, err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
