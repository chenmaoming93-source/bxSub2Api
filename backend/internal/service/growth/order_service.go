package growth

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthexchangeorder"
	"github.com/Wei-Shaw/sub2api/ent/growthfulfillmentjob"
	"github.com/Wei-Shaw/sub2api/ent/growthproduct"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	"github.com/Wei-Shaw/sub2api/ent/growthproductversion"
	"github.com/Wei-Shaw/sub2api/ent/growthtransfer"
	"github.com/Wei-Shaw/sub2api/ent/growthwallet"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/google/uuid"
)

var (
	ErrProductVersionChanged = infraerrors.Conflict("PRODUCT_VERSION_CHANGED", "product version changed")
	ErrProductOffline        = infraerrors.Conflict("PRODUCT_OFFLINE", "product is offline")
	ErrProductSoldOut        = infraerrors.Conflict("PRODUCT_SOLD_OUT", "product is sold out")
	ErrPurchaseLimitReached  = infraerrors.Conflict("PURCHASE_LIMIT_REACHED", "purchase limit reached")
	ErrOrderNotFound         = infraerrors.NotFound("ORDER_NOT_FOUND", "exchange order not found")
)

type ExchangeCommand struct {
	UserID          int64
	ProductID       int64
	ExpectedVersion *int
	RequestID       string
	IdempotencyKey  string
}

type ExchangeCreateResult struct {
	OrderID           int64  `json:"order_id"`
	OrderNo           string `json:"order_no"`
	Status            string `json:"status"`
	ProductVersion    int    `json:"product_version"`
	PointsCost        int64  `json:"points_cost"`
	TransferID        *int64 `json:"transfer_id,omitempty"`
	CurrentBalance    int64  `json:"current_balance"`
	FulfillmentStatus string `json:"fulfillment_status"`
	ServerTime        string `json:"server_time"`
}

