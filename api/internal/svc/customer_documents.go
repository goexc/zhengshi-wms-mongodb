package svc

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureCustomerDocumentIndexes also serves isolated integration fixtures.
func EnsureCustomerDocumentIndexes(ctx context.Context, s *ServiceContext) error {
	for _, entry := range []struct {
		collection *mongo.Collection
		date       string
	}{
		{s.CustomerAcceptanceModel, "received_date"}, {s.CustomerInvoiceModel, "invoice_date"},
	} {
		indexes := []mongo.IndexModel{
			{Keys: bson.D{{Key: "number_key", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_document_number")},
			{Keys: bson.D{{Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_document_request")},
			{Keys: bson.D{{Key: "customer_id", Value: 1}, {Key: "status", Value: 1}, {Key: entry.date, Value: -1}, {Key: "_id", Value: -1}}, Options: options.Index().SetName("idx_document_customer_status_date")},
			{Keys: bson.D{{Key: entry.date, Value: -1}, {Key: "_id", Value: -1}}, Options: options.Index().SetName("idx_document_date")},
		}
		if _, err := entry.collection.Indexes().CreateMany(ctx, indexes); err != nil {
			return err
		}
	}
	_, err := s.CustomerAcceptanceModel.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "transaction_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_acceptance_payment").SetPartialFilterExpression(bson.M{"transaction_id": bson.M{"$type": "string", "$gt": ""}})},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "due_date", Value: 1}}, Options: options.Index().SetName("idx_acceptance_due")},
	})
	return err
}
