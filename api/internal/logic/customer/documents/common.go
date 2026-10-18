package documents

import (
	finance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/attachmentx"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

const acceptanceSource = "customer_acceptance"
const maxMoney = 9999999999.99

type ruleError struct {
	code    int
	message string
}

func (e *ruleError) Error() string  { return e.message }
func invalid(message string) error  { return &ruleError{http.StatusBadRequest, message} }
func conflict(message string) error { return &ruleError{http.StatusConflict, message} }

func responseStatus(ctx context.Context, err error) (int, string) {
	if err == nil {
		return http.StatusOK, "成功"
	}
	var rule *ruleError
	if errors.As(err, &rule) {
		return rule.code, rule.message
	}
	if message, ok := finance.IsBusinessError(err); ok {
		return http.StatusBadRequest, message
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return http.StatusNotFound, "记录不存在"
	}
	if mongo.IsDuplicateKeyError(err) {
		return http.StatusConflict, "票号、发票号码或回款关联已存在，请刷新核对"
	}
	logx.WithContext(ctx).Errorf("customer document operation: %v", err)
	return http.StatusInternalServerError, "操作失败或结果尚未确认，请保留当前请求并重试或刷新核对"
}

func objectID(value string) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(strings.TrimSpace(value))
	if err != nil || id.IsZero() {
		return id, invalid("记录或客户ID格式错误")
	}
	return id, nil
}

func moneyMinor(value float64, allowZero bool) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > maxMoney || (!allowZero && value == 0) {
		return 0, invalid("金额须在允许范围内且大于0（税额可为0）")
	}
	d := decimal.NewFromFloat(value).Shift(2)
	if !d.Equal(d.Truncate(0)) {
		return 0, invalid("金额最多保留两位小数")
	}
	return d.IntPart(), nil
}
func major(minor int64) float64 { return decimal.NewFromInt(minor).Shift(-2).InexactFloat64() }

func number(value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 128 {
		return "", "", invalid("票号或发票号码必填且最多128字")
	}
	for _, r := range value {
		if r < ' ' {
			return "", "", invalid("号码不能含控制字符")
		}
	}
	return value, hash(strings.ToUpper(value)), nil
}
func key(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return "", invalid("缺少有效幂等键，请重新打开表单")
	}
	return value, nil
}
func remark(value string) (string, error) {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > 2000 {
		return "", invalid("备注最多2000字")
	}
	return value, nil
}
func validDate(value int64, optional bool) bool {
	return (optional && value == 0) || (value > 0 && value <= 4102444799)
}

