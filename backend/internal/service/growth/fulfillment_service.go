package growth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/growthexchangeorder"
	"github.com/Wei-Shaw/sub2api/ent/growthfulfillmentjob"
	"github.com/Wei-Shaw/sub2api/ent/growthproductinventory"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrRetryableFulfillment = errors.New("retryable fulfillment failure")

type FulfillmentRequest struct {
	OrderID         int64
	UserID          int64
	ProductType     string
	TokenAmount     int64
	ProductSnapshot map[string]any
}

type FulfillmentResponse struct {
	ProviderReference string
}

type FulfillmentAdapter interface {
	Fulfill(ctx context.Context, request FulfillmentRequest) (FulfillmentResponse, error)
}

type FulfillmentAdapterFunc func(context.Context, FulfillmentRequest) (FulfillmentResponse, error)

func (f FulfillmentAdapterFunc) Fulfill(ctx context.Context, request FulfillmentRequest) (FulfillmentResponse, error) {
	return f(ctx, request)
}

type FulfillmentService struct {
	client    *ent.Client
	transfers *TransferService
	mu        sync.RWMutex
	adapters  map[string]FulfillmentAdapter
	now       func() time.Time
	stop      chan struct{}
	stopOnce  sync.Once
}

func NewFulfillmentService(client *ent.Client, transfers *TransferService) *FulfillmentService {
	service := &FulfillmentService{
		client:    client,
		transfers: transfers,
		adapters:  make(map[string]FulfillmentAdapter),
		now:       time.Now,
		stop:      make(chan struct{}),
	}
	go service.run(10 * time.Second)
	return service
}

func (s *FulfillmentService) SetAdapter(productType string, adapter FulfillmentAdapter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adapters[strings.ToUpper(strings.TrimSpace(productType))] = adapter
}

func (s *FulfillmentService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stop) })
}

func (s *FulfillmentService) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			_, _ = s.ProcessDue(context.Background())
		}
	}
}

func (s *FulfillmentService) ProcessDue(ctx context.Context) (int, error) {
	now := s.now()
	jobs, err := s.client.GrowthFulfillmentJob.Query().
		Where(
			growthfulfillmentjob.JobStatusIn("PENDING", "RETRY"),
			growthfulfillmentjob.NextRetryAtLTE(now),
			growthfulfillmentjob.Or(
				growthfulfillmentjob.LockedUntilIsNil(),
				growthfulfillmentjob.LockedUntilLT(now),
			),
		).
		Order(ent.Asc(growthfulfillmentjob.FieldNextRetryAt)).
		Limit(50).
		All(ctx)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, job := range jobs {
		order, err := s.client.GrowthExchangeOrder.Get(ctx, job.OrderID)
		if err != nil {
			continue
		}
		if productType(order) == "PHYSICAL" {
			continue
		}
		claimed, err := s.claim(ctx, job.ID, now)
		if err != nil || !claimed {
			continue
		}
		processed++
		job, err = s.client.GrowthFulfillmentJob.Get(ctx, job.ID)
		if err != nil {
			continue
		}
		order, err = s.client.GrowthExchangeOrder.Get(ctx, job.OrderID)
		if err != nil {
			continue
		}
		adapter := s.adapter(productType(order))
		if adapter == nil {
			_ = s.failOrRetry(ctx, job, order, fmt.Errorf("%w: adapter not configured", ErrRetryableFulfillment), "")
			continue
		}
		snapshot := normalizeJSONMap(order.ProductSnapshot)
		response, fulfillErr := adapter.Fulfill(ctx, FulfillmentRequest{
			OrderID: order.ID, UserID: order.UserID, ProductType: productType(order),
			TokenAmount:     int64FromSnapshot(snapshot, "token_amount"),
			ProductSnapshot: snapshot,
		})
		if fulfillErr != nil {
			_ = s.failOrRetry(ctx, job, order, fulfillErr, "")
			continue
		}
		if err := s.complete(ctx, job, order, response.ProviderReference); err != nil {
			_ = s.failOrRetry(ctx, job, order, err, response.ProviderReference)
		}
	}
	return processed, nil
}

func (s *FulfillmentService) claim(ctx context.Context, jobID int64, now time.Time) (bool, error) {
	count, err := s.client.GrowthFulfillmentJob.Update().
		Where(
			growthfulfillmentjob.ID(jobID),
			growthfulfillmentjob.JobStatusIn("PENDING", "RETRY"),
			growthfulfillmentjob.NextRetryAtLTE(now),
			growthfulfillmentjob.Or(
				growthfulfillmentjob.LockedUntilIsNil(),
				growthfulfillmentjob.LockedUntilLT(now),
			),
		).
		SetJobStatus("PROCESSING").
		SetLockedUntil(now.Add(45 * time.Second)).
		AddAttemptCount(1).
		Save(ctx)
	return count == 1, err
}

