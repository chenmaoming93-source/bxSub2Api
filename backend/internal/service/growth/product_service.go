package growth

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthproduct"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	"github.com/Wei-Shaw/sub2api/ent/growthproductversion"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/google/uuid"
)

var (
	ErrProductCodeDuplicated = infraerrors.Conflict("PRODUCT_CODE_DUPLICATED", "product_code already exists")
	ErrGrowthProductNotFound = infraerrors.NotFound("PRODUCT_NOT_FOUND", "growth product not found")
	ErrProductStateInvalid   = infraerrors.Conflict("PRODUCT_STATE_INVALID", "product state does not allow this operation")
	ErrProductPublishInvalid = infraerrors.BadRequest("PRODUCT_PUBLISH_INVALID", "product configuration is not publishable")
	ErrInventoryUnderflow    = infraerrors.Conflict("INVENTORY_UNDERFLOW", "inventory cannot be negative")
)

type ProductVersionInput struct {
	PointsCost     int64          `json:"points_cost"`
	BenefitSummary string         `json:"benefit_summary"`
	BenefitDetail  map[string]any `json:"benefit_detail"`
	TokenAmount    int64          `json:"token_amount"`
	PerUserLimit   int            `json:"per_user_limit"`
}

type ProductCreateInput struct {
	OperatorID      int64               `json:"-"`
	ProductCode     string              `json:"product_code"`
	Title           string              `json:"title"`
	ProductType     string              `json:"product_type"`
	Icon            string              `json:"icon"`
	FulfillmentMode string              `json:"fulfillment_mode"`
	SortOrder       int                 `json:"sort_order"`
	StockMode       string              `json:"stock_mode"`
	InitialStock    int64               `json:"initial_stock"`
	Version         ProductVersionInput `json:"version"`
	RequestID       string              `json:"request_id"`
}

type ProductUpdateInput struct {
	OperatorID      int64               `json:"-"`
	Title           string              `json:"title"`
	Icon            string              `json:"icon"`
	FulfillmentMode string              `json:"fulfillment_mode"`
	SortOrder       int                 `json:"sort_order"`
	Version         ProductVersionInput `json:"version"`
	RequestID       string              `json:"request_id"`
}

