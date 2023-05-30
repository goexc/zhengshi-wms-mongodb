package user

import (
	"api/model"
	"api/pkg/cryptx"
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

func (l *UpdateLogic) Update(req *types.UserRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	if strings.TrimSpace(req.Id) == "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请选择用户"
		return resp, nil
	}

	// 1.用户是否存在
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.Id))
	if err != nil {
		fmt.Printf("[Error]角色[%s]id转换：%s\n", req.Id, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "参数错误"
		return resp, nil
	}

	var filter = bson.M{"_id": id}
	singleRes := l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil: //用户存在
	case mongo.ErrNoDocuments: //用户不存在
		fmt.Printf("[Error]用户[%s]不存在\n", req.Id)
		resp.Code = http.StatusBadRequest
		resp.Msg = "用户不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询用户[%s]是否存在:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//2.账号名称、手机号码、Email是否重复
	var or = []bson.M{
		{"name": strings.TrimSpace(req.Account)},
		{"mobile": strings.TrimSpace(req.Mobile)},
	}
	if strings.TrimSpace(req.Email) != "" {
		or = append(or, bson.M{"email": req.Email})
	}

	filter = bson.M{
		"_id": bson.M{
			"$ne": id,
		},
		"$or": or,
	}

	singleRes = l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch singleRes.Err() {
	case nil:
		var one model.User
		if err = singleRes.Decode(&one); err != nil {
			fmt.Printf("[Error]解析重复用户:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Account == strings.TrimSpace(req.Account):
			resp.Msg = "账号名称重复"
		case one.Mobile == strings.TrimSpace(req.Mobile):
			resp.Msg = "手机号码重复"
		case one.Email != "" && one.Email == strings.TrimSpace(req.Email):
			resp.Msg = "Email重复"
		}

		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //账号名称、手机号码、Email未占用
	default:
		fmt.Printf("[Error]查询重复用户:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.部门是否存在
	departmentId, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.DepartmentId))
	if err != nil {
		fmt.Printf("[Error]解析部门id[%s]：%s\n", req.DepartmentId, err.Error())
		resp.Msg = "部门参数错误"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	filter = bson.M{"_id": departmentId}
	singleRes = l.svcCtx.DepartmentModel.FindOne(l.ctx, filter)
	switch err {
	case nil: //存在
	case mongo.ErrNoDocuments: //部门不存在
		fmt.Printf("[Error]部门[%s]不存在\n", req.DepartmentId)
		resp.Code = http.StatusBadRequest
		resp.Msg = "部门不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询部门[%s]是否存在:%s\n", req.DepartmentId, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//4.角色是否存在
	var rolesId bson.A
	for _, one := range req.RolesId {
		roleId, e := primitive.ObjectIDFromHex(strings.TrimSpace(one))
		if e != nil {
			fmt.Printf("[Error]解析角色id[%s]：%s\n", req.DepartmentId, e.Error())
			resp.Msg = "角色参数错误"
			resp.Code = http.StatusBadRequest
			return resp, nil
		}
		rolesId = append(rolesId, roleId)
	}

	filter = bson.M{"_id": bson.M{"$in": rolesId}}
	count, err := l.svcCtx.RoleModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询角色列表是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count != int64(len(req.RolesId)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分角色不存在"
		return resp, nil
	}

	//解绑旧角色,绑定新角色
	var roles []string
	for _, roleId := range req.RolesId {
		roles = append(roles, fmt.Sprintf("role_%s", roleId))
	}

	_, err = l.svcCtx.Enforcer.DeleteRolesForUser(fmt.Sprintf("user_%s", req.Id))
	if err != nil {
		fmt.Printf("[Error]用户[%s]清空角色:%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if len(roles) > 0 {
		_, err = l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", req.Id), roles)
		if err != nil {
			fmt.Printf("[Error]用户[%s]分配角色:%s\n", req.Id, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	//6.更新
	update := bson.M{
		"$set": bson.M{
			"account":    strings.TrimSpace(req.Account),
			"password":   cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password),
			"mobile":     req.Mobile,
			"email":      req.Email,
			"avatar":     l.svcCtx.Config.Avatar,
			"sex":        req.Sex,
			"status":     req.Status,
			"updated_at": time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.UserModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新用户[%s]信息：%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
