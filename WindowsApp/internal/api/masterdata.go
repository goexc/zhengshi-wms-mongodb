package api

import (
	"context"
	"net/http"
	"strings"
)

type SupplierRequest struct {
	ID                            string `json:"id,omitempty"`
	Type                          string `json:"type"`
	Level                         int    `json:"level"`
	Code                          string `json:"code"`
	Image                         string `json:"image,omitempty"`
	Name                          string `json:"name"`
	LegalRepresentative           string `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string `json:"unified_social_credit_identifier"`
	Contact                       string `json:"contact"`
	Manager                       string `json:"manager"`
	Email                         string `json:"email,omitempty"`
	Address                       string `json:"address,omitempty"`
	Remark                        string `json:"remark,omitempty"`
}

type CustomerRequest struct {
	ID                            string  `json:"id,omitempty"`
	Type                          string  `json:"type"`
	Code                          string  `json:"code"`
	Name                          string  `json:"name"`
	Image                         string  `json:"image,omitempty"`
	LegalRepresentative           string  `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string  `json:"unified_social_credit_identifier"`
	Address                       string  `json:"address,omitempty"`
	Contact                       string  `json:"contact"`
	Manager                       string  `json:"manager"`
	Email                         string  `json:"email,omitempty"`
	Remark                        string  `json:"remark,omitempty"`
	ReceivableBalance             float64 `json:"receivable_balance,omitempty"`
}

type CarrierRequest struct {
	ID                            string `json:"id,omitempty"`
	Type                          string `json:"type"`
	Code                          string `json:"code"`
	Name                          string `json:"name"`
	Image                         string `json:"image,omitempty"`
	LegalRepresentative           string `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string `json:"unified_social_credit_identifier"`
	Address                       string `json:"address,omitempty"`
	Contact                       string `json:"contact"`
	Manager                       string `json:"manager"`
	Email                         string `json:"email,omitempty"`
	Remark                        string `json:"remark,omitempty"`
}

type PartnerStatusRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type WarehouseRequest struct {
	ID           string  `json:"id,omitempty"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Image        string  `json:"image,omitempty"`
	Address      string  `json:"address,omitempty"`
	Capacity     float64 `json:"capacity,omitempty"`
	CapacityUnit string  `json:"capacity_unit,omitempty"`
	Manager      string  `json:"manager,omitempty"`
	Contact      string  `json:"contact,omitempty"`
	Remark       string  `json:"remark,omitempty"`
}

type WarehouseZoneRequest struct {
	ID           string  `json:"id,omitempty"`
	WarehouseID  string  `json:"warehouse_id"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Image        string  `json:"image"`
	Capacity     float64 `json:"capacity,omitempty"`
	CapacityUnit string  `json:"capacity_unit,omitempty"`
	Manager      string  `json:"manager,omitempty"`
	Contact      string  `json:"contact,omitempty"`
	Remark       string  `json:"remark,omitempty"`
}

type WarehouseRackRequest struct {
	ID              string  `json:"id,omitempty"`
	WarehouseZoneID string  `json:"warehouse_zone_id"`
	Type            string  `json:"type"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	Image           string  `json:"image"`
	Capacity        float64 `json:"capacity,omitempty"`
	CapacityUnit    string  `json:"capacity_unit,omitempty"`
	Manager         string  `json:"manager,omitempty"`
	Contact         string  `json:"contact,omitempty"`
	Remark          string  `json:"remark,omitempty"`
}

