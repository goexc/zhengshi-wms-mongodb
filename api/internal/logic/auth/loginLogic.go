package auth

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/cryptx"
	"api/pkg/jwtx"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

const userTokenKey = "token:user:%s"

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	//1.响应初始化
	resp = new(types.LoginResponse)

	//2.查询账号
	filter := bson.M{
		"name": req.Name,
		//"password": req.Password,
		"status": bson.M{"$ne": "删除"},
	}
	res := l.svcCtx.UserModel.FindOne(l.ctx, filter)

	switch res.Err() {
	case nil:
	case mongo.ErrClientDisconnected:
		fmt.Println("[Error]账号查询：MongoDB连接断开")
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	case mongo.ErrNoDocuments:
		fmt.Println("[Error]账号不存在")
		resp.Code = http.StatusUnauthorized
		resp.Msg = "不存在的账号"
		return resp, nil
	default:
		fmt.Printf("[Error]账号查询：%s\n", res.Err())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.账号解析
	var user model.User
	if err = res.Decode(&user); err != nil {
		fmt.Println("[Error]账号数据解析失败：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//4.密码校验
	if user.Password != cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, strings.TrimSpace(req.Password)) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "密码错误"
		return resp, nil
	}

	switch user.Status {
	case "启用": //启用
	case "禁用": //禁用
		resp.Code = http.StatusBadRequest
		resp.Msg = "账号已禁用"
		return resp, nil
	default:
		resp.Code = http.StatusBadRequest
		resp.Msg = "账号状态异常，请咨询管理员"
		return resp, nil
	}

	//5.生成token
	now := time.Now()
	token, err := jwtx.GetToken(l.svcCtx.Config.Auth.AccessSecret, user.Id.Hex(), now.Unix(), l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		fmt.Printf("[Error]账号[%s]生成token:%s\n", user.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//6.缓存token
	err = l.svcCtx.Cache.SetWithExpireCtx(l.ctx, fmt.Sprintf(userTokenKey, user.Id.Hex()), token, time.Duration(l.svcCtx.Config.Auth.AccessExpire))
	if err != nil {
		fmt.Printf("[Error]缓存用户[%s]Token:%s\n", user.Name, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	//todo:test
	var tokenString string
	if err = l.svcCtx.Cache.Get(fmt.Sprintf(userTokenKey, user.Id.Hex()), &tokenString); err != nil {
		fmt.Println("查询token缓存失败：", err.Error())

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}
	fmt.Println("tokenString:", tokenString)

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Name = user.Name
	resp.Data.Token = token
	resp.Data.Exp = now.Unix() + l.svcCtx.Config.Auth.AccessExpire
	return resp, nil
}
