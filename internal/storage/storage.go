// Package storage 提供历史价格存储功能
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"token-bridge-crawler/internal/adapters"
)

type Storage interface {
	SaveSnapshot(ctx context.Context, snapshot *VendorPriceSnapshot) error
	SavePriceDetails(ctx context.Context, details []VendorPriceDetail) error
	GetLatestSnapshot(ctx context.Context, vendor string) (*VendorPriceSnapshot, error)
	GetPriceHistory(ctx context.Context, vendor, modelCode string, days int) ([]VendorPriceDetail, error)
	Close()
}

type VendorPriceSnapshot struct {
	ID            string    `db:"id"`
	Vendor        string    `db:"vendor"`
	SnapshotDate  time.Time `db:"snapshot_date"`
	SnapshotAt    time.Time `db:"snapshot_at"`
	TotalModels   int       `db:"total_models"`
	NewModels     int       `db:"new_models"`
	UpdatedModels int       `db:"updated_models"`
	RemovedModels int       `db:"removed_models"`
	RawDataHash   string    `db:"raw_data_hash"`
	Status        string    `db:"status"`
	ErrorLog      string    `db:"error_log"`
	CreatedAt     time.Time `db:"created_at"`
}

type VendorPriceDetail struct {
	ID                  string          `db:"id"`
	SnapshotID          string          `db:"snapshot_id"`
	Vendor              string          `db:"vendor"`
	ModelCode           string          `db:"model_code"`
	SnapshotDate        time.Time       `db:"snapshot_date"`
	InputUSDPerMillion  float64         `db:"input_usd_per_million"`
	OutputUSDPerMillion float64         `db:"output_usd_per_million"`
	Currency            string          `db:"currency"`
	Capabilities        json.RawMessage `db:"capabilities"`
	ChangeType          string          `db:"change_type"`
	PrevPrice           json.RawMessage `db:"prev_price"`
	CreatedAt           time.Time       `db:"created_at"`
}

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(databaseURL string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return &PostgresStorage{pool: pool}, nil
}

func (s *PostgresStorage) SaveSnapshot(ctx context.Context, snapshot *VendorPriceSnapshot) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO vendor_price_snapshots 
		(id, vendor, snapshot_date, snapshot_at, total_models, new_models, updated_models, 
		 removed_models, raw_data_hash, status, error_log, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, snapshot.ID, snapshot.Vendor, snapshot.SnapshotDate, snapshot.SnapshotAt,
		snapshot.TotalModels, snapshot.NewModels, snapshot.UpdatedModels,
		snapshot.RemovedModels, snapshot.RawDataHash, snapshot.Status, snapshot.ErrorLog, snapshot.CreatedAt)
	return err
}

func (s *PostgresStorage) SavePriceDetails(ctx context.Context, details []VendorPriceDetail) error {
	if len(details) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, d := range details {
		batch.Queue(`
			INSERT INTO vendor_price_details
			(id, snapshot_id, vendor, model_code, snapshot_date, input_usd_per_million,
			 output_usd_per_million, currency, capabilities, change_type, prev_price, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (vendor, model_code, snapshot_date) DO UPDATE SET
				snapshot_id = EXCLUDED.snapshot_id,
				input_usd_per_million = EXCLUDED.input_usd_per_million,
				output_usd_per_million = EXCLUDED.output_usd_per_million,
				change_type = EXCLUDED.change_type,
				prev_price = EXCLUDED.prev_price
		`, d.ID, d.SnapshotID, d.Vendor, d.ModelCode, d.SnapshotDate,
			d.InputUSDPerMillion, d.OutputUSDPerMillion, d.Currency,
			d.Capabilities, d.ChangeType, d.PrevPrice, d.CreatedAt)
	}

	results := s.pool.SendBatch(ctx, batch)
	return results.Close()
}

