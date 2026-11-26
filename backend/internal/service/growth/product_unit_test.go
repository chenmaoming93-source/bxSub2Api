//go:build unit

package growth

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestValidateProductCreate(t *testing.T) {
	valid := ProductCreateInput{
		OperatorID:      1,
		ProductCode:     "token_1m",
		Title:           "100 万 Token",
		ProductType:     "TOKEN_QUOTA",
		FulfillmentMode: "AUTOMATIC",
		StockMode:       "LIMITED",
		InitialStock:    10,
		Version: ProductVersionInput{
			PointsCost:   50,
			TokenAmount:  1_000_000,
			PerUserLimit: 3,
		},
	}
	require.NoError(t, validateProductCreate(valid))
	manual := valid
	manual.FulfillmentMode = "MANUAL"
	require.NoError(t, validateProductCreate(manual))

	tests := []struct {
		name   string
		mutate func(*ProductCreateInput)
	}{
		{name: "missing code", mutate: func(input *ProductCreateInput) { input.ProductCode = "" }},
		{name: "invalid type", mutate: func(input *ProductCreateInput) { input.ProductType = "OTHER" }},
		{name: "negative stock", mutate: func(input *ProductCreateInput) { input.InitialStock = -1 }},
		{name: "invalid price", mutate: func(input *ProductCreateInput) { input.Version.PointsCost = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			require.Error(t, validateProductCreate(input))
		})
	}
}

func TestBuildProductViewUnlimitedStock(t *testing.T) {
	product := &ent.GrowthProduct{
		ID:              1,
		ProductCode:     "pro_7d",
		Title:           "Pro 7 天",
		ProductType:     "SUBSCRIPTION",
		FulfillmentMode: "AUTOMATIC",
		ProductStatus:   "ACTIVE",
		CurrentVersion:  2,
	}
	version := &ent.GrowthProductVersion{
		ID:             3,
		ProductID:      1,
		Version:        2,
		PointsCost:     300,
		BenefitSummary: "Pro 体验 7 天",
		PerUserLimit:   1,
	}
	inventory := &ent.GrowthProductInventory{
		ProductID: 1,
		StockMode: "UNLIMITED",
		Sold:      5,
	}
	view := buildProductView(product, version, nil, inventory)
	require.Equal(t, int64(-1), view.Stock)
	require.Equal(t, int64(300), view.PointsCost)
	require.Equal(t, 2, view.CurrentVersion)
}
