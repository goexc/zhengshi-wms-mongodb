package auth

import (
	"api/model"
	"api/pkg/cryptx"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
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

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.手机号码、Email校验
	if len(strings.TrimSpace(req.Mobile)) == 0 || len(strings.TrimSpace(req.Email)) == 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "手机号码、email不能为空"
		return resp, nil
	}

	//2.账号、手机号码、email唯一性校验
	filter := bson.M{
		"$or": []bson.M{
			{"account": req.Account},
			{"mobile": req.Mobile},
			{"email": req.Email},
		},
	}
	res := l.svcCtx.UserModel.FindOne(l.ctx, filter)
	switch res.Err() {
	case nil:
		var user model.User
		if err = res.Decode(&user); err != nil {
			fmt.Printf("[Error]已注册用户解析：%s\n", err.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		resp.Code = http.StatusBadRequest
		if strings.TrimSpace(req.Account) == user.Account {
			resp.Msg = "账号名称已占用"
		} else if strings.TrimSpace(req.Email) == user.Email {
			resp.Msg = "Email已占用"
		} else if strings.TrimSpace(req.Mobile) == user.Mobile {
			resp.Msg = "手机号码已占用"
		}

		return resp, nil
	case mongo.ErrClientDisconnected:
		fmt.Println("[Error]账号、手机号码、email唯一性校验：MongoDB连接断开")
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	case mongo.ErrNoDocuments: //可以注册
	default:
		fmt.Printf("[Error]账号、手机号码、email唯一性校验：%s\n", res.Err())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//3.写入数据库
	user := bson.M{
		"account":    strings.TrimSpace(req.Account),
		"password":   cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password),
		"mobile":     strings.TrimSpace(req.Mobile),
		"email":      strings.TrimSpace(req.Email),
		"avatar":     l.svcCtx.Config.Avatar,
		"sex":        req.Sex,
		"status":     0,
		"created_at": time.Now().Unix(),
		"updated_at": time.Now().Unix(),
	}
	_, err = l.svcCtx.UserModel.InsertOne(l.ctx, &user)
	if err != nil {
		fmt.Printf("[Error]注册入库：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