type WarehouseBinRequest struct {
	ID              string  `json:"id,omitempty"`
	WarehouseRackID string  `json:"warehouse_rack_id"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	Image           string  `json:"image,omitempty"`
	Capacity        float64 `json:"capacity,omitempty"`
	CapacityUnit    string  `json:"capacity_unit,omitempty"`
	Manager         string  `json:"manager,omitempty"`
	Contact         string  `json:"contact,omitempty"`
	Remark          string  `json:"remark,omitempty"`
}

type WarehouseStatusRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *Client) ChangePassword(ctx context.Context, password string) error {
	return c.do(ctx, http.MethodPatch, "/account/password", nil, map[string]string{
		"password": password,
	}, nil)
}

func (c *Client) CreateSupplier(ctx context.Context, request SupplierRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/supplier", nil, request, nil)
}

func (c *Client) UpdateSupplier(ctx context.Context, request SupplierRequest) error {
	return c.do(ctx, http.MethodPut, "/supplier", nil, request, nil)
}

func (c *Client) UpdateSupplierStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/supplier/status", nil, PartnerStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateCustomer(ctx context.Context, request CustomerRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/customer", nil, request, nil)
}

func (c *Client) UpdateCustomer(ctx context.Context, request CustomerRequest) error {
	return c.do(ctx, http.MethodPut, "/customer", nil, request, nil)
}

func (c *Client) UpdateCustomerStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/customer/status", nil, PartnerStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateCarrier(ctx context.Context, request CarrierRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/carrier", nil, request, nil)
}

func (c *Client) UpdateCarrier(ctx context.Context, request CarrierRequest) error {
	return c.do(ctx, http.MethodPut, "/carrier", nil, request, nil)
}

func (c *Client) UpdateCarrierStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/carrier/status", nil, PartnerStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateWarehouse(ctx context.Context, request WarehouseRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/warehouse", nil, request, nil)
}

func (c *Client) UpdateWarehouse(ctx context.Context, request WarehouseRequest) error {
	return c.do(ctx, http.MethodPut, "/warehouse", nil, request, nil)
}

func (c *Client) UpdateWarehouseStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/warehouse/status", nil, WarehouseStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateWarehouseZone(ctx context.Context, request WarehouseZoneRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/warehouse_zone", nil, request, nil)
}

func (c *Client) UpdateWarehouseZone(ctx context.Context, request WarehouseZoneRequest) error {
	return c.do(ctx, http.MethodPut, "/warehouse_zone", nil, request, nil)
}

func (c *Client) UpdateWarehouseZoneStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/warehouse_zone/status", nil, WarehouseStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateWarehouseRack(ctx context.Context, request WarehouseRackRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/warehouse_rack", nil, request, nil)
}

func (c *Client) UpdateWarehouseRack(ctx context.Context, request WarehouseRackRequest) error {
	return c.do(ctx, http.MethodPut, "/warehouse_rack", nil, request, nil)
}

func (c *Client) UpdateWarehouseRackStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/warehouse_rack/status", nil, WarehouseStatusRequest{ID: id, Status: status}, nil)
}

func (c *Client) CreateWarehouseBin(ctx context.Context, request WarehouseBinRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/warehouse_bin", nil, request, nil)
}

func (c *Client) UpdateWarehouseBin(ctx context.Context, request WarehouseBinRequest) error {
	return c.do(ctx, http.MethodPut, "/warehouse_bin", nil, request, nil)
}

func (c *Client) UpdateWarehouseBinStatus(ctx context.Context, id, status string) error {
	return c.do(ctx, http.MethodPatch, "/warehouse_bin/status", nil, WarehouseStatusRequest{ID: id, Status: status}, nil)
}

func sameID(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func (c *Client) FindSupplier(ctx context.Context, id, code string) (Supplier, bool, error) {
	page, err := c.SupplierDirectory(ctx, 1, 100, PartnerFilters{Code: code})
	if err != nil {
		return Supplier{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return Supplier{}, false, nil
}

func (c *Client) FindCustomer(ctx context.Context, id, code string) (Customer, bool, error) {
	page, err := c.CustomerDirectory(ctx, 1, 100, PartnerFilters{Code: code})
	if err != nil {
		return Customer{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return Customer{}, false, nil
}

func (c *Client) FindCarrier(ctx context.Context, id, code string) (Carrier, bool, error) {
	page, err := c.CarrierDirectory(ctx, 1, 100, PartnerFilters{Code: code})
	if err != nil {
		return Carrier{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return Carrier{}, false, nil
}

func (c *Client) FindWarehouse(ctx context.Context, id, code string) (Warehouse, bool, error) {
	page, err := c.Warehouses(ctx, 1, 100, WarehouseFilters{Code: code})
	if err != nil {
		return Warehouse{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return Warehouse{}, false, nil
}

func (c *Client) FindWarehouseZone(ctx context.Context, id, code string) (WarehouseZone, bool, error) {
	page, err := c.WarehouseZones(ctx, 1, 100, WarehouseFilters{Code: code})
	if err != nil {
		return WarehouseZone{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return WarehouseZone{}, false, nil
}

func (c *Client) FindWarehouseRack(ctx context.Context, id, code string) (WarehouseRack, bool, error) {
	page, err := c.WarehouseRacks(ctx, 1, 100, WarehouseFilters{Code: code})
	if err != nil {
		return WarehouseRack{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return WarehouseRack{}, false, nil
}

func (c *Client) FindWarehouseBin(ctx context.Context, id, code string) (WarehouseBin, bool, error) {
	page, err := c.WarehouseBins(ctx, 1, 100, WarehouseFilters{Code: code})
	if err != nil {
		return WarehouseBin{}, false, err
	}
	for _, item := range page.List {
		if sameID(item.ID, id) {
			return item, true, nil
		}
	}
	return WarehouseBin{}, false, nil
}
