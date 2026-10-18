package outbound

import (
	"api/model"
	code "api/pkg/code"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func removalUnitSnapshot() removalSnapshot {
	customer := model.Customer{Id: primitive.NewObjectID(), Name: "测试客户", ReceivableBalance: 999}
	order := model.OutboundOrder{Id: primitive.NewObjectID(), Code: "O-UNIT", Status: "已签收", CustomerId: customer.Id.Hex(), CustomerName: customer.Name, TotalAmount: 100}
	inventory := model.Inventory{Id: primitive.NewObjectID(), MaterialId: primitive.NewObjectID().Hex(), Name: "测试物料", Unit: "件", Quantity: 10, AvailableQuantity: 8}
	material := model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: order.Code, MaterialId: inventory.MaterialId, Name: inventory.Name, Unit: inventory.Unit, Quantity: 2, Price: 50,
		Inventorys: []model.OutboundMaterialInventory{{InventoryId: inventory.Id.Hex(), ShipmentQuantity: 2}}}
	record := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: customer.Id.Hex(), TransactionType: code.TransactionTypeOutboundAR,
		SourceType: code.SourceTypeOutboundOrder, SourceId: order.Id.Hex(), OrderCode: order.Code, SourceCode: order.Code, Amount: 100}
	return removalSnapshot{Order: order, Customer: &customer, Materials: []model.OutboundOrderMaterial{material}, Inventories: []model.Inventory{inventory}, Transactions: []model.CustomerTransaction{record}}
}

func TestRemovalUsesLedgerNetAndKeepsIndependentPayments(t *testing.T) {
	snapshot := removalUnitSnapshot()
	adjustment := snapshot.Transactions[0]
	adjustment.Id = primitive.NewObjectID()
	adjustment.TransactionType = code.TransactionTypeARAdjustment
	adjustment.SourceType = code.SourceTypeSystemRecount
	adjustment.Direction = code.TransactionDirectionDecrease
	adjustment.Amount = 20
	payment := adjustment
	payment.Id = primitive.NewObjectID()
	payment.TransactionType = code.TransactionTypePayment
	payment.Amount = 80
	snapshot.Transactions = append(snapshot.Transactions, adjustment, payment)
	plan, err := prepareRemovalPlan(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Preview.TransactionCount != 2 || plan.Preview.ReceivableReduction != 80 || plan.Preview.ReceivableBefore != 0 || plan.Preview.CreditAfter != 80 {
		t.Fatalf("incorrect ledger projection: %+v", plan.Preview)
	}
	if len(plan.Changes) != 1 || plan.Changes[0].Restore != 2 {
		t.Fatal("incorrect stock restoration")
	}
}

func TestRemovalLegacyOwnershipCannotOverrideExplicitIdentity(t *testing.T) {
	for _, test := range []struct {
		name    string
		mutate  func(*model.CustomerTransaction)
		unique  bool
		blocked bool
	}{
		{"legacy", func(r *model.CustomerTransaction) {
			r.SourceId = ""
			r.SourceType = ""
			r.TransactionType = ""
			r.Type = "应收账款"
		}, true, false},
		{"duplicate-code", func(r *model.CustomerTransaction) { r.SourceId = ""; r.SourceType = "" }, false, true},
		{"other-id", func(r *model.CustomerTransaction) { r.SourceId = primitive.NewObjectID().Hex() }, true, true},
		{"other-customer", func(r *model.CustomerTransaction) { r.CustomerId = primitive.NewObjectID().Hex() }, true, true},
		{"other-source", func(r *model.CustomerTransaction) { r.SourceType = code.SourceTypeManualAdjustment }, true, true},
		{"wrong-source-code", func(r *model.CustomerTransaction) { r.SourceCode = "O-WRONG" }, true, true},
		{"source-code-only", func(r *model.CustomerTransaction) { r.SourceId = ""; r.OrderCode = "" }, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := removalUnitSnapshot()
			test.mutate(&snapshot.Transactions[0])
			selected, err := selectRemovalTransactions(snapshot.Order, snapshot.Transactions, test.unique)
			if (err != nil) != test.blocked {
				t.Fatalf("selected=%v err=%v", selected, err)
			}
			if !test.blocked && len(selected) != 1 {
				t.Fatal("legacy row not selected")
			}
		})
	}
}

func TestRemovalInventoryAggregatesAndRejectsMissingOrMismatchedAllocations(t *testing.T) {
	snapshot := removalUnitSnapshot()
	snapshot.Order.Status = "待拣货"
	snapshot.Inventories[0].LockedQuantity = 2
	snapshot.Materials[0].Inventorys = []model.OutboundMaterialInventory{
		{InventoryId: snapshot.Inventories[0].Id.Hex(), ShipmentQuantity: 0.5},
		{InventoryId: snapshot.Inventories[0].Id.Hex(), ShipmentQuantity: 1.5},
	}
	changes, err := removalInventoryChanges(snapshot.Order, snapshot.Materials, snapshot.Inventories)
	if err != nil || len(changes) != 1 || changes[0].Release != 2 {
		t.Fatalf("changes=%+v err=%v", changes, err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*removalSnapshot)
	}{
		{"missing", func(s *removalSnapshot) { s.Inventories = nil }},
		{"unit", func(s *removalSnapshot) { s.Inventories[0].Unit = "箱" }},
		{"material", func(s *removalSnapshot) { s.Inventories[0].MaterialId = "other" }},
		{"quantity", func(s *removalSnapshot) { s.Materials[0].Quantity = 3 }},
		{"locked-floor", func(s *removalSnapshot) { s.Order.Status = "待拣货"; s.Inventories[0].LockedQuantity = 1 }},
		{"empty-allocation", func(s *removalSnapshot) { s.Materials[0].Inventorys = nil }},
		{"bad-id", func(s *removalSnapshot) { s.Materials[0].Inventorys[0].InventoryId = "bad" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := removalUnitSnapshot()
			test.mutate(&s)
			if _, err := removalInventoryChanges(s.Order, s.Materials, s.Inventories); err == nil {
				t.Fatal("invalid inventory accepted")
			}
		})
	}
}

