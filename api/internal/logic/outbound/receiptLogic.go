package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReceiptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiptLogic {
	return &ReceiptLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReceiptLogic) Receipt(req *types.OutboundOrderReceiptRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	//1.查询出库单
	//1.1 查询出库单
	var code = strings.TrimSpace(req.Code)
	singleRes := l.svcCtx.OutboundOrderModel.FindOne(l.ctx, bson.M{"code": code})
	switch singleRes.Err() {
	case nil: //出库单存在
	case mongo.ErrNoDocuments: //出库单不存在
		fmt.Printf("[Error]出库单[%s]不存在\n", code)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单不存在"
		return resp, nil
	default: //其他错误
		fmt.Printf("[Error]查询出库单[%s]:%s\n", code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务内部错误"
		return resp, nil
	}

	var order model.OutboundOrder
	if err = singleRes.Decode(&order); err != nil {
		fmt.Printf("[Error]解析出库单[%s]:%s\n", code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.2 出库单状态是“已出库”
	switch order.Status {
	case "预发货", "待拣货", "已拣货", "已打包", "已称重":
		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("%s出库单无法签收", order.Status)
		return resp, nil
	case "已出库":
	default:
		resp.Code = http.StatusBadRequest
		resp.Msg = "不能重复签收"
		return resp, nil
	}

	//1.3 签收时间不能超过当前时间
	if req.ReceiptTime > time.Now().Unix() {
		resp.Code = http.StatusBadRequest
		resp.Msg = fmt.Sprintf("出库单签收时间不能超过当前时间")
		return resp, nil
	}

	//1.4 计算出库单总金额
	var amount decimal.Decimal

	//1.4.1 检索出库单物料列表
	cur, err := l.svcCtx.OutboundMaterialModel.Find(l.ctx, bson.M{"order_code": code})
	if err != nil {
		fmt.Printf("[Error]检索出库单[%s]物料列表：%s\n", code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	//1.4.2 解析出库单物料列表
	var materials []model.OutboundOrderMaterial
	if err = cur.All(l.ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单[%s]物料列表：%s\n", code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	//1.4.3 累计物料金额
	unpricedCount := 0
	for _, one := range materials {
		if one.Price <= 0 {
			unpricedCount++
		}
		amount = decimal.NewFromFloat(one.Quantity).Mul(decimal.NewFromFloat(one.Price)).Add(amount)
	}

	//1.4.4 累计运费、其他费用
	amount = amount.Add(decimal.NewFromFloat(order.CarrierCost)).Add(decimal.NewFromFloat(order.OtherCost))

	//2.创建事务
	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]签收出库单：创建事务：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	// 根据会话获取数据库操作的上下文
	dbCtx := mongo.NewSessionContext(l.ctx, session)

	//2.1 开始事务
	err = session.StartTransaction()
	if err != nil {
		fmt.Printf("[Error]签收出库单：开始事务：%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	priceStatus := financeCode.OutboundPriceStatusFullyPriced
	switch {
	case len(materials) > 0 && unpricedCount == len(materials):
		priceStatus = financeCode.OutboundPriceStatusUnpriced
	case unpricedCount > 0:
		priceStatus = financeCode.OutboundPriceStatusPartialPriced
	}
	outboundTypeCode := order.TypeCode
	if outboundTypeCode == "" {
		outboundTypeCode = financeCode.OutboundTypeCodeFromLabel(order.Type)
	}
	arStatus := financeCode.OutboundARStatusNotApplicable
	shouldCreateAR := financeCode.ShouldCreateReceivable(outboundTypeCode)
	if shouldCreateAR {
		arStatus = financeCode.OutboundARStatusPending
		if priceStatus == financeCode.OutboundPriceStatusFullyPriced {
			arStatus = financeCode.OutboundARStatusPosted
		}
	}

	//4.修改发货单状态：已出库->已签收
	var set = bson.M{
		"$set": bson.M{
			"status":       "已签收",
			"annex":        strings.Join(req.Annex, ","),
			"total_amount": amount.InexactFloat64(),
			"type_code":    outboundTypeCode,
			"price_status": priceStatus,
			"ar_status":    arStatus,
			"receipt_time": req.ReceiptTime,
		},
	}

	updateRes, err := l.svcCtx.OutboundOrderModel.UpdateOne(dbCtx, bson.M{"_id": order.Id, "status": "已出库"}, &set)
	if err != nil {
		fmt.Printf("[Error]更新出库单[%s]状态(已签收)：%s\n", code, err.Error())
		// 回滚事务
		session.AbortTransaction(dbCtx)

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if updateRes.MatchedCount != 1 {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单状态已变化，请刷新后重试"
		return resp, nil
	}

	if shouldCreateAR && priceStatus == financeCode.OutboundPriceStatusFullyPriced {
		//5.添加客户交易记录。未完全定价的签收单只进入 pending，应收在补齐价格后生成。
		now := time.Now().Unix()
		_, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
			TransactionType: financeCode.TransactionTypeOutboundAR,
			Status:          financeCode.TransactionStatusConfirmed,
			Code:            fmt.Sprintf("CT-%s-%d", time.Now().Format("2006-01-02-15-04-05"), time.Now().UnixMilli()%1000),
			OrderCode:       order.Code,
			SourceType:      financeCode.SourceTypeOutboundOrder,
			SourceId:        order.Id.Hex(),
			SourceCode:      order.Code,
			IdempotencyKey:  financeCode.IdempotencyKey(financeCode.TransactionTypeOutboundAR, order.Id.Hex()),
			CustomerId:      order.CustomerId,
			CustomerName:    order.CustomerName,
			Amount:          amount.InexactFloat64(),
			Annex:           strings.Join(req.Annex, ","),
			Time:            req.ReceiptTime,
			Creator:         l.ctx.Value("uid").(string),
			CreatorName:     l.ctx.Value("name").(string),
			CreatedAt:       now,
		})
		if err != nil {
			fmt.Printf("[Error]添加客户[%s]订单[%s]应收账款:%s\n", order.CustomerName, order.Code, err.Error())

			// 回滚事务
			session.AbortTransaction(dbCtx)

			if msg, isBusiness := customerfinance.IsBusinessError(err); isBusiness {
				resp.Code = http.StatusBadRequest
				resp.Msg = msg
				return resp, nil
			}
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	}

	// 提交事务
	err = session.CommitTransaction(dbCtx)
	if err != nil {
		fmt.Printf("[Error]出库单[%s]签收事务提交失败: %s\n", code, err.Error())

		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	/*
		//5.生成流水记录
		var transaction = model.CustomerTransaction{
			//Code:         fmt.Sprintf("CT-%s-%d", time.Now().Format("2006-01-02-15-04-05"), time.Now().UnixMilli()%1000), //YYYY-MM-DD-HH-mm-ss-SSS
			Code:         fmt.Sprintf("CT-%s", order.Code),
			OrderCode:    order.Code,
			Type:         order.Type,
			CustomerId:   order.CustomerId,
			CustomerName: order.CustomerName,
			Amount:       amount.InexactFloat64(),
			Remark:       "",
			Creator:      l.ctx.Value("uid").(string),
			CreatorName:  l.ctx.Value("name").(string),
			CreatedAt:    time.Now().Unix(),
		}

		_, err = l.svcCtx.CustomerTransactionModel.InsertOne(l.ctx, &transaction)
		if err != nil {
			fmt.Printf("[Error]新增出库单流水[%s]:%s\n", order.Code, err.Error())
			// 回滚事务
			session.AbortTransaction(dbCtx)

			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	*/
	/*
		//6.更新客户余额
		//6.1 客户余额记录是否存在
		var balance model.CustomerBalance
		singleRes = l.svcCtx.CustomerBalanceModel.FindOne(l.ctx, bson.M{"customer_id": order.CustomerId})
		switch singleRes.Err() {
		case nil:
			if err = singleRes.Decode(&balance); err != nil {
				fmt.Printf("[Error]客户[%s]余额记录解析:%s\n", order.CustomerName, err.Error())
				// 回滚事务
				session.AbortTransaction(dbCtx)

				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
			_, err = l.svcCtx.CustomerBalanceModel.UpdateByID(l.ctx,
				balance.Id,
				bson.M{
					"$inc": bson.M{
						"transaction_amount":  amount.InexactFloat64(),
						"accounts_receivable": amount.InexactFloat64(),
					},
					"$set": bson.M{
						"editor":      l.ctx.Value("uid").(string),
						"editor_name": l.ctx.Value("name").(string),
						"updated_at":  time.Now().Unix(),
					},
				},
			)
			if err != nil {
				fmt.Printf("[Error]更新客户[%s]余额记录：%s\n", order.CustomerName, err.Error())
				// 回滚事务
				session.AbortTransaction(dbCtx)

				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		case mongo.ErrNoDocuments: //客户余额记录不存在
			_, err = l.svcCtx.CustomerBalanceModel.InsertOne(l.ctx, &model.CustomerBalance{
				CustomerId:         order.CustomerId,
				CustomerName:       order.CustomerName,
				TransactionAmount:  amount.InexactFloat64(),
				AccountsReceivable: amount.InexactFloat64(),
				Remark:             "",
				Creator:            l.ctx.Value("uid").(string),
				CreatorName:        l.ctx.Value("name").(string),
				Editor:             "",
				EditorName:         "",
				CreatedAt:          time.Now().Unix(),
				UpdatedAt:          0,
			})
			if err != nil {
				fmt.Printf("[Error]新增客户[%s]余额记录：%s\n", order.CustomerName, err.Error())
				// 回滚事务
				session.AbortTransaction(dbCtx)

				resp.Code = http.StatusInternalServerError
				resp.Msg = "服务器内部错误"
				return resp, nil
			}
		default:
			fmt.Printf("[Error]客户[%s]余额记录查询：%s\n", order.CustomerName, singleRes.Err().Error())
			// 回滚事务
			session.AbortTransaction(dbCtx)

			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		// 提交事务
		err = session.CommitTransaction(dbCtx)
		if err != nil {
			fmt.Printf("[Error]出库单[%s]签收事务提交失败: %s\n", code, err.Error())

			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
	*/
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}