func (s *FulfillmentService) adapter(productType string) FulfillmentAdapter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.adapters[productType]
}

func (s *FulfillmentService) complete(ctx context.Context, job *ent.GrowthFulfillmentJob, order *ent.GrowthExchangeOrder, providerReference string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	currentOrder, err := tx.GrowthExchangeOrder.Query().Where(growthexchangeorder.ID(order.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return err
	}
	if currentOrder.OrderStatus == "SUCCEEDED" {
		return nil
	}
	if currentOrder.OrderStatus != "FULFILLING" {
		return infraerrors.Conflict("ORDER_STATE_INVALID", "order cannot be completed")
	}
	now := s.now()
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(currentOrder.ProductID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return err
	}
	if inventory.Reserved > 0 {
		inventory, err = tx.GrowthProductInventory.UpdateOneID(inventory.ID).
			SetReserved(inventory.Reserved - 1).
			SetSold(inventory.Sold + 1).
			SetLockVersion(inventory.LockVersion + 1).
			Save(ctx)
		if err != nil {
			return err
		}
		if err := s.inventoryLedger(ctx, tx, "sell:"+fmt.Sprint(order.ID), order.ID, "SELL", 0, -1, 1, inventory, "订单履约成功"); err != nil && !isDuplicateKey(err) {
			return err
		}
	}
	if _, err := tx.GrowthExchangeOrder.UpdateOneID(order.ID).
		SetOrderStatus("SUCCEEDED").
		SetFulfilledAt(now).
		Save(ctx); err != nil {
		return err
	}
	if _, err := tx.GrowthFulfillmentJob.UpdateOneID(job.ID).
		SetJobStatus("SUCCEEDED").
		SetCompletedAt(now).
		ClearLockedUntil().
		SetProviderReference(providerReference).
		Save(ctx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *FulfillmentService) failOrRetry(ctx context.Context, job *ent.GrowthFulfillmentJob, order *ent.GrowthExchangeOrder, failure error, providerReference string) error {
	if errors.Is(failure, ErrRetryableFulfillment) && job.AttemptCount < job.MaxAttempts {
		backoffSeconds := 1 << min(job.AttemptCount, 6)
		_, err := s.client.GrowthFulfillmentJob.UpdateOneID(job.ID).
			SetJobStatus("RETRY").
			SetLastError(failure.Error()).
			SetProviderReference(providerReference).
			SetNextRetryAt(s.now().Add(time.Duration(backoffSeconds) * time.Second)).
			ClearLockedUntil().
			Save(ctx)
		return err
	}
	return s.terminalRefund(ctx, order.ID, failure.Error())
}

func (s *FulfillmentService) terminalRefund(ctx context.Context, orderID int64, reason string) error {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	order, err := tx.GrowthExchangeOrder.Query().Where(growthexchangeorder.ID(orderID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ErrOrderNotFound
	}
	if err != nil {
		return err
	}
	if order.RefundTransferID != nil || order.OrderStatus == "REFUNDED" {
		return nil
	}
	if order.OrderStatus != "FULFILLING" && order.OrderStatus != "FAILED" {
		return infraerrors.Conflict("ORDER_STATE_INVALID", "order cannot be refunded")
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(order.ProductID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return err
	}
	if inventory.Reserved > 0 {
		update := tx.GrowthProductInventory.UpdateOneID(inventory.ID).
			SetReserved(inventory.Reserved - 1).
			SetLockVersion(inventory.LockVersion + 1).
			SetAvailable(inventory.Available + 1)
		deltaAvailable := int64(1)
		if inventory.StockMode != "LIMITED" {
			update.SetAvailable(inventory.Available)
			deltaAvailable = 0
		}
		inventory, err = update.Save(ctx)
		if err != nil {
			return err
		}
		if err := s.inventoryLedger(ctx, tx, "release:"+fmt.Sprint(order.ID), order.ID, "RELEASE", deltaAvailable, -1, 0, inventory, reason); err != nil && !isDuplicateKey(err) {
			return err
		}
	}
	refundKey := "refund:" + fmt.Sprint(order.ID)
	refund, err := s.transfers.reverseRedeemTx(ctx, tx, RedeemReverseCommand{
		UserID: order.UserID, Points: order.PointsCost, BizType: "REFUND",
		IdempotencyKey: refundKey, Title: "兑换失败退款", ReferenceID: order.ID,
	})
	if err != nil {
		return err
	}
	now := s.now()
	if _, err := tx.GrowthExchangeOrder.UpdateOneID(order.ID).
		SetOrderStatus("REFUNDED").
		SetFailureReason(reason).
		SetRefundTransferID(refund.TransferID).
		SetRefundedAt(now).
		Save(ctx); err != nil {
		return err
	}
	if _, err := tx.GrowthFulfillmentJob.Update().
		Where(growthfulfillmentjob.OrderIDEQ(order.ID)).
		SetJobStatus("FAILED").
		SetLastError(reason).
		SetCompletedAt(now).
		ClearLockedUntil().
		Save(ctx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *FulfillmentService) inventoryLedger(ctx context.Context, tx *ent.Tx, key string, orderID int64, operation string, deltaAvailable, deltaReserved, deltaSold int64, inventory *ent.GrowthProductInventory, reason string) error {
	_, err := tx.GrowthInventoryLedger.Create().
		SetBusinessKey(key).
		SetProductID(inventory.ProductID).
		SetInventoryID(inventory.ID).
		SetOperationType(operation).
		SetDeltaAvailable(deltaAvailable).
		SetDeltaReserved(deltaReserved).
		SetDeltaSold(deltaSold).
		SetAvailableAfter(inventory.Available).
		SetReservedAfter(inventory.Reserved).
		SetSoldAfter(inventory.Sold).
		SetOrderID(orderID).
		SetReason(reason).
		Save(ctx)
	return err
}

func (s *FulfillmentService) ManualComplete(ctx context.Context, orderID, operatorID int64, reason, requestID string) (ExchangeOrderDetail, error) {
	if operatorID <= 0 {
		return ExchangeOrderDetail{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateReason(reason); err != nil {
		return ExchangeOrderDetail{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	order, err := tx.GrowthExchangeOrder.Query().Where(growthexchangeorder.ID(orderID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ExchangeOrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	if productType(order) != "PHYSICAL" || order.OrderStatus != "FULFILLING" {
		return ExchangeOrderDetail{}, infraerrors.Conflict("ORDER_STATE_INVALID", "manual completion requires a fulfilling physical order")
	}
	inventory, err := tx.GrowthProductInventory.Query().
		Where(growthproductinventory.ProductIDEQ(order.ProductID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	if inventory.Reserved > 0 {
		inventory, err = tx.GrowthProductInventory.UpdateOneID(inventory.ID).
			SetReserved(inventory.Reserved - 1).
			SetSold(inventory.Sold + 1).
			SetLockVersion(inventory.LockVersion + 1).
			Save(ctx)
		if err != nil {
			return ExchangeOrderDetail{}, err
		}
		if err := s.inventoryLedger(ctx, tx, "sell:"+fmt.Sprint(order.ID), order.ID, "SELL", 0, -1, 1, inventory, reason); err != nil && !isDuplicateKey(err) {
			return ExchangeOrderDetail{}, err
		}
	}
	now := s.now()
	if _, err := tx.GrowthExchangeOrder.UpdateOneID(order.ID).SetOrderStatus("SUCCEEDED").SetFulfilledAt(now).Save(ctx); err != nil {
		return ExchangeOrderDetail{}, err
	}
	if _, err := tx.GrowthFulfillmentJob.Update().
		Where(growthfulfillmentjob.OrderIDEQ(order.ID)).
		SetJobStatus("SUCCEEDED").
		SetCompletedAt(now).
		ClearLockedUntil().
		Save(ctx); err != nil {
		return ExchangeOrderDetail{}, err
	}
	if err := appendOrderAudit(ctx, tx, operatorID, "order.manual-complete", order, reason, requestID); err != nil {
		return ExchangeOrderDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ExchangeOrderDetail{}, err
	}
	committed = true
	order, err = s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	orderService := NewOrderService(s.client, s.transfers)
	return orderService.detail(ctx, order)
}

func (s *FulfillmentService) AdminRefund(ctx context.Context, orderID, operatorID int64, reason, requestID string) (ExchangeOrderDetail, error) {
	if operatorID <= 0 {
		return ExchangeOrderDetail{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	if err := validateReason(reason); err != nil {
		return ExchangeOrderDetail{}, err
	}
	order, err := s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if ent.IsNotFound(err) {
		return ExchangeOrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	if order.RefundTransferID != nil || order.OrderStatus == "REFUNDED" {
		return NewOrderService(s.client, s.transfers).detail(ctx, order)
	}
	if order.OrderStatus != "FAILED" {
		return ExchangeOrderDetail{}, infraerrors.Conflict("ORDER_STATE_INVALID", "only failed orders can be refunded")
	}
	if err := s.terminalRefund(ctx, orderID, reason); err != nil {
		return ExchangeOrderDetail{}, err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	order, err = tx.GrowthExchangeOrder.Get(ctx, orderID)
	if err == nil {
		err = appendOrderAudit(ctx, tx, operatorID, "order.refund", order, reason, requestID)
	}
	if err == nil {
		err = tx.Commit()
	} else {
		_ = tx.Rollback()
	}
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	order, err = s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	return NewOrderService(s.client, s.transfers).detail(ctx, order)
}

func (s *FulfillmentService) ListAdminOrders(ctx context.Context, page, pageSize int, status, search string) ([]ExchangeOrderDetail, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := s.client.GrowthExchangeOrder.Query()
	if strings.TrimSpace(status) != "" {
		query = query.Where(growthexchangeorder.OrderStatusEQ(strings.TrimSpace(status)))
	}
	if strings.TrimSpace(search) != "" {
		query = query.Where(growthexchangeorder.OrderNoContainsFold(strings.TrimSpace(search)))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	orders, err := query.Order(ent.Desc(growthexchangeorder.FieldCreatedAt)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	items := make([]ExchangeOrderDetail, 0, len(orders))
	orderService := NewOrderService(s.client, s.transfers)
	for _, order := range orders {
		item, err := orderService.detail(ctx, order)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, int64(total), nil
}

func (s *FulfillmentService) AdminDetail(ctx context.Context, orderID int64) (ExchangeOrderDetail, error) {
	order, err := s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if ent.IsNotFound(err) {
		return ExchangeOrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	return NewOrderService(s.client, s.transfers).detail(ctx, order)
}

func (s *FulfillmentService) Retry(ctx context.Context, orderID, operatorID int64, requestID string) (ExchangeOrderDetail, error) {
	if operatorID <= 0 {
		return ExchangeOrderDetail{}, infraerrors.Unauthorized("UNAUTHORIZED", "administrator identity is required")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	order, err := tx.GrowthExchangeOrder.Query().Where(growthexchangeorder.ID(orderID)).ForUpdate().Only(ctx)
	if ent.IsNotFound(err) {
		return ExchangeOrderDetail{}, ErrOrderNotFound
	}
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	if order.OrderStatus != "FAILED" || order.RefundTransferID != nil {
		return ExchangeOrderDetail{}, infraerrors.Conflict("ORDER_RETRY_INVALID", "order is not eligible for retry")
	}
	now := s.now()
	if _, err := tx.GrowthExchangeOrder.UpdateOneID(order.ID).SetOrderStatus("FULFILLING").Save(ctx); err != nil {
		return ExchangeOrderDetail{}, err
	}
	job, err := tx.GrowthFulfillmentJob.Query().Where(growthfulfillmentjob.OrderIDEQ(order.ID)).Only(ctx)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	if _, err := tx.GrowthFulfillmentJob.UpdateOneID(job.ID).
		SetJobStatus("PENDING").
		SetAttemptCount(0).
		SetNextRetryAt(now).
		ClearLockedUntil().
		Save(ctx); err != nil {
		return ExchangeOrderDetail{}, err
	}
	if err := appendOrderAudit(ctx, tx, operatorID, "order.retry", order, "人工重试履约", requestID); err != nil {
		return ExchangeOrderDetail{}, err
	}
	if err := tx.Commit(); err != nil {
		return ExchangeOrderDetail{}, err
	}
	committed = true
	order, err = s.client.GrowthExchangeOrder.Get(ctx, orderID)
	if err != nil {
		return ExchangeOrderDetail{}, err
	}
	return NewOrderService(s.client, s.transfers).detail(ctx, order)
}

func appendOrderAudit(ctx context.Context, tx *ent.Tx, operatorID int64, action string, order *ent.GrowthExchangeOrder, reason, requestID string) error {
	_, err := tx.GrowthAdminAudit.Create().
		SetOperatorID(operatorID).
		SetActionType(action).
		SetResourceType("growth_exchange_order").
		SetResourceID(order.ID).
		SetBeforeJSON(normalizeJSONMap(map[string]any{"status": order.OrderStatus, "refund_transfer_id": order.RefundTransferID})).
		SetAfterJSON(normalizeJSONMap(map[string]any{"reason": reason})).
		SetReason(strings.TrimSpace(reason)).
		SetRequestID(strings.TrimSpace(requestID)).
		Save(ctx)
	return err
}

func productType(order *ent.GrowthExchangeOrder) string {
	if value, ok := normalizeJSONMap(order.ProductSnapshot)["product_type"].(string); ok {
		return value
	}
	return ""
}

func int64FromSnapshot(snapshot map[string]any, key string) int64 {
	switch value := snapshot[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case json.Number:
		result, _ := value.Int64()
		return result
	default:
		return 0
	}
}

func min(value, limit int) int {
	if value < limit {
		return value
	}
	return limit
}
