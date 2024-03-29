package account

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

type ProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProfileLogic {
	return &ProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProfileLogic) Profile() (resp *types.ProfileResponse, err error) {
	resp = new(types.ProfileResponse)

	uid := l.ctx.Value("uid").(string)

	// 1.个人是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(uid))
	if err != nil {
		fmt.Printf("[Error]角色[%s]id转换：%s\n", uid, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //个人存在
	case mongo.ErrNoDocuments: //个人不存在
		fmt.Printf("[Error]个人[%s]不存在\n", uid)
		resp.Code = http.StatusBadRequest
		resp.Msg = "个人不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询个人[%s]是否存在:%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var profile model.User
	if err = singleRes.Decode(&profile); err != nil {
		fmt.Printf("[Error]解析个人信息:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", uid), []string{"6472101cbf3beb2563ff8fa0"})

	//2.查询个人角色列表
	var casbinRoles []string
	casbinRoles, err = l.svcCtx.Enforcer.GetRolesForUser(fmt.Sprintf("user_%s", profile.Id.Hex()))

	fmt.Println("用户id：", profile.Id.Hex())
	fmt.Println("角色：", casbinRoles)

	//5.汇总
	//resp.Data.Routes = rootRoute
	resp.Data.Name = profile.Name
	resp.Data.Sex = profile.Sex
	resp.Data.DepartmentId = profile.DepartmentId
	resp.Data.DepartmentName = profile.DepartmentName
	resp.Data.Avatar = profile.Avatar
	resp.Data.Mobile = profile.Mobile
	resp.Data.Email = profile.Email
	resp.Data.Status = profile.Status
	resp.Data.Remark = profile.Remark
	resp.Data.CreatedAt = profile.CreatedAt
	resp.Data.UpdatedAt = profile.UpdatedAt

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
