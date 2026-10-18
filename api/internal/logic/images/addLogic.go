package images

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/attachmentx"
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AddLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddLogic {
	return &AddLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *AddLogic) Add(r *http.Request) (*types.ImageResponse, error) {
	resp := &types.ImageResponse{Code: http.StatusBadRequest}
	r.Body = http.MaxBytesReader(nil, r.Body, attachmentx.MaxRequestBytes)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
		resp.Msg = "上传失败，单个文件不能超过20MB"
		return resp, nil
	}
	defer r.MultipartForm.RemoveAll()
	scope := strings.TrimSpace(r.FormValue("scope"))
	if _, err := attachmentx.Prefix(scope); err != nil {
		resp.Msg = err.Error()
		return resp, nil
	}
	if !scopeAllowed(l.ctx, l.svcCtx, scope, true) {
		resp.Code = http.StatusForbidden
		resp.Msg = "没有该业务附件上传权限"
		return resp, nil
	}
	if len(r.MultipartForm.File["files"]) != 1 {
		resp.Msg = "每个上传请求应包含一个文件"
		return resp, nil
	}
	f, h, err := r.FormFile("files")
	if err != nil {
		resp.Msg = "文件读取失败"
		return resp, nil
	}
	defer f.Close()
	if h.Size <= 0 || h.Size > attachmentx.MaxFileBytes {
		resp.Msg = "文件不能为空且不能超过20MB"
		return resp, nil
	}
	key, err := attachmentx.NewReference(scope, h.Filename, time.Now())
	if err != nil {
		resp.Msg = err.Error()
		return resp, nil
	}
	objectPath := "images/" + key
	if _, err = l.svcCtx.Cos.Object.Put(l.ctx, objectPath, f, nil); err != nil {
		l.Errorf("upload attachment: %v", err)
		resp.Msg = "文件上传失败"
		return resp, nil
	}
	name := filepath.Base(strings.ReplaceAll(h.Filename, "\\", "/"))
	record := model.Image{Id: primitive.NewObjectID(), ObjectKey: key, Scope: scope, Alt: strings.TrimSuffix(name, filepath.Ext(name)), Size: h.Size / 1024, CreatedAt: time.Now().Unix()}
	if _, err = l.svcCtx.ImageModel.InsertOne(l.ctx, &record); err != nil {
		l.Errorf("save uploaded attachment metadata %s (%s): %v", key, record.Id.Hex(), err)
		// A network/timeout error does not prove that InsertOne failed. Reconcile
		// using the preallocated ID even when the original request was cancelled.
		// Never delete the COS object on an unknown metadata outcome: a committed
		// row could already have been selected from the library and bound elsewhere.
		verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		verifyErr := l.svcCtx.ImageModel.FindOne(verifyCtx, bson.M{"_id": record.Id, "object_key": key, "deleting": bson.M{"$ne": true}}).Err()
		cancel()
		if verifyErr != nil {
			resp.Code = http.StatusInternalServerError
			resp.Msg = "附件信息保存结果未确认，请到素材库核对后重试"
			return resp, nil
		}
	}
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Url = key
	return resp, nil
}