type ProductListView struct {
	ProductID             int64          `json:"product_id"`
	ProductCode           string         `json:"product_code"`
	Title                 string         `json:"title"`
	ProductType           string         `json:"product_type"`
	Icon                  string         `json:"icon"`
	FulfillmentMode       string         `json:"fulfillment_mode"`
	SortOrder             int            `json:"sort_order"`
	Status                string         `json:"status"`
	CurrentVersion        int            `json:"current_version"`
	PendingVersion        *int           `json:"pending_version,omitempty"`
	PointsCost            int64          `json:"points_cost"`
	BenefitSummary        string         `json:"benefit_summary"`
	BenefitDetail         map[string]any `json:"benefit_detail,omitempty"`
	TokenAmount           int64          `json:"token_amount"`
	PerUserLimit          int            `json:"per_user_limit"`
	StockMode             string         `json:"stock_mode"`
	Stock                 int64          `json:"stock"`
	Reserved              int64          `json:"reserved"`
	Sold                  int64          `json:"sold"`
	TokenQuotaTotal       int64          `json:"token_quota_total"`
	TokenQuotaDistributed int64          `json:"token_quota_distributed"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type ProductVersionView struct {
	ID             int64          `json:"id"`
	Version        int            `json:"version"`
	PointsCost     int64          `json:"points_cost"`
	BenefitSummary string         `json:"benefit_summary"`
	BenefitDetail  map[string]any `json:"benefit_detail,omitempty"`
	TokenAmount    int64          `json:"token_amount"`
	PerUserLimit   int            `json:"per_user_limit"`
	CreatedBy      int64          `json:"created_by"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	EffectiveFrom  *time.Time     `json:"effective_from,omitempty"`
	EffectiveTo    *time.Time     `json:"effective_to,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

type StockAdjustment struct {
	OperatorID int64  `json:"-"`
	Delta      int64  `json:"delta"`
	Reason     string `json:"reason"`
	RequestID  string `json:"request_id"`
}

type ProductService struct {
	client *ent.Client
}

func NewProductService(client *ent.Client) *ProductService {
	return &ProductService{client: client}
}

func (s *ProductService) Create(ctx context.Context, input ProductCreateInput) (ProductListView, error) {
	if err := validateProductCreate(input); err != nil {
		return ProductListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	product, err := tx.GrowthProduct.Create().
		SetProductCode(strings.TrimSpace(input.ProductCode)).
		SetTitle(strings.TrimSpace(input.Title)).
		SetProductType(input.ProductType).
		SetIcon(strings.TrimSpace(input.Icon)).
		SetFulfillmentMode(input.FulfillmentMode).
		SetSortOrder(input.SortOrder).
		SetProductStatus("DRAFT").
		SetCurrentVersion(0).
		Save(ctx)
	if err != nil {
		if isDuplicateKey(err) {
			return ProductListView{}, ErrProductCodeDuplicated
		}
		return ProductListView{}, err
	}
	version, err := createProductVersion(ctx, tx, product.ID, 1, input.OperatorID, input.Version)
	if err != nil {
		return ProductListView{}, err
	}
	available := input.InitialStock
	if input.StockMode == "UNLIMITED" {
		available = 0
	}
	inventory, err := tx.GrowthProductInventory.Create().
		SetProductID(product.ID).
		SetStockMode(input.StockMode).
		SetAvailable(available).
		SetLockVersion(1).
		Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	if err := appendProductAudit(ctx, tx, input.OperatorID, "product.create", product.ID, nil, product, input.RequestID, "创建兑换商品"); err != nil {
		return ProductListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductListView{}, err
	}
	committed = true
	return buildProductView(product, nil, version, inventory), nil
}

func (s *ProductService) Update(ctx context.Context, productID int64, input ProductUpdateInput) (ProductListView, error) {
	if productID <= 0 {
		return ProductListView{}, infraerrors.BadRequest("INVALID_PRODUCT_ID", "product_id must be positive")
	}
	if input.OperatorID <= 0 {
		return ProductListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateProductVersion(input.Version); err != nil {
		return ProductListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	product, err := tx.GrowthProduct.Query().Where(growthproduct.ID(productID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	before := *product
	product, err = tx.GrowthProduct.UpdateOneID(productID).
		SetTitle(strings.TrimSpace(input.Title)).
		SetIcon(strings.TrimSpace(input.Icon)).
		SetFulfillmentMode(input.FulfillmentMode).
		SetSortOrder(input.SortOrder).
		Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	draft, err := latestDraftProductVersionTx(ctx, tx, productID)
	if ent.IsNotFound(err) {
		latest, latestErr := latestProductVersionTx(ctx, tx, productID)
		if latestErr != nil {
			return ProductListView{}, latestErr
		}
		next := 1
		if latest != nil {
			next = latest.Version + 1
		}
		draft, err = createProductVersion(ctx, tx, productID, next, input.OperatorID, input.Version)
	} else if err == nil {
		update := tx.GrowthProductVersion.UpdateOneID(draft.ID).
			SetPointsCost(input.Version.PointsCost).
			SetBenefitSummary(strings.TrimSpace(input.Version.BenefitSummary)).
			SetTokenAmount(input.Version.TokenAmount).
			SetPerUserLimit(input.Version.PerUserLimit)
		update.SetBenefitDetail(normalizeJSONMap(input.Version.BenefitDetail))
		draft, err = update.Save(ctx)
	}
	if err != nil {
		return ProductListView{}, err
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(productID)).
		Only(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	if err := appendProductAudit(ctx, tx, input.OperatorID, "product.update", productID, &before, product, input.RequestID, "更新商品版本"); err != nil {
		return ProductListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductListView{}, err
	}
	committed = true
	current, _ := currentProductVersion(ctx, s.client, productID)
	return buildProductView(product, current, draft, inventory), nil
}

func (s *ProductService) Publish(ctx context.Context, productID int64, operatorID int64, requestID string) (ProductListView, error) {
	if operatorID <= 0 {
		return ProductListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	product, err := tx.GrowthProduct.Query().Where(growthproduct.ID(productID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	draft, err := latestDraftProductVersionTx(ctx, tx, productID)
	if ent.IsNotFound(err) {
		if product.ProductStatus == "ACTIVE" {
			current, currentErr := currentProductVersionTx(ctx, tx, productID)
			if currentErr == nil {
				inventory, inventoryErr := tx.GrowthProductInventory.Query().
					Where(growthproductinventory.ProductIDEQ(productID)).
					Only(ctx)
				if inventoryErr != nil {
					return ProductListView{}, inventoryErr
				}
				if rollbackErr := tx.Rollback(); rollbackErr != nil {
					return ProductListView{}, rollbackErr
				}
				return buildProductView(product, current, nil, inventory), nil
			}
		}
		return ProductListView{}, ErrProductPublishInvalid
	}
	if err != nil {
		return ProductListView{}, err
	}
	if product.ProductType == "TOKEN_QUOTA" && draft.TokenAmount <= 0 {
		return ProductListView{}, ErrProductPublishInvalid
	}
	now := time.Now()
	if _, err := tx.GrowthProductVersion.Update().
		Where(
			growthproductversion.ProductIDEQ(productID),
			growthproductversion.PublishedAtNotNil(),
			growthproductversion.EffectiveToIsNil(),
		).
		SetEffectiveTo(now).
		Save(ctx); err != nil {
		return ProductListView{}, err
	}
	published, err := tx.GrowthProductVersion.UpdateOneID(draft.ID).
		SetPublishedAt(now).
		SetEffectiveFrom(now).
		ClearEffectiveTo().
		Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	before := *product
	product, err = tx.GrowthProduct.UpdateOneID(productID).
		SetCurrentVersion(published.Version).
		SetProductStatus("ACTIVE").
		Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(productID)).
		Only(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	if err := appendProductAudit(ctx, tx, operatorID, "product.publish", productID, &before, product, requestID, "发布商品版本"); err != nil {
		return ProductListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductListView{}, err
	}
	committed = true
	return buildProductView(product, published, nil, inventory), nil
}

func (s *ProductService) AdjustStock(ctx context.Context, productID int64, adjustment StockAdjustment) (ProductListView, error) {
	if productID <= 0 {
		return ProductListView{}, infraerrors.BadRequest("INVALID_PRODUCT_ID", "product_id must be positive")
	}
	if adjustment.OperatorID <= 0 {
		return ProductListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if adjustment.Delta == 0 {
		return ProductListView{}, infraerrors.BadRequest("INVALID_STOCK_DELTA", "stock delta cannot be zero")
	}
	if err := validateReason(adjustment.Reason); err != nil {
		return ProductListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	product, err := tx.GrowthProduct.Get(ctx, productID)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(productID)).
		ForUpdate().
		Only(ctx)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	beforeInventory := inventory
	if inventory.StockMode == "LIMITED" && (adjustment.Delta < 0 && inventory.Available < -adjustment.Delta) {
		return ProductListView{}, ErrInventoryUnderflow
	}
	nextAvailable := inventory.Available
	if inventory.StockMode == "LIMITED" {
		nextAvailable += adjustment.Delta
	}
	inventory, err = tx.GrowthProductInventory.UpdateOneID(inventory.ID).
		SetAvailable(nextAvailable).
		SetLockVersion(inventory.LockVersion + 1).
		Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	businessKey := strings.TrimSpace(adjustment.RequestID)
	if businessKey == "" {
		businessKey = "stock:" + uuid.NewString()
	}
	operation := "ADJUST"
	if adjustment.Delta > 0 {
		operation = "STOCK_IN"
	}
	_, err = tx.GrowthInventoryLedger.Create().
		SetBusinessKey(businessKey).
		SetProductID(productID).
		SetInventoryID(inventory.ID).
		SetOperationType(operation).
		SetDeltaAvailable(adjustment.Delta).
		SetAvailableAfter(inventory.Available).
		SetReservedAfter(inventory.Reserved).
		SetSoldAfter(inventory.Sold).
		SetOperatorID(adjustment.OperatorID).
		SetReason(strings.TrimSpace(adjustment.Reason)).
		Save(ctx)
	if err != nil {
		if isDuplicateKey(err) {
			return ProductListView{}, infraerrors.Conflict("INVENTORY_REQUEST_REPLAYED", "stock request was already processed")
		}
		return ProductListView{}, err
	}
	if err := appendProductInventoryAudit(ctx, tx, adjustment.OperatorID, productID, beforeInventory, inventory, adjustment.RequestID, adjustment.Reason); err != nil {
		return ProductListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductListView{}, err
	}
	committed = true
	current, _ := currentProductVersion(ctx, s.client, productID)
	draft, _ := latestDraftProductVersionClient(ctx, s.client, productID)
	return buildProductView(product, current, draft, inventory), nil
}

func (s *ProductService) Offline(ctx context.Context, productID int64, operatorID int64, reason string, requestID string) (ProductListView, error) {
	if operatorID <= 0 {
		return ProductListView{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateReason(reason); err != nil {
		return ProductListView{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	product, err := tx.GrowthProduct.Query().Where(growthproduct.ID(productID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	if product.ProductStatus == "OFFLINE" {
		return ProductListView{}, ErrProductStateInvalid
	}
	before := *product
	product, err = tx.GrowthProduct.UpdateOneID(productID).SetProductStatus("OFFLINE").Save(ctx)
	if err != nil {
		return ProductListView{}, err
	}
	if err := appendProductAudit(ctx, tx, operatorID, "product.offline", productID, &before, product, requestID, reason); err != nil {
		return ProductListView{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductListView{}, err
	}
	committed = true
	inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(productID)).Only(ctx)
	current, _ := currentProductVersion(ctx, s.client, productID)
	draft, _ := latestDraftProductVersionClient(ctx, s.client, productID)
	return buildProductView(product, current, draft, inventory), nil
}

func (s *ProductService) List(ctx context.Context, filter BudgetListFilter, productType string) ([]ProductListView, int64, error) {
	filter = normalizeBudgetFilter(filter)
	query := s.client.GrowthProduct.Query()
	if filter.Status != "" {
		query = query.Where(growthproduct.ProductStatusEQ(filter.Status))
	}
	if productType != "" {
		query = query.Where(growthproduct.ProductTypeEQ(productType))
	}
	if filter.Search != "" {
		query = query.Where(growthproduct.Or(
			growthproduct.ProductCodeContainsFold(filter.Search),
			growthproduct.TitleContainsFold(filter.Search),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	products, err := query.
		Order(ent.Asc(growthproduct.FieldSortOrder), ent.Desc(growthproduct.FieldID)).
		Offset((filter.Page - 1) * filter.PageSize).
		Limit(filter.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	current, pending, err := s.productVersionMaps(ctx, products)
	if err != nil {
		return nil, 0, err
	}
	inventories, err := s.inventoryMap(ctx, products)
	if err != nil {
		return nil, 0, err
	}
	result := make([]ProductListView, 0, len(products))
	for _, product := range products {
		result = append(result, buildProductView(product, current[product.ID], pending[product.ID], inventories[product.ID]))
	}
	return result, int64(total), nil
}

func (s *ProductService) Get(ctx context.Context, productID int64) (ProductListView, error) {
	product, err := s.client.GrowthProduct.Get(ctx, productID)
	if ent.IsNotFound(err) {
		return ProductListView{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ProductListView{}, err
	}
	current, _ := currentProductVersion(ctx, s.client, productID)
	draft, _ := latestDraftProductVersionClient(ctx, s.client, productID)
	inventory, _ := s.client.GrowthProductInventory.Query().Where(growthproductinventory.ProductIDEQ(productID)).Only(ctx)
	return buildProductView(product, current, draft, inventory), nil
}

func (s *ProductService) Versions(ctx context.Context, productID int64) ([]ProductVersionView, error) {
	versions, err := s.client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID)).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ProductVersionView, 0, len(versions))
	for _, version := range versions {
		result = append(result, productVersionView(version))
	}
	return result, nil
}

func createProductVersion(ctx context.Context, tx *ent.Tx, productID int64, version int, operatorID int64, input ProductVersionInput) (*ent.GrowthProductVersion, error) {
	create := tx.GrowthProductVersion.Create().
		SetProductID(productID).
		SetVersion(version).
		SetPointsCost(input.PointsCost).
		SetBenefitSummary(strings.TrimSpace(input.BenefitSummary)).
		SetTokenAmount(input.TokenAmount).
		SetPerUserLimit(input.PerUserLimit).
		SetCreatedBy(operatorID)
	create.SetBenefitDetail(normalizeJSONMap(input.BenefitDetail))
	return create.Save(ctx)
}

func latestDraftProductVersionTx(ctx context.Context, tx *ent.Tx, productID int64) (*ent.GrowthProductVersion, error) {
	return tx.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID), growthproductversion.PublishedAtIsNil()).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		First(ctx)
}

func latestProductVersionTx(ctx context.Context, tx *ent.Tx, productID int64) (*ent.GrowthProductVersion, error) {
	version, err := tx.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID)).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return version, err
}

func currentProductVersion(ctx context.Context, client *ent.Client, productID int64) (*ent.GrowthProductVersion, error) {
	return client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID), growthproductversion.PublishedAtNotNil()).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		First(ctx)
}

func currentProductVersionTx(ctx context.Context, tx *ent.Tx, productID int64) (*ent.GrowthProductVersion, error) {
	return tx.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID), growthproductversion.PublishedAtNotNil()).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		First(ctx)
}

func latestDraftProductVersionClient(ctx context.Context, client *ent.Client, productID int64) (*ent.GrowthProductVersion, error) {
	return client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDEQ(productID), growthproductversion.PublishedAtIsNil()).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		First(ctx)
}

func (s *ProductService) productVersionMaps(ctx context.Context, products []*ent.GrowthProduct) (map[int64]*ent.GrowthProductVersion, map[int64]*ent.GrowthProductVersion, error) {
	ids := make([]int64, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}
	current := make(map[int64]*ent.GrowthProductVersion)
	pending := make(map[int64]*ent.GrowthProductVersion)
	if len(ids) == 0 {
		return current, pending, nil
	}
	versions, err := s.client.GrowthProductVersion.Query().
		Where(growthproductversion.ProductIDIn(ids...)).
		Order(ent.Desc(growthproductversion.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, version := range versions {
		if version.PublishedAt != nil {
			if _, exists := current[version.ProductID]; !exists {
				current[version.ProductID] = version
			}
		} else if _, exists := pending[version.ProductID]; !exists {
			pending[version.ProductID] = version
		}
	}
	return current, pending, nil
}

func (s *ProductService) inventoryMap(ctx context.Context, products []*ent.GrowthProduct) (map[int64]*ent.GrowthProductInventory, error) {
	ids := make([]int64, 0, len(products))
	for _, product := range products {
		ids = append(ids, product.ID)
	}
	result := make(map[int64]*ent.GrowthProductInventory, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	inventories, err := s.client.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, inventory := range inventories {
		result[inventory.ProductID] = inventory
	}
	return result, nil
}

func buildProductView(product *ent.GrowthProduct, current, pending *ent.GrowthProductVersion, inventory *ent.GrowthProductInventory) ProductListView {
	view := ProductListView{
		ProductID:       product.ID,
		ProductCode:     product.ProductCode,
		Title:           product.Title,
		ProductType:     product.ProductType,
		Icon:            product.Icon,
		FulfillmentMode: product.FulfillmentMode,
		SortOrder:       product.SortOrder,
		Status:          product.ProductStatus,
		CurrentVersion:  product.CurrentVersion,
		UpdatedAt:       product.UpdatedAt,
	}
	if current != nil {
		view.PointsCost = current.PointsCost
		view.BenefitSummary = current.BenefitSummary
		view.BenefitDetail = normalizeJSONMap(current.BenefitDetail)
		view.TokenAmount = current.TokenAmount
		view.PerUserLimit = current.PerUserLimit
	}
	if pending != nil {
		version := pending.Version
		view.PendingVersion = &version
	}
	if inventory != nil {
		view.StockMode = inventory.StockMode
		if inventory.StockMode == "UNLIMITED" {
			view.Stock = -1
		} else {
			view.Stock = inventory.Available
		}
		view.Reserved = inventory.Reserved
		view.Sold = inventory.Sold
		view.TokenQuotaTotal = inventory.TokenQuotaTotal
		view.TokenQuotaDistributed = inventory.TokenQuotaDistributed
	}
	return view
}

func productVersionView(version *ent.GrowthProductVersion) ProductVersionView {
	return ProductVersionView{
		ID:             version.ID,
		Version:        version.Version,
		PointsCost:     version.PointsCost,
		BenefitSummary: version.BenefitSummary,
		BenefitDetail:  normalizeJSONMap(version.BenefitDetail),
		TokenAmount:    version.TokenAmount,
		PerUserLimit:   version.PerUserLimit,
		CreatedBy:      version.CreatedBy,
		PublishedAt:    version.PublishedAt,
		EffectiveFrom:  version.EffectiveFrom,
		EffectiveTo:    version.EffectiveTo,
		CreatedAt:      version.CreatedAt,
	}
}

func validateProductCreate(input ProductCreateInput) error {
	if input.OperatorID <= 0 {
		return infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if strings.TrimSpace(input.ProductCode) == "" || strings.TrimSpace(input.Title) == "" {
		return infraerrors.BadRequest("INVALID_PRODUCT", "product_code and title are required")
	}
	if !isProductType(input.ProductType) {
		return infraerrors.BadRequest("INVALID_PRODUCT", "product_type is invalid")
	}
	if input.FulfillmentMode != "AUTOMATIC" && input.FulfillmentMode != "MANUAL" {
		return infraerrors.BadRequest("INVALID_PRODUCT", "fulfillment_mode is invalid")
	}
	if input.StockMode != "LIMITED" && input.StockMode != "UNLIMITED" {
		return infraerrors.BadRequest("INVALID_PRODUCT", "stock_mode is invalid")
	}
	if input.StockMode == "LIMITED" && input.InitialStock < 0 {
		return infraerrors.BadRequest("INVALID_STOCK", "initial_stock must be non-negative")
	}
	return validateProductVersion(input.Version)
}

func validateProductVersion(input ProductVersionInput) error {
	if input.PointsCost <= 0 {
		return infraerrors.BadRequest("INVALID_PRODUCT_VERSION", "points_cost must be positive")
	}
	if input.TokenAmount < 0 || input.PerUserLimit < 0 {
		return infraerrors.BadRequest("INVALID_PRODUCT_VERSION", "token amount and limits must be non-negative")
	}
	return nil
}

func isProductType(value string) bool {
	return value == "TOKEN_QUOTA" || value == "SUBSCRIPTION" || value == "PHYSICAL"
}

func appendProductAudit(ctx context.Context, tx *ent.Tx, operatorID int64, action string, productID int64, before, after *ent.GrowthProduct, requestID, reason string) error {
	beforeJSON := map[string]any{}
	if before != nil {
		beforeJSON = productAuditSnapshot(before)
	}
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(operatorID).
		SetActionType(action).
		SetResourceType("growth_product").
		SetResourceID(productID).
		SetBeforeJSON(normalizeJSONMap(beforeJSON)).
		SetAfterJSON(normalizeJSONMap(productAuditSnapshot(after))).
		SetReason(strings.TrimSpace(reason)).
		SetRequestID(strings.TrimSpace(requestID)).
		Save(ctx)
	return err
}

func appendProductInventoryAudit(ctx context.Context, tx *ent.Tx, operatorID int64, productID int64, before, after *ent.GrowthProductInventory, requestID, reason string) error {
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(operatorID).
		SetActionType("product.stock.adjust").
		SetResourceType("growth_product_inventory").
		SetResourceID(productID).
		SetBeforeJSON(normalizeJSONMap(productInventorySnapshot(before))).
		SetAfterJSON(normalizeJSONMap(productInventorySnapshot(after))).
		SetReason(strings.TrimSpace(reason)).
		SetRequestID(strings.TrimSpace(requestID)).
		Save(ctx)
	return err
}

func productAuditSnapshot(product *ent.GrowthProduct) map[string]any {
	if product == nil {
		return map[string]any{}
	}
	return map[string]any{
		"product_id":       product.ID,
		"product_code":     product.ProductCode,
		"title":            product.Title,
		"status":           product.ProductStatus,
		"current_version":  product.CurrentVersion,
		"fulfillment_mode": product.FulfillmentMode,
	}
}

func productInventorySnapshot(inventory *ent.GrowthProductInventory) map[string]any {
	if inventory == nil {
		return map[string]any{}
	}
	return map[string]any{
		"available":    inventory.Available,
		"reserved":     inventory.Reserved,
		"sold":         inventory.Sold,
		"stock_mode":   inventory.StockMode,
		"lock_version": inventory.LockVersion,
	}
}
