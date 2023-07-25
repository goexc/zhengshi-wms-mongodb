package supplier

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *types.SuppliersRequest) (resp *types.SuppliersResponse, err error) {
	resp = new(types.SuppliersResponse)

	name := strings.TrimSpace(req.Name)
	//1.供应商分页
	var filter = bson.M{"status": bson.M{"$ne": 100}} //过滤已删除供应商
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter = bson.M{"name": regex, "status": bson.M{"$ne": 100}}
		//matchStage = bson.D{
		//	{"$match", filter},
		//}
	}
	if strings.TrimSpace(req.Code) != "" {
		filter["code"] = strings.TrimSpace(req.Code)
	}

	if strings.TrimSpace(req.Manager) != "" {
		filter["manager"] = strings.TrimSpace(req.Manager)
	}

	if strings.TrimSpace(req.Contact) != "" {
		filter["contact"] = strings.TrimSpace(req.Contact)
	}
	if strings.TrimSpace(req.Email) != "" {
		filter["email"] = strings.TrimSpace(req.Email)
	}
	if req.Level > 0 {
		filter["level"] = req.Level
	}
	var matchStage = bson.D{{"$match", filter}}

	//$lookup 阶段进行关联查询，
	//将 supplier 集合中的 create_by 字段与 user 集合中的 _id 字段进行关联
	lookupStage := bson.D{
		{"$lookup", bson.D{
			{"from", "user"},
			{"localField", "creator"},
			{"foreignField", "_id"},
			{"as", "user"},
		}},
	}

	//使用 $unwind 阶段展开 user 数组
	//使用 $unwind 阶段展开关联结果，并设置 preserveNullAndEmptyArrays 选项为 true，以便在没有关联数据时仍然保留主集合中的记录。
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$user"}, {"preserveNullAndEmptyArrays", true}}}}

	//使用 $project 阶段对结果进行投影，将 user.name 的值赋给 create_by 字段。
	projectStage := bson.D{
		{"$project", bson.D{
			{"_id", 1},
			{"type", 1},
			{"level", 1},
			{"status", 1},
			{"name", 1},
			{"code", 1},
			{"image", 1},
			{"legal_representative", 1},
			{"unified_social_credit_identifier", 1},
			{"manager", 1},
			{"contact", 1},
			{"email", 1},
			{"address", 1},
			{"remark", 1},
			{"creator_name", bson.D{
				//在投影中，使用 $ifNull 操作符检查 user.name 字段，
				//如果为空（即关联集合没有数据），则将 create_by 字段置为空字符串。
				//这样就可以将关联字段置为空，以处理关联集合没有查找到数据的情况。
				{"$ifNull", bson.A{"$user.name", ""}},
			}},
			{"created_at", 1},
			{"updated_at", 1},
			//{"creator", 0},//$project 阶段只能用于指定要包含的字段，而不能同时指定要包含和排除的字段。
		}},
	}
	sortStage := bson.D{
		{"$sort", bson.D{
			{"created_at", -1}, // 降序排列
		},
		},
	}
	skipStage := bson.D{{"$skip", (req.Page - 1) * req.Size}}
	limitStage := bson.D{{"$limit", req.Size}}

	pipeline := mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		projectStage,
		sortStage,
		skipStage,
		limitStage,
	}

	cur, err := l.svcCtx.SupplierModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询供应商列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var suppliers []model.Supplier
	if err = cur.All(l.ctx, &suppliers); err != nil {
		fmt.Println("[Error]解析供应商列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	fmt.Printf("供应商数量:%d\n", len(suppliers))

	//2.供应商总数量
	total, err := l.svcCtx.SupplierModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]供应商总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.Total = total
	resp.Data.List = make([]types.Supplier, 0)
	for _, supplier := range suppliers {
		resp.Data.List = append(resp.Data.List, types.Supplier{
			Id:                            supplier.Id.Hex(),
			Type:                          supplier.Type,
			Level:                         supplier.Level,
			Status:                        code.SupplierStatusText(supplier.Status),
			Name:                          supplier.Name,
			Code:                          strings.TrimSpace(supplier.Code),
			Image:                         strings.TrimSpace(supplier.Image),
			LegalRepresentative:           strings.TrimSpace(supplier.LegalRepresentative),
			UnifiedSocialCreditIdentifier: strings.TrimSpace(supplier.UnifiedSocialCreditIdentifier),
			Manager:                       supplier.Manager,
			Contact:                       supplier.Contact,
			Email:                         supplier.Email,
			Address:                       supplier.Address,
			Remark:                        supplier.Remark,
			CreateBy:                      supplier.CreatorName,
			CreatedAt:                     supplier.CreatedAt,
			UpdatedAt:                     supplier.UpdatedAt,
		})
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
