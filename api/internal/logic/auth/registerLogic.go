package auth

import (
	"api/model"
	"api/pkg/cryptx"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	resp = new(types.RegisterResponse)

	//1.是否已注册根账号
	userId, err := primitive.ObjectIDFromHex(l.svcCtx.Config.Ids.User)
	if err != nil {
		fmt.Printf("[Error]解析根账号id：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	count, err := l.svcCtx.UserModel.CountDocuments(l.ctx, bson.M{"_id": userId})
	if err != nil {
		fmt.Printf("[Error]查询根账号数量：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "系统已注册根账号，添加子账号请与系统管理员联系"
		return resp, nil
	}

	//2.添加企业信息
	companyId, err := primitive.ObjectIDFromHex(l.svcCtx.Config.Ids.Company)
	if err != nil {
		fmt.Printf("[Error]解析企业id：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.1 删除所有企业、部门信息
	_, err = l.svcCtx.DepartmentModel.DeleteMany(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]删除遗留企业、部门：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.2 企业未注册
	var company model.Department
	company = model.Department{
		Id:        companyId,
		Type:      80,
		SortId:    0,
		ParentId:  "",
		Name:      req.Company,
		FullName:  req.Company,
		Code:      "",
		Remark:    "",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_, err = l.svcCtx.DepartmentModel.InsertOne(l.ctx, &company)
	if err != nil {
		fmt.Printf("[Error]注册企业信息:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.添加系统管理角色
	roleId, err := primitive.ObjectIDFromHex(l.svcCtx.Config.Ids.Role)
	if err != nil {
		fmt.Printf("[Error]解析系统管理角色id：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.1 删除所有遗留角色
	_, err = l.svcCtx.RoleModel.DeleteMany(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]删除所有遗留角色：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.2 系统管理角色未注册
	var role = model.Role{
		Id:        roleId,
		Name:      "系统管理",
		ParentId:  "",
		SortId:    0,
		Status:    20,
		Remark:    "拥有系统所有权限",
		CreatedBy: "",
		CreatedAt: time.Now().Unix(),
		UpdatedAt: time.Now().Unix(),
	}
	_, err = l.svcCtx.RoleModel.InsertOne(l.ctx, &role)
	if err != nil {
		fmt.Printf("[Error]注册系统管理角色:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.删除角色菜单关联数据
	_, err = l.svcCtx.RoleMenuModel.DeleteMany(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]删除角色菜单关联数据:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.添加根账号
	//5.1 删除所有账号
	_, err = l.svcCtx.UserModel.DeleteMany(l.ctx, bson.M{})
	if err != nil {
		fmt.Printf("[Error]删除所有账号：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//5.2 添加根账号
	var user = model.User{
		Id:             userId,
		Account:        req.Account,
		Password:       cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password),
		Mobile:         req.Mobile,
		Email:          req.Email,
		Avatar:         l.svcCtx.Config.Avatar,
		Sex:            "",
		DepartmentId:   l.svcCtx.Config.Ids.Company,
		DepartmentName: req.Company,
		Status:         20,
		Remark:         "",
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
	}
	_, err = l.svcCtx.UserModel.InsertOne(l.ctx, &user)
	if err != nil {
		fmt.Printf("[Error]注册根账号:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
