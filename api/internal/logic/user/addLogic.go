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

func (l *AddLogic) Add(req *types.UserAddRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.用户名、手机号码、Email是否存在
	var or = []bson.M{
		{"name": strings.TrimSpace(req.Name)},
		{"mobile": strings.TrimSpace(req.Mobile)},
	}
	if strings.TrimSpace(req.Email) != "" {
		or = append(or, bson.M{"email": req.Email})
	}

	var filter = bson.M{
		"$or":    or,
		"status": bson.M{"$ne": "删除"},
	}

	singleRes := l.svcCtx.UserModel.FindOne(l.ctx, filter)
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
		case one.Name == strings.TrimSpace(req.Name):
			resp.Msg = "账号名称重复"
		case one.Mobile == strings.TrimSpace(req.Mobile):
			resp.Msg = "手机号码重复"
		case one.Email != "" && one.Email == strings.TrimSpace(req.Email):
			resp.Msg = "Email重复"
		}

		resp.Code = http.StatusBadRequest
		return resp, nil
	case mongo.ErrNoDocuments: //用户未占用
	default:
		fmt.Printf("[Error]查询重复用户:%s\n", singleRes.Err().Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//2.角色是否存在
	if len(req.RolesId) == 0 {
		resp.Msg = "请选择角色"
		resp.Code = http.StatusBadRequest
		return resp, nil
	}

	var rolesId = make([]primitive.ObjectID, 0)
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
		fmt.Printf("[Error]查询用户角色是否存在:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count != int64(len(req.RolesId)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分角色不存在"
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

	//4.添加用户
	insert := bson.D{
		{"name", strings.TrimSpace(req.Name)},
		{"password", cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password)},
		{"mobile", req.Mobile},
		{"email", req.Email},
		{"avatar", l.svcCtx.Config.Avatar},
		{"sex", req.Sex},
		{"status", "启用"},
		{"department_id", strings.TrimSpace(req.DepartmentId)},
		{"department_name", department.Name},
		{"remark", req.Remark},
		{"created_at", time.Now().Unix()},
		{"updated_at", time.Now().Unix()},
	}

	result, err := l.svcCtx.UserModel.InsertOne(l.ctx, &insert)
	if err != nil {
		fmt.Printf("[Error]新增用户入库：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	fmt.Println("新增用户：", result.InsertedID)
	userId := result.InsertedID.(primitive.ObjectID).Hex()

	//5.绑定新角色
	var roles []string
	for _, roleId := range req.RolesId {
		roles = append(roles, fmt.Sprintf("role_%s", roleId))
	}

	if len(roles) > 0 {
		_, err = l.svcCtx.Enforcer.AddRolesForUser(fmt.Sprintf("user_%s", userId), roles)
		if err != nil {
			fmt.Printf("[Error]用户[%s]分配角色:%s\n", userId, err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务内部错误"
			return resp, nil
		}
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
