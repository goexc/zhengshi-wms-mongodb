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
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEditProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditProfileLogic {
	return &EditProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditProfileLogic) EditProfile(req *types.ProfileRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	uid := l.ctx.Value("uid").(string)

	// 1.个人是否存在
	id, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		fmt.Printf("[Error]个人[%s]id转换：%s\n", uid, err.Error())
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
		fmt.Printf("[Error]查询个人[%s]:%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//2.账号名称、手机号码、Email是否重复
	var or = []bson.M{
		{"name": strings.TrimSpace(req.Name)},
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
			fmt.Printf("[Error]解析重复个人:%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		switch true {
		case one.Name == strings.TrimSpace(req.Name):
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
		fmt.Printf("[Error]查询重复个人:%s\n", singleRes.Err().Error())
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
	switch singleRes.Err() {
	case nil: //存在
	case mongo.ErrNoDocuments: //部门不存在
		fmt.Printf("[Error]部门[%s]不存在\n", req.DepartmentId)
		resp.Code = http.StatusBadRequest
		resp.Msg = "部门不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询部门[%s]是否存在:%s\n", req.DepartmentId, singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var department model.Department
	if err = singleRes.Decode(&department); err != nil {
		fmt.Printf("[Error]部门[%s]数据解析\n", req.DepartmentId)
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.角色是否存在
	var rolesId bson.A
	for _, one := range req.RolesId {
		roleId, e := primitive.ObjectIDFromHex(strings.TrimSpace(one))
		if e != nil {
			fmt.Printf("[Error]解析角色id[%s]：%s\n", one, e.Error())
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

	_, err = l.svcCtx.Enforcer.DeleteRolesForUser(fmt.Sprintf("user_%s", uid))
	if err != nil {
		fmt.Printf("[Error]个人[%s]清空角色:%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	if len(roles) > 0 {
		_, err = l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", uid), roles)
		if err != nil {
			fmt.Printf("[Error]个人[%s]分配角色:%s\n", uid, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	//6.更新：密码单独修改
	update := bson.M{
		"$set": bson.M{
			"name":            strings.TrimSpace(req.Name),
			"mobile":          req.Mobile,
			"email":           req.Email,
			"avatar":          l.svcCtx.Config.Avatar,
			"sex":             req.Sex,
			"department_id":   strings.TrimSpace(req.DepartmentId),
			"department_name": department.Name,
			"updated_at":      time.Now().Unix(),
		},
	}

	_, err = l.svcCtx.UserModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]更新个人[%s]信息：%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
