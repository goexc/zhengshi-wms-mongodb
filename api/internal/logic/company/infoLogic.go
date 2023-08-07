package company

import (
	"api/model"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InfoLogic {
	return &InfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InfoLogic) Info() (resp *types.CompanyResponse, err error) {
	resp = new(types.CompanyResponse)

	//1.获取企业信息
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.svcCtx.Config.Ids.Company))
	if err != nil {
		fmt.Printf("[Error]企业id[%s]转换：%s\n", l.svcCtx.Config.Ids.Company, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	var company model.Company
	res := l.svcCtx.CompanyModel.FindOne(l.ctx, filter)
	switch res.Err() {
	case nil:
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "企业信息不存在"
		return resp, nil
	default:
		fmt.Printf("[Error]查询企业[%s]信息：%s\n", l.svcCtx.Config.Ids.Company, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if err = res.Decode(&company); err != nil {
		fmt.Printf("[Error]解析企业[%s]信息：%s\n", l.svcCtx.Config.Ids.Company, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data = types.Company{
		Name:                          company.Name,
		Address:                       company.Address,
		Contact:                       company.Contact,
		LegalRepresentative:           company.LegalRepresentative,
		UnifiedSocialCreditIdentifier: company.UnifiedSocialCreditIdentifier,
		Email:                         company.Email,
		Site:                          company.Site,
		Introduction:                  company.Introduction,
		BusinessScope:                 company.BusinessScope,
		CreatedAt:                     company.CreatedAt,
		UpdatedAt:                     company.UpdatedAt,
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
