package supplier

import (
	"api/model"
	"api/pkg/code"
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

func (l *AddLogic) Add(req *types.SupplierRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)
	uObjectID, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]uid[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}
	//1.供应商是否存在
	var name = strings.TrimSpace(req.Name)
	//i 表示不区分大小写
	filter := bson.M{
		"$or": []bson.M{
			{"name": name},
			{"code": strings.TrimSpace(req.Code)},
			{"unified_social_credit_identifier": strings.TrimSpace(req.UnifiedSocialCreditIdentifier)},
		},
		"status": bson.M{"$ne": 100},
	}
	singleRes := l.svcCtx.SupplierModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.Supplier
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复供应商:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "供应商名称已占用"
		case one.Code == strings.TrimSpace(req.Code):
			resp.Msg = "供应商编号已占用"
		case one.UnifiedSocialCreditIdentifier == strings.TrimSpace(req.UnifiedSocialCreditIdentifier):
			resp.Msg = "供应商统一社会信用代码已占用"
		default:
			resp.Msg = "供应商未知问题导致无法注册，请与系统管理员联系"
		}
		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //供应商未占用
	default:
		fmt.Printf("[Error]查询重复供应商:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.添加供应商
	var supplier = model.Supplier{
		Type:                          req.Type,
		Level:                         req.Level,
		Name:                          strings.TrimSpace(req.Name),
		Code:                          strings.TrimSpace(req.Code),
		Image:                         strings.TrimSpace(req.Image),
		LegalRepresentative:           strings.TrimSpace(req.LegalRepresentative),
		UnifiedSocialCreditIdentifier: strings.TrimSpace(req.UnifiedSocialCreditIdentifier),
		Manager:                       strings.TrimSpace(req.Manager),
		Contact:                       strings.TrimSpace(req.Contact),
		Status:                        code.SupplierStatusCode("审核中"), //默认:审核中
		Email:                         req.Email,
		Address:                       strings.TrimSpace(req.Address),
		Remark:                        strings.TrimSpace(req.Remark),
		Creator:                       uObjectID,
		CreatedAt:                     time.Now().Unix(),
		UpdatedAt:                     time.Now().Unix(),
	}
	_, err = l.svcCtx.SupplierModel.InsertOne(l.ctx, &supplier)
	if err != nil {
		fmt.Printf("[Error]供应商[%s]入库:%s\n", req.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