type ExchangeOrderDetail struct {
	OrderID           int64          `json:"order_id"`
	OrderNo           string         `json:"order_no"`
	Status            string         `json:"status"`
	ProductSnapshot   map[string]any `json:"product_snapshot"`
	PointsCost        int64          `json:"points_cost"`
	TransferID        *int64         `json:"transfer_id,omitempty"`
	FulfillmentStatus string         `json:"fulfillment_status"`
	FailureReason     string         `json:"failure_reason"`
	RefundTransferID  *int64         `json:"refund_transfer_id,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	FulfilledAt       *time.Time     `json:"fulfilled_at,omitempty"`
}

type OrderService struct {
	client    *ent.Client
	transfers *TransferService
	now       func() time.Time
}

func NewOrderService(client *ent.Client, transfers *TransferService) *OrderService {
	return &OrderService{client: client, transfers: transfers, now: time.Now}
}

func (s *OrderService) Create(ctx context.Context, command ExchangeCommand) (ExchangeCreateResult, error) {
	if command.UserID <= 0 || command.ProductID <= 0 || strings.TrimSpace(command.IdempotencyKey) == "" {
		return ExchangeCreateResult{}, infraerrors.BadRequest("INVALID_REQUEST", "user, product and idempotency_key are required")
	}
	if existing, err := s.existingOrder(ctx, command.UserID, command.IdempotencyKey); err == nil {
		return s.createResult(ctx, existing)
	} else if !ent.IsNotFound(err) {
		return ExchangeCreateResult{}, err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if existing, err := tx.GrowthExchangeOrder.Query().
		Where(
			growthexchangeorder.UserIDEQ(command.UserID),
			growthexchangeorder.IdempotencyKeyEQ(command.IdempotencyKey),
		).
		Only(ctx); err == nil {
		_ = tx.Rollback()
		return s.createResult(ctx, existing)
	} else if !ent.IsNotFound(err) {
		return ExchangeCreateResult{}, err
	}

	product, err := tx.GrowthProduct.Query().Where(growthproduct.ID(command.ProductID)).Only(ctx)
	if ent.IsNotFound(err) {
		return ExchangeCreateResult{}, ErrGrowthProductNotFound
	}
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	if product.ProductStatus != "ACTIVE" {
		return ExchangeCreateResult{}, ErrProductOffline
	}
	if command.ExpectedVersion != nil && *command.ExpectedVersion != product.CurrentVersion {
		return ExchangeCreateResult{}, ErrProductVersionChanged
	}
	version, err := tx.GrowthProductVersion.Query().
		Where(
			growthproductversion.ProductIDEQ(product.ID),
			growthproductversion.VersionEQ(product.CurrentVersion),
			growthproductversion.PublishedAtNotNil(),
		).
		Only(ctx)
	if ent.IsNotFound(err) {
		return ExchangeCreateResult{}, ErrProductVersionChanged
	}
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(product.ID)).
		ForUpdate().
		Only(ctx)
	if ent.IsNotFound(err) {
		return ExchangeCreateResult{}, ErrProductSoldOut
	}
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	if inventory.StockMode == "LIMITED" && inventory.Available <= 0 {
		return ExchangeCreateResult{}, ErrProductSoldOut
	}
	if version.PerUserLimit > 0 {
		count, err := tx.GrowthExchangeOrder.Query().
			Where(
				growthexchangeorder.UserIDEQ(command.UserID),
				growthexchangeorder.ProductIDEQ(product.ID),
				growthexchangeorder.OrderStatusIn("CREATED", "FULFILLING", "SUCCEEDED"),
			).
			Count(ctx)
		if err != nil {
			return ExchangeCreateResult{}, err
		}
		if count >= version.PerUserLimit {
			return ExchangeCreateResult{}, ErrPurchaseLimitReached
		}
	}
	wallet, err := tx.GrowthWallet.Query().Where(growthwallet.UserIDEQ(command.UserID)).Only(ctx)
	if ent.IsNotFound(err) || (wallet != nil && wallet.AvailablePoints < version.PointsCost) {
		return ExchangeCreateResult{}, ErrInsufficientPoints
	}
	if err != nil {
		return ExchangeCreateResult{}, err
	}

	nextAvailable := inventory.Available
	deltaAvailable := int64(0)
	if inventory.StockMode == "LIMITED" {
		nextAvailable--
		deltaAvailable = -1
	}
	inventory, err = tx.GrowthProductInventory.UpdateOneID(inventory.ID).
		SetAvailable(nextAvailable).
		SetReserved(inventory.Reserved + 1).
		SetLockVersion(inventory.LockVersion + 1).
		Save(ctx)
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	orderNo := newGrowthOrderNo(s.now())
	reserveKey := "reserve:" + command.IdempotencyKey
	if _, err := tx.GrowthInventoryLedger.Create().
		SetBusinessKey(reserveKey).
		SetProductID(product.ID).
		SetInventoryID(inventory.ID).
		SetOperationType("RESERVE").
		SetDeltaAvailable(deltaAvailable).
		SetDeltaReserved(1).
		SetAvailableAfter(inventory.Available).
		SetReservedAfter(inventory.Reserved).
		SetSoldAfter(inventory.Sold).
		SetReason("兑换订单预占库存").
		Save(ctx); err != nil {
		return ExchangeCreateResult{}, err
	}

	var orderID int64
	snapshot := map[string]any{
		"product_id":       product.ID,
		"product_code":     product.ProductCode,
		"title":            product.Title,
		"product_type":     product.ProductType,
		"fulfillment_mode": product.FulfillmentMode,
		"version":          version.Version,
		"points_cost":      version.PointsCost,
		"benefit_summary":  version.BenefitSummary,
		"benefit_detail":   normalizeJSONMap(version.BenefitDetail),
		"token_amount":     version.TokenAmount,
		"per_user_limit":   version.PerUserLimit,
	}
	debit, err := s.transfers.debitWalletTx(ctx, tx, WalletDebitCommand{
		UserID: command.UserID, Points: version.PointsCost, BizType: "EXCHANGE",
		IdempotencyKey: command.IdempotencyKey, Title: "兑换 " + product.Title,
		AfterLedger: func(callbackCtx context.Context, callbackTx *ent.Tx) error {
			transfer, err := callbackTx.GrowthTransfer.Query().
				Where(
					growthtransfer.BizTypeEQ("EXCHANGE"),
					growthtransfer.IdempotencyKeyEQ(command.IdempotencyKey),
				).
				Only(callbackCtx)
			if err != nil {
				return err
			}
			order, err := callbackTx.GrowthExchangeOrder.Create().
				SetOrderNo(orderNo).
				SetUserID(command.UserID).
				SetProductID(product.ID).
				SetProductVersion(version.Version).
				SetProductSnapshot(normalizeJSONMap(snapshot)).
				SetPointsCost(version.PointsCost).
				SetOrderStatus("FULFILLING").
				SetIdempotencyKey(command.IdempotencyKey).
				SetTransferID(transfer.ID).
				Save(callbackCtx)
			if err != nil {
				return err
			}
			orderID = order.ID
			_, err = callbackTx.GrowthFulfillmentJob.Create().SetOrderID(order.ID).Save(callbackCtx)
			return err
		},
	})
	if err != nil {
		_ = tx.Rollback()
		if isDuplicateKey(err) {
			if existing, existingErr := s.existingOrder(ctx, command.UserID, command.IdempotencyKey); existingErr == nil {
				return s.createResult(ctx, existing)
			}
		}
		return ExchangeCreateResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ExchangeCreateResult{}, err
	}
	committed = true
	order, err := s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	result, err := s.createResult(ctx, order)
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	result.TransferID = &debit.TransferID
	result.ServerTime = s.now().UTC().Format(time.RFC3339Nano)
	return result, nil
}

func (s *OrderService) UDetail(ctx context.Context, userID, orderID int64) (ExchangeOrderDetail, error) {
	order, err := s.orderForUser(ctx, userID, orderID)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	return s.detail(ctx, order)
}

func (s *OrderService) WDetail(ctx context.Context, userID, orderID int64) (ExchangeOrderDetail, error) {
	return s.UDetail(ctx, userID, orderID)
}

func (s *OrderService) existingOrder(ctx context.Context, userID int64, key string) (*ent.GrowthExchangeOrder, error) {
	return s.client.GrowthExchangeOrder.Query().
		Where(
			growthexchangeorder.UserIDEQ(userID),
			growthexchangeorder.IdempotencyKeyEQ(key),
		).
		Only(ctx)
}

func (s *OrderService) orderForUser(ctx context.Context, userID, orderID int64) (*ent.GrowthExchangeOrder, error) {
	order, err := s.client.GrowthExchangeOrder.Query().
		Where(growthexchangeorder.ID(orderID), growthexchangeorder.UserIDEQ(userID)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrOrderNotFound
	}
	return order, err
}

func (s *OrderService) createResult(ctx context.Context, order *ent.GrowthExchangeOrder) (ExchangeCreateResult, error) {
	balance, err := s.transfers.CurrentBalance(ctx, order.UserID)
	if err != nil {
		return ExchangeCreateResult{}, err
	}
	job, err := s.client.GrowthFulfillmentJob.Query().Where(growthfulfillmentjob.OrderIDEQ(order.ID)).Only(ctx)
	status := "PENDING"
	if err == nil {
		status = job.JobStatus
	}
	var transferID *int64
	if order.TransferID != nil {
		value := *order.TransferID
		transferID = &value
	}
	return ExchangeCreateResult{
		OrderID: order.ID, OrderNo: order.OrderNo, Status: order.OrderStatus,
		ProductVersion: order.ProductVersion, PointsCost: order.PointsCost,
		TransferID: transferID, CurrentBalance: balance, FulfillmentStatus: status,
		ServerTime: s.now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *OrderService) detail(ctx context.Context, order *ent.GrowthExchangeOrder) (ExchangeOrderDetail, error) {
	job, err := s.client.GrowthFulfillmentJob.Query().Where(growthfulfillmentjob.OrderIDEQ(order.ID)).Only(ctx)
	status := "PENDING"
	if err == nil {
		status = job.JobStatus
	}
	var transferID, refundID *int64
	if order.TransferID != nil {
		value := *order.TransferID
		transferID = &value
	}
	if order.RefundTransferID != nil {
		value := *order.RefundTransferID
		refundID = &value
	}
	return ExchangeOrderDetail{
		OrderID: order.ID, OrderNo: order.OrderNo, Status: order.OrderStatus,
		ProductSnapshot: normalizeJSONMap(order.ProductSnapshot), PointsCost: order.PointsCost,
		TransferID: transferID, FulfillmentStatus: status,
		FailureReason: order.FailureReason, RefundTransferID: refundID,
		CreatedAt: order.CreatedAt, FulfilledAt: order.FulfilledAt,
	}, nil
}

func newGrowthOrderNo(now time.Time) string {
	return "EX" + now.UTC().Format("20060102150405") + strings.ToUpper(uuid.NewString()[:6])
}