func TestRemovalTokenBindsCustomerPaymentsPricesAndStock(t *testing.T) {
	base := removalUnitSnapshot()
	token, err := removalPreviewToken(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*removalSnapshot)
	}{
		{"price", func(s *removalSnapshot) { s.Materials[0].Price = 51 }},
		{"status", func(s *removalSnapshot) { s.Order.Status = "已出库" }},
		{"stock", func(s *removalSnapshot) { s.Inventories[0].AvailableQuantity = 7 }},
		{"payment", func(s *removalSnapshot) {
			s.Transactions = append(s.Transactions, model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: s.Order.CustomerId, TransactionType: code.TransactionTypePayment, Amount: 1})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := base
			s.Materials = append([]model.OutboundOrderMaterial(nil), base.Materials...)
			s.Inventories = append([]model.Inventory(nil), base.Inventories...)
			s.Transactions = append([]model.CustomerTransaction(nil), base.Transactions...)
			test.mutate(&s)
			changed, err := removalPreviewToken(s)
			if err != nil || changed == token {
				t.Fatalf("unchanged token: %v", err)
			}
		})
	}
	if len(token) != 64 || strings.TrimSpace(token) != token {
		t.Fatal("invalid token")
	}
}

func TestRemovalKeepsExplicitIndependentManualARAdjustment(t *testing.T) {
	snapshot := removalUnitSnapshot()
	manual := snapshot.Transactions[0]
	manual.Id = primitive.NewObjectID()
	manual.TransactionType = code.TransactionTypeARAdjustment
	manual.Direction = code.TransactionDirectionDecrease
	manual.Amount = 20
	manual.SourceType = code.SourceTypeManualAdjustment
	manual.SourceId = "manual-adjustment:independent"
	manual.SourceCode = ""
	snapshot.Transactions = append(snapshot.Transactions, manual)
	plan, err := prepareRemovalPlan(snapshot, true)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Preview.TransactionCount != 1 || plan.Preview.ReceivableReduction != 100 || plan.Preview.CreditAfter != 20 {
		t.Fatalf("independent manual adjustment removed: %+v", plan.Preview)
	}
}

func TestRemovalRejectsAdjustmentLinkedToDifferentOriginalTransaction(t *testing.T) {
	snapshot := removalUnitSnapshot()
	adjustment := snapshot.Transactions[0]
	adjustment.Id = primitive.NewObjectID()
	adjustment.TransactionType = code.TransactionTypeARAdjustment
	adjustment.Direction = code.TransactionDirectionDecrease
	adjustment.Amount = 20
	adjustment.OriginalTransactionId = primitive.NewObjectID().Hex()
	snapshot.Transactions = append(snapshot.Transactions, adjustment)
	if _, err := prepareRemovalPlan(snapshot, true); err == nil {
		t.Fatal("accepted adjustment pointing outside this order's ledger")
	}
}

func TestRemovalAdjustmentOriginalChainValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		link    string
		blocked bool
	}{
		{"anchored-chain", "chain", false}, {"legacy-empty", "empty", false}, {"self", "self", true}, {"cycle", "cycle", true}, {"missing-anchor", "missing", true}, {"independent-parent", "manual", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := removalUnitSnapshot()
			first := snapshot.Transactions[0]
			first.Id = primitive.NewObjectID()
			first.TransactionType = code.TransactionTypeARAdjustment
			first.Direction = code.TransactionDirectionIncrease
			first.Amount = 10
			first.OriginalTransactionId = snapshot.Transactions[0].Id.Hex()
			second := first
			second.Id = primitive.NewObjectID()
			second.Direction = code.TransactionDirectionDecrease
			second.Amount = 5
			second.OriginalTransactionId = first.Id.Hex()
			switch test.link {
			case "empty":
				first.OriginalTransactionId = ""
				second.OriginalTransactionId = ""
			case "self":
				first.OriginalTransactionId = first.Id.Hex()
			case "cycle":
				first.OriginalTransactionId = second.Id.Hex()
			case "missing":
				first.OriginalTransactionId = primitive.NewObjectID().Hex()
			case "manual":
				first.SourceType = code.SourceTypeManualAdjustment
				first.SourceId = "manual:independent"
			}
			snapshot.Transactions = append(snapshot.Transactions, first, second)
			_, err := prepareRemovalPlan(snapshot, true)
			if (err != nil) != test.blocked {
				t.Fatalf("blocked=%v err=%v", test.blocked, err)
			}
		})
	}
	// The historical original-ID field on the initial AR does not change its
	// identity; only adjustment chains require a valid original AR anchor.
	snapshot := removalUnitSnapshot()
	snapshot.Transactions[0].OriginalTransactionId = "historical-unused"
	if _, err := prepareRemovalPlan(snapshot, true); err != nil {
		t.Fatal(err)
	}
}
