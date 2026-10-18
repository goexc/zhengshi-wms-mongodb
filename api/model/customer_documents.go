package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DocumentHistory is append-only. Request hashes distinguish a retry from reuse
// of the same key with a different financial instruction.
type DocumentHistory struct {
	Action         string `bson:"action"`
	Remark         string `bson:"remark"`
	Time           int64  `bson:"time"`
	Operator       string `bson:"operator"`
	OperatorName   string `bson:"operator_name"`
	IdempotencyKey string `bson:"idempotency_key,omitempty"`
	RequestHash    string `bson:"request_hash,omitempty"`
	Before         bson.M `bson:"before,omitempty"`
}

type CustomerDocumentBase struct {
	Id                primitive.ObjectID `bson:"_id,omitempty"`
	CustomerId        string             `bson:"customer_id"`
	CustomerName      string             `bson:"customer_name"`
	NumberKey         string             `bson:"number_key"`
	Status            string             `bson:"status"`
	Annex             string             `bson:"annex"`
	Remark            string             `bson:"remark"`
	Version           int64              `bson:"version"`
	IdempotencyKey    string             `bson:"idempotency_key"`
	RequestHash       string             `bson:"request_hash"`
	Creator           string             `bson:"creator"`
	CreatorName       string             `bson:"creator_name"`
	CreatedAt         int64              `bson:"created_at"`
	UpdatedAt         int64              `bson:"updated_at"`
	History           []DocumentHistory  `bson:"history"`
	AttachmentHistory []string           `bson:"attachment_history"`
}

// Monetary minor fields are the exact cent values used for sums and equality.
// Public float fields preserve the existing API convention at the boundary.
type CustomerAcceptance struct {
	CustomerDocumentBase `bson:",inline"`
	BillNumber           string  `bson:"bill_number"`
	Amount               float64 `bson:"amount"`
	AmountMinor          int64   `bson:"amount_minor"`
	ReceivedDate         int64   `bson:"received_date"`
	IssueDate            int64   `bson:"issue_date"`
	DueDate              int64   `bson:"due_date"`
	RedemptionDate       int64   `bson:"redemption_date"`
	RedemptionAnnex      string  `bson:"redemption_annex"`
	TransactionId        string  `bson:"transaction_id,omitempty"`
}

type CustomerInvoice struct {
	CustomerDocumentBase    `bson:",inline"`
	InvoiceNumber           string  `bson:"invoice_number"`
	InvoiceDate             int64   `bson:"invoice_date"`
	TaxRate                 float64 `bson:"tax_rate"`
	AmountExcludingTax      float64 `bson:"amount_excluding_tax"`
	TaxAmount               float64 `bson:"tax_amount"`
	TotalAmount             float64 `bson:"total_amount"`
	AmountExcludingTaxMinor int64   `bson:"amount_excluding_tax_minor"`
	TaxAmountMinor          int64   `bson:"tax_amount_minor"`
	TotalAmountMinor        int64   `bson:"total_amount_minor"`
}