func (s *PostgresStorage) GetLatestSnapshot(ctx context.Context, vendor string) (*VendorPriceSnapshot, error) {
	var snapshot VendorPriceSnapshot
	err := s.pool.QueryRow(ctx, `
		SELECT id, vendor, snapshot_date, snapshot_at, total_models, new_models, 
		       updated_models, removed_models, raw_data_hash, status, error_log, created_at
		FROM vendor_price_snapshots
		WHERE vendor = $1
		ORDER BY snapshot_date DESC
		LIMIT 1
	`, vendor).Scan(&snapshot.ID, &snapshot.Vendor, &snapshot.SnapshotDate, &snapshot.SnapshotAt,
		&snapshot.TotalModels, &snapshot.NewModels, &snapshot.UpdatedModels,
		&snapshot.RemovedModels, &snapshot.RawDataHash, &snapshot.Status,
		&snapshot.ErrorLog, &snapshot.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (s *PostgresStorage) GetPriceHistory(ctx context.Context, vendor, modelCode string, days int) ([]VendorPriceDetail, error) {
	var rows pgx.Rows
	var err error

	if modelCode == "" {
		rows, err = s.pool.Query(ctx, `
			SELECT id, snapshot_id, vendor, model_code, snapshot_date, input_usd_per_million,
			       output_usd_per_million, currency, capabilities, change_type, prev_price, created_at
			FROM vendor_price_details
			WHERE vendor = $1
			  AND snapshot_date >= CURRENT_DATE - make_interval(days => $2)
			ORDER BY snapshot_date DESC
		`, vendor, days)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, snapshot_id, vendor, model_code, snapshot_date, input_usd_per_million,
			       output_usd_per_million, currency, capabilities, change_type, prev_price, created_at
			FROM vendor_price_details
			WHERE vendor = $1 AND model_code = $2
			  AND snapshot_date >= CURRENT_DATE - make_interval(days => $3)
			ORDER BY snapshot_date DESC
		`, vendor, modelCode, days)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var details []VendorPriceDetail
	for rows.Next() {
		var d VendorPriceDetail
		err := rows.Scan(&d.ID, &d.SnapshotID, &d.Vendor, &d.ModelCode, &d.SnapshotDate,
			&d.InputUSDPerMillion, &d.OutputUSDPerMillion, &d.Currency,
			&d.Capabilities, &d.ChangeType, &d.PrevPrice, &d.CreatedAt)
		if err != nil {
			return nil, err
		}
		details = append(details, d)
	}
	return details, rows.Err()
}

func (s *PostgresStorage) Close() {
	s.pool.Close()
}

func AdapterPricesToDetails(snapshotID string, snapshotDate time.Time, prices []adapters.ModelPrice, prevPrices map[string]VendorPriceDetail) []VendorPriceDetail {
	details := make([]VendorPriceDetail, 0, len(prices))

	for _, p := range prices {
		detail := VendorPriceDetail{
			ID:           uuid.New().String(),
			SnapshotID:   snapshotID,
			Vendor:       p.Vendor,
			ModelCode:    p.ModelCode,
			SnapshotDate: snapshotDate,
			Currency:     p.PricingRaw.Currency,
			CreatedAt:    time.Now().UTC(),
		}

		if p.PricingRaw.InputUSDPerMillion != nil {
			detail.InputUSDPerMillion = *p.PricingRaw.InputUSDPerMillion
		}
		if p.PricingRaw.OutputUSDPerMillion != nil {
			detail.OutputUSDPerMillion = *p.PricingRaw.OutputUSDPerMillion
		}
		if p.Capabilities != nil {
			detail.Capabilities, _ = json.Marshal(p.Capabilities)
		}

		if prev, ok := prevPrices[p.ModelCode]; ok {
			if prev.InputUSDPerMillion != detail.InputUSDPerMillion ||
				prev.OutputUSDPerMillion != detail.OutputUSDPerMillion {
				detail.ChangeType = "updated"
				prevJSON, _ := json.Marshal(map[string]float64{
					"input":  prev.InputUSDPerMillion,
					"output": prev.OutputUSDPerMillion,
				})
				detail.PrevPrice = prevJSON
			} else {
				detail.ChangeType = "unchanged"
			}
		} else {
			detail.ChangeType = "new"
		}

		details = append(details, detail)
	}

	return details
}