// Business dates compare by the Beijing calendar day. Keep their original
// seconds in storage and request fingerprints so existing receipts still replay.
func beforeShanghaiDate(left, right int64) bool {
	return shanghaiToday(time.Unix(left, 0)) < shanghaiToday(time.Unix(right, 0))
}
func hash(value any) string {
	bytes, _ := json.Marshal(value)
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
func annex(refs []string) (string, error) {
	clean, err := attachmentx.ValidateReferences(refs)
	if err != nil {
		return "", invalid(err.Error())
	}
	return strings.Join(clean, ","), nil
}
func annexList(value string) []string {
	result := make([]string, 0)
	for _, s := range strings.Split(value, ",") {
		if s = strings.TrimSpace(s); s != "" {
			result = append(result, s)
		}
	}
	return result
}
func actor(ctx context.Context) (string, string) {
	id, _ := ctx.Value("uid").(string)
	name, _ := ctx.Value("name").(string)
	return id, name
}
func history(ctx context.Context, action, note, requestKey, requestHash string) model.DocumentHistory {
	id, name := actor(ctx)
	return model.DocumentHistory{Action: action, Remark: note, Time: time.Now().Unix(), Operator: id, OperatorName: name, IdempotencyKey: requestKey, RequestHash: requestHash}
}
func historyView(rows []model.DocumentHistory) []types.CustomerDocumentHistory {
	result := make([]types.CustomerDocumentHistory, 0, len(rows))
	for _, row := range rows {
		result = append(result, types.CustomerDocumentHistory{Action: row.Action, Remark: row.Remark, Time: row.Time, Operator: row.Operator, OperatorName: row.OperatorName})
	}
	return result
}
func replay(rows []model.DocumentHistory, requestKey, requestHash string) (bool, error) {
	for _, row := range rows {
		if row.IdempotencyKey == requestKey {
			if row.RequestHash != requestHash {
				return false, conflict("幂等键已用于不同的操作内容")
			}
			return true, nil
		}
	}
	return false, nil
}
func transaction(ctx context.Context, s *svc.ServiceContext, fn func(mongo.SessionContext) error) error {
	session, err := s.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(dbCtx mongo.SessionContext) (interface{}, error) { return nil, fn(dbCtx) }, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}

// Touching the customer closes the insert-versus-delete write-skew window.
// Customer deletion writes the same document inside its own transaction.
func touchCustomer(ctx context.Context, s *svc.ServiceContext, customerID string) (model.Customer, error) {
	var customer model.Customer
	id, err := objectID(customerID)
	if err != nil {
		return customer, err
	}
	err = s.CustomerModel.FindOneAndUpdate(ctx, bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}}, bson.M{"$inc": bson.M{"document_revision": int64(1)}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&customer)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return customer, invalid("客户不存在或已删除")
	}
	return customer, err
}
func base(ctx context.Context, customer model.Customer, numberKey, status, annexValue, note, requestKey, requestHash string) model.CustomerDocumentBase {
	id, name := actor(ctx)
	now := time.Now().Unix()
	return model.CustomerDocumentBase{Id: primitive.NewObjectID(), CustomerId: customer.Id.Hex(), CustomerName: customer.Name, NumberKey: numberKey, Status: status, Annex: annexValue, Remark: note, Version: 1, IdempotencyKey: requestKey, RequestHash: requestHash, Creator: id, CreatorName: name, CreatedAt: now, UpdatedAt: now, History: []model.DocumentHistory{history(ctx, "create", "登记", requestKey, requestHash)}}
}
func checkVersion(actual, requested int64) error {
	if requested < 1 || actual != requested {
		return conflict("记录已被修改，请刷新后重试")
	}
	return nil
}
func modified(res *mongo.UpdateResult, err error) error {
	if err != nil {
		return err
	}
	if res.MatchedCount != 1 {
		return conflict("记录状态或版本已变化，请刷新后重试")
	}
	return nil
}
func voidInput(req *types.CustomerDocumentVoidRequest) (primitive.ObjectID, string, string, error) {
	id, err := objectID(req.Id)
	if err != nil {
		return id, "", "", err
	}
	requestKey, err := key(req.IdempotencyKey)
	if err != nil {
		return id, "", "", err
	}
	reason, err := remark(req.Reason)
	if err != nil {
		return id, "", "", err
	}
	if reason == "" {
		return id, "", "", invalid("作废原因必填")
	}
	return id, requestKey, reason, nil
}
func stateFilter(id primitive.ObjectID, version int64, status string) bson.M {
	return bson.M{"_id": id, "version": version, "status": status}
}
func beforeSnapshot(record any) bson.M {
	bytes, _ := bson.Marshal(record)
	result := bson.M{}
	_ = bson.Unmarshal(bytes, &result)
	delete(result, "history")
	return result
}

func bindAttachments(ctx context.Context, s *svc.ServiceContext, refs []string, scope string, old []string) error {
	if err := attachmentx.ValidateScopedReferences(refs, scope, old); err != nil {
		return invalid(err.Error())
	}
	for _, ref := range refs {
		if s.ImageModel == nil {
			return invalid("附件元数据服务不可用")
		}
		res, err := s.ImageModel.UpdateOne(ctx, bson.M{"object_key": ref, "deleting": bson.M{"$ne": true}}, bson.M{"$inc": bson.M{"reference_revision": int64(1)}})
		if err != nil {
			return err
		}
		if res.MatchedCount != 1 {
			if err = s.ImageModel.FindOne(ctx, bson.M{"object_key": ref}).Err(); err == nil {
				return invalid("附件正在删除，请重新上传")
			}
			if !errors.Is(err, mongo.ErrNoDocuments) {
				return err
			}
			found := false
			for _, v := range old {
				if v == ref {
					found = true
					break
				}
			}
			if !found {
				return invalid("附件不存在或已删除，请重新上传")
			}
		}
	}
	return nil
}

// The original version identifies an update operation even for older clients
// which only send id/version. Persist its fingerprint before allowing retries.
func updateReceipt(id primitive.ObjectID, version int64, payload any) (string, string) {
	return fmt.Sprintf("update:%s:%d", id.Hex(), version), hash([]any{"update", id.Hex(), version, payload})
}
func mergeReferences(old []string, values ...string) []string {
	result := append([]string{}, old...)
	seen := map[string]bool{}
	for _, v := range old {
		seen[v] = true
	}
	for _, value := range values {
		for _, v := range annexList(value) {
			if !seen[v] {
				result = append(result, v)
				seen[v] = true
			}
		}
	}
	return result
}
