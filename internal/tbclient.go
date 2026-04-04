// Package internal 提供 TB API 客户端
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// StagingImportBody 主项目 POST /v1/admin/supplier_catalog_staging/import 期望的顶层结构
type StagingImportBody struct {
	Source                string               `json:"source"`
	CanonicalSupplierCode *string              `json:"canonical_supplier_code,omitempty"`
	ExternalBatchID       *string              `json:"external_batch_id,omitempty"`
	DefaultPricingRule    *string              `json:"default_pricing_rule_template,omitempty"`
	Items                 []StagingImportItem  `json:"items"`
}

// StagingImportItem Staging 导入项（嵌套在 items 数组内）
type StagingImportItem struct {
	ModelCode                       string          `json:"model_code"`
	DisplayName                     string          `json:"display_name"`
	PricingRaw                      json.RawMessage `json:"pricing_raw"`
	SuggestedRetailUsdMinorPer1k    *int64          `json:"suggested_retail_usd_minor_per_1k,omitempty"`
	SuggestedInputUpstreamCnyPer1k  *int64          `json:"suggested_input_upstream_cny_minor_per_1k,omitempty"`
	SuggestedOutputUpstreamCnyPer1k *int64          `json:"suggested_output_upstream_cny_minor_per_1k,omitempty"`
	PricingRuleTemplate             string          `json:"pricing_rule_template,omitempty"`
	Notes                           string          `json:"notes,omitempty"`
}

// ImportToStaging 导入到 TB Staging
func (c *TBClient) ImportToStaging(ctx context.Context, body *StagingImportBody) error {
	if body == nil || len(body.Items) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/v1/admin/supplier_catalog_staging/import", c.baseURL)

	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
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

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("import failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
	}

	return nil
}

// AdapterPricesToStagingBody 将适配器价格转换为 Staging 导入格式
// source: 批次来源标识（如 "openai-2026-03-31"）
// canonicalSupplierCode: 建议对齐的供应商编码（可选）
func AdapterPricesToStagingBody(prices []adapters.ModelPrice, source string, canonicalSupplierCode *string) *StagingImportBody {
	if len(prices) == 0 {
		return nil
	}

	items := make([]StagingImportItem, 0, len(prices))
	for _, p := range prices {
		pricingRaw, _ := json.Marshal(p.PricingRaw)

		item := StagingImportItem{
			ModelCode:   p.ModelCode,
			DisplayName: p.ModelName, // model_name 映射到 display_name
			PricingRaw:  pricingRaw,
		}

		items = append(items, item)
	}

	return &StagingImportBody{
		Source:                source,
		CanonicalSupplierCode: canonicalSupplierCode,
		Items:                 items,
	}
}

// ImportPrices 批量导入价格
// source: 批次来源标识（如 "openai-2026-03-31"）
// canonicalSupplierCode: 建议对齐的供应商编码（可选）
// batchSize: 每批最大条目数
func (c *TBClient) ImportPrices(ctx context.Context, prices []adapters.ModelPrice, source string, canonicalSupplierCode *string, batchSize int) error {
	if len(prices) == 0 {
		return nil
	}

	// 分批导入
	for i := 0; i < len(prices); i += batchSize {
		end := i + batchSize
		if end > len(prices) {
			end = len(prices)
		}

		batch := prices[i:end]
		body := AdapterPricesToStagingBody(batch, source, canonicalSupplierCode)
		if err := c.ImportToStaging(ctx, body); err != nil {
			return fmt.Errorf("import batch %d-%d failed: %w", i, end, err)
		}

		// 批次间稍作延迟
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}