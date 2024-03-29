package images

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 * 2 << 20 // 20 MB

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

func (l *AddLogic) Add(r *http.Request) (resp *types.ImageResponse, err error) {
	resp = new(types.ImageResponse)
	fmt.Println("图片上传 starting")

	if err = r.ParseMultipartForm(maxFileSize); err != nil {
		fmt.Printf("[Error]图片不能超过5MB：%s\n", err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "图片过大"
		return resp, nil
	}

	f, h, err := r.FormFile("files")
	if err != nil {
		fmt.Printf("[Error]图片读取失败：%s\n", err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = err.Error()
		return resp, nil
	}
	defer f.Close()

	fmt.Printf("Uploaded File: %+v\n", h.Filename)
	fmt.Printf("File Size: %+v\n", h.Size)
	fmt.Printf("MIME Header: %+v\n", h.Header)

	objectKey := fmt.Sprintf("%s-%d%s", time.Now().Format("20060102150405"), rand.Intn(10000), strings.ToLower(filepath.Ext(h.Filename)))
	//objectPath := filepath.Join("images", objectKey)
	objectPath := "images/" + objectKey
	_, err = l.svcCtx.Cos.Object.Put(l.ctx, objectPath, f, nil)
	if err != nil {
		fmt.Printf("[Error]图片上传：%s,%s\n", objectPath, err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "图片上传失败"
		return resp, nil
	}

	// 去掉文件名中的后缀部分
	alt := h.Filename[:len(h.Filename)-len(filepath.Ext(h.Filename))]

	//入库
	image := model.Image{
		ObjectKey: objectKey,
		Alt:       alt,
		Size:      h.Size / 1024,
		CreatedAt: time.Now().Unix(),
	}
	_, err = l.svcCtx.ImageModel.InsertOne(l.ctx, &image)
	if err != nil {
		fmt.Printf("[Error]图片[%s]入库:%s\n", objectKey, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	//resp.Data.Url = fmt.Sprintf("%s/%s", l.svcCtx.Config.OSS.Domain, objectKey)
	resp.Data.Url = objectKey
	return resp, nil
}
