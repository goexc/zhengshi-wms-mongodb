package images

import (
	"api/internal/svc"
	"context"
	"fmt"
	"regexp"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func referencedAttachment(ctx context.Context, s *svc.ServiceContext, key string) (string, error) {
	pattern := primitive.Regex{Pattern: "(^|,)" + regexp.QuoteMeta(key) + "(,|$)"}
	for _, entry := range []struct {
		collection *mongo.Collection
		name       string
		fields     []string
	}{
		{s.CustomerAcceptanceModel, "客户承兑", []string{"annex", "redemption_annex", "attachment_history"}},
		{s.CustomerInvoiceModel, "销售开票记录", []string{"annex", "attachment_history"}},
		{s.CustomerTransactionModel, "客户交易流水", []string{"annex"}},
		{s.InboundReceiptModel, "入库单", []string{"annex"}},
		{s.InboundReceiptReceiveModel, "入库记录", []string{"annex"}},
		{s.OutboundOrderModel, "出库单", []string{"annex"}},
		{s.MaterialModel, "物料", []string{"image"}},
		{s.UserModel, "用户", []string{"avatar"}},
		{s.SupplierModel, "供应商", []string{"image"}},
		{s.CustomerModel, "客户", []string{"image"}},
		{s.CarrierModel, "承运商", []string{"image"}},
	} {
		if entry.collection == nil {
			continue
		}
		or := bson.A{}
		for _, field := range entry.fields {
			or = append(or, bson.M{field: pattern}, bson.M{field: key})
		}
		count, err := entry.collection.CountDocuments(ctx, bson.M{"$or": or}, options.Count().SetLimit(1))
		if err != nil {
			return "", err
		}
		if count != 0 {
			return entry.name + "或其历史记录引用该附件，无法删除", nil
		}
	}
	return "", nil
}

// A tombstone is committed before touching COS. New financial attachments bind
// by writing the same image row with deleting != true, so a concurrent bind
// cannot race past the reference check. Failed COS deletes can safely retry.
func reserveAttachmentRemoval(ctx context.Context, s *svc.ServiceContext, key string) error {
	session, err := s.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (interface{}, error) {
		message, e := referencedAttachment(tx, s, key)
		if e != nil {
			return nil, e
		}
		if message != "" {
			return nil, fmt.Errorf("%s", message)
		}
		result, e := s.ImageModel.UpdateOne(tx, bson.M{"object_key": key}, bson.M{"$set": bson.M{"deleting": true}, "$inc": bson.M{"reference_revision": 1}})
		if e != nil {
			return nil, e
		}
		if result.MatchedCount != 1 {
			return nil, fmt.Errorf("附件已不存在")
		}
		return nil, nil
	})
	return err
}
