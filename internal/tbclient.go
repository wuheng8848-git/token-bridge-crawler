// Package internal 提供 TB API 客户端
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"token-bridge-crawler/internal/adapters"
)

// TBClient Token Bridge API 客户端
type TBClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewTBClient 创建 TB 客户端
func NewTBClient(baseURL, token string) *TBClient {
	return &TBClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// StagingImportItem Staging 导入项
type StagingImportItem struct {
	Source                       string          `json:"source"`
	ModelCode                    string          `json:"model_code"`
	ModelName                    string          `json:"model_name"`
	PricingRaw                   json.RawMessage `json:"pricing_raw"`
	SuggestedRetailUSDMinorPer1k *int64          `json:"suggested_retail_usd_minor_per_1k,omitempty"`
}

// ImportToStaging 导入到 TB Staging
func (c *TBClient) ImportToStaging(ctx context.Context, items []StagingImportItem) error {
	if len(items) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/v1/admin/supplier_catalog_staging/import", c.baseURL)

	body, err := json.Marshal(items)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("import failed with status: %d", resp.StatusCode)
	}

	return nil
}

// AdapterPricesToStaging 将适配器价格转换为 Staging 格式
func AdapterPricesToStaging(prices []adapters.ModelPrice) []StagingImportItem {
	items := make([]StagingImportItem, 0, len(prices))

	for _, p := range prices {
		pricingRaw, _ := json.Marshal(p.PricingRaw)

		item := StagingImportItem{
			Source:     p.Source,
			ModelCode:  p.ModelCode,
			ModelName:  p.ModelName,
			PricingRaw: pricingRaw,
		}

		items = append(items, item)
	}

	return items
}

// ImportPrices 批量导入价格（分批）
func (c *TBClient) ImportPrices(ctx context.Context, prices []adapters.ModelPrice, batchSize int) error {
	items := AdapterPricesToStaging(prices)

	// 分批导入
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}

		batch := items[i:end]
		if err := c.ImportToStaging(ctx, batch); err != nil {
			return fmt.Errorf("import batch %d-%d failed: %w", i, end, err)
		}

		// 批次间稍作延迟
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
