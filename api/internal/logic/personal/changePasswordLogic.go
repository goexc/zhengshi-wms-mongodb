package personal

import (
	"api/pkg/cryptx"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"time"

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

func (l *ChangePasswordLogic) ChangePassword(req *types.ProfilePasswordRequest) (resp *types.BaseResponse, err error) {
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
	update := bson.M{
		"$set": bson.M{
			"password":   cryptx.PasswordEncrypt(l.svcCtx.Config.Salt, req.Password),
			"updated_at": time.Now().Unix(),
		},
	}

	singleRes := l.svcCtx.UserModel.FindOneAndUpdate(l.ctx, filter, &update)
	switch singleRes.Err() {
	case nil: //个人存在
	case mongo.ErrNoDocuments: //个人不存在
		fmt.Printf("[Error]个人[%s]不存在\n", uid)
		resp.Code = http.StatusBadRequest
		resp.Msg = "个人不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询并修改个人[%s]密码:%s\n", uid, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
