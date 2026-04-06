// Package rules 提供噪声识别与质量评分的规则引擎
package rules

import (
	"database/sql"
	"fmt"
	"time"
)

// DBStorage 数据库规则存储
type DBStorage struct {
	db *sql.DB
}

// NewDBStorage 创建数据库存储
func NewDBStorage(db *sql.DB) *DBStorage {
	return &DBStorage{db: db}
}

// LoadRules 从数据库加载规则
func (s *DBStorage) LoadRules() ([]Rule, error) {
	query := `
		SELECT id, rule_type, rule_name, rule_value, weight, is_active, priority, created_at, updated_at
		FROM noise_rules
		WHERE is_active = true
		ORDER BY priority DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []Rule
	for rows.Next() {
		var rule Rule
		err := rows.Scan(
			&rule.ID,
			&rule.RuleType,
			&rule.RuleName,
			&rule.RuleValue,
			&rule.Weight,
			&rule.IsActive,
			&rule.Priority,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// SaveRule 保存规则到数据库
func (s *DBStorage) SaveRule(rule *Rule) error {
	now := time.Now()

	if rule.ID == 0 {
		// 插入新规则
		query := `
			INSERT INTO noise_rules (rule_type, rule_name, rule_value, weight, is_active, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`
		err := s.db.QueryRow(
			query,
			rule.RuleType,
			rule.RuleName,
			rule.RuleValue,
			rule.Weight,
			rule.IsActive,
			rule.Priority,
			now,
			now,
		).Scan(&rule.ID)
		if err != nil {
			return fmt.Errorf("failed to insert rule: %w", err)
		}
		rule.CreatedAt = now
		rule.UpdatedAt = now
	} else {
		// 更新现有规则
		query := `
			UPDATE noise_rules
			SET rule_type = $1, rule_name = $2, rule_value = $3, weight = $4,
			    is_active = $5, priority = $6, updated_at = $7
			WHERE id = $8
		`
		_, err := s.db.Exec(
			query,
			rule.RuleType,
			rule.RuleName,
			rule.RuleValue,
			rule.Weight,
			rule.IsActive,
			rule.Priority,
			now,
			rule.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update rule: %w", err)
		}
		rule.UpdatedAt = now
	}

	return nil
}

// DeleteRule 从数据库删除规则
func (s *DBStorage) DeleteRule(ruleID int64) error {
	query := `DELETE FROM noise_rules WHERE id = $1`
	_, err := s.db.Exec(query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	return nil
}

// MemoryStorage 内存规则存储（用于测试）
type MemoryStorage struct {
	rules []Rule
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		rules: DefaultRules(),
	}
}

// LoadRules 加载规则
func (s *MemoryStorage) LoadRules() ([]Rule, error) {
	return s.rules, nil
}

// SaveRule 保存规则
func (s *MemoryStorage) SaveRule(rule *Rule) error {
	if rule.ID == 0 {
		// 分配新 ID
		maxID := int64(0)
		for _, r := range s.rules {
			if r.ID > maxID {
				maxID = r.ID
			}
		}
		rule.ID = maxID + 1
		s.rules = append(s.rules, *rule)
	} else {
		// 更新现有规则
		for i, r := range s.rules {
			if r.ID == rule.ID {
				s.rules[i] = *rule
				break
			}
		}
	}
	return nil
}

// DeleteRule 删除规则
func (s *MemoryStorage) DeleteRule(ruleID int64) error {
	for i, r := range s.rules {
		if r.ID == ruleID {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			break
		}
	}
	return nil
}
