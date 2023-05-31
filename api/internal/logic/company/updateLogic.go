package company

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (l *UpdateLogic) Update(req *types.CompanyRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.更新企业信息
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.svcCtx.Config.Ids.Company))
	if err != nil {
		fmt.Printf("[Error]企业id[%s]转换：%s\n", l.svcCtx.Config.Ids.Company, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	update := bson.M{
		"$set": bson.M{
			"_id":                              id,
			"name":                             req.Name,
			"address":                          req.Address,
			"contact":                          req.Contact,
			"legal_representative":             req.LegalRepresentative,
			"unified_social_credit_identifier": req.UnifiedSocialCreditIdentifier,
			"email":                            req.Email,
			"site":                             req.Site,
			"introduction":                     req.Introduction,
			"business_scope":                   req.Name,
			"updated_at":                       time.Now().Unix(),
		},
	}

	//更新时，不存在就插入
	opts := options.Update().SetUpsert(true)

	_, err = l.svcCtx.CompanyModel.UpdateByID(l.ctx, id, &update, opts)
	if err != nil {
		fmt.Printf("[Error]修改企业信息:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.企业信息同步到顶级部门(企业)
	departmentId, err := primitive.ObjectIDFromHex(strings.TrimSpace(l.svcCtx.Config.Ids.Department))
	if err != nil {
		fmt.Printf("[Error]顶级部门(企业)id[%s]转换：%s\n", l.svcCtx.Config.Ids.Department, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	update = bson.M{
		"$set": bson.M{
			"_id":        departmentId,
			"name":       req.Name,
			"type":       80,
			"sort_id":    0,
			"parent_id":  "",
			"updated_at": time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.DepartmentModel.UpdateByID(l.ctx, departmentId, &update, opts)
	if err != nil {
		fmt.Printf("[Error]修改顶级部门(企业)信息:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
