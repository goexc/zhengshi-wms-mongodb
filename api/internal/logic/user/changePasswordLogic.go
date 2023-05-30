package user

import (
	"api/pkg/cryptx"
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

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordRequest) (resp *types.BaseResponse, err error) {
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

	//2.修改用户状态
	update := bson.M{
		"$set": bson.M{
			"password": cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password),
		},
	}

	_, err = l.svcCtx.UserModel.UpdateByID(l.ctx, id, &update)
	if err != nil {
		fmt.Printf("[Error]修改用户[%s]密码：%s\n", req.Id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
