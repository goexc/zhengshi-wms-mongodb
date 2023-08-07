package customer

import (
	"api/model"
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

type PageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PageLogic {
	return &PageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PageLogic) Page(req *types.CustomersRequest) (resp *types.CustomersResponse, err error) {
	resp = new(types.CustomersResponse)

	name := strings.TrimSpace(req.Name)
	//1.客户分页
	var filter = bson.M{"status": bson.M{"$ne": "删除"}} //过滤已删除客户
	if name != "" {
		//i 表示不区分大小写
		regex := bson.M{"$regex": primitive.Regex{Pattern: ".*" + name + ".*", Options: "i"}}
		filter = bson.M{"name": regex, "status": bson.M{"$ne": "删除"}}
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
			{"code", 1},
			{"image", 1},
			{"legal_representative", 1},
			{"unified_social_credit_identifier", 1},
			{"name", 1},
			{"address", 1},
			{"contact", 1},
			{"manager", 1},
			{"level", 1},
			{"status", 1},
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

	cur, err := l.svcCtx.CustomerModel.Aggregate(l.ctx, pipeline)
	if err != nil {
		fmt.Printf("[Error]查询客户列表:%s\n", err.Error())
		resp.Msg = "服务器内部错误"
		resp.Code = http.StatusInternalServerError
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var suppliers []model.Customer
	if err = cur.All(l.ctx, &suppliers); err != nil {
		fmt.Println("[Error]解析客户列表：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	fmt.Printf("客户数量:%d\n", len(suppliers))

	//2.客户总数量
	total, err := l.svcCtx.CustomerModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Println("[Error]客户总数量：", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Data.Total = total
	resp.Data.List = make([]types.Customer, 0)
	for _, supplier := range suppliers {
		resp.Data.List = append(resp.Data.List, types.Customer{
			Id:                            supplier.Id.Hex(),
			Type:                          supplier.Type,
			Code:                          strings.TrimSpace(supplier.Code),
			Image:                         supplier.Image,
			LegalRepresentative:           strings.TrimSpace(supplier.LegalRepresentative),
			UnifiedSocialCreditIdentifier: strings.TrimSpace(supplier.UnifiedSocialCreditIdentifier),
			Name:                          supplier.Name,
			Address:                       supplier.Address,
			Contact:                       supplier.Contact,
			Manager:                       supplier.Manager,
			Status:                        supplier.Status,
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
