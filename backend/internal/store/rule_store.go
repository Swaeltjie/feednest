package store

import (
	"regexp"
	"strings"
	"sync"

	"github.com/feednest/backend/internal/models"
)

var (
	regexCache   = make(map[string]*regexp.Regexp)
	regexCacheMu sync.RWMutex
)

func compileRegexCached(pattern string) (*regexp.Regexp, error) {
	regexCacheMu.RLock()
	if re, ok := regexCache[pattern]; ok {
		regexCacheMu.RUnlock()
		return re, nil
	}
	regexCacheMu.RUnlock()

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCacheMu.Lock()
	if len(regexCache) >= 100 {
		// Simple eviction: clear the entire cache when it reaches max size
		regexCache = make(map[string]*regexp.Regexp)
	}
	regexCache[pattern] = re
	regexCacheMu.Unlock()
	return re, nil
}

func (q *Queries) CreateRule(userID int64, name, field, operator, value, action string, feedID *int64) (*models.FilterRule, error) {
	result, err := q.db.Exec(`
		INSERT INTO filter_rules (user_id, name, feed_id, field, operator, value, action)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, name, feedID, field, operator, value, action,
	)
	if err != nil {
		return nil, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return q.getRuleByID(id, userID)
}

func (q *Queries) ListRules(userID int64) ([]models.FilterRule, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, name, feed_id, field, operator, value, action, enabled, created_at
		FROM filter_rules WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.FilterRule
	for rows.Next() {
		var r models.FilterRule
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.FeedID, &r.Field, &r.Operator, &r.Value, &r.Action, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func (q *Queries) UpdateRule(id, userID int64, req models.UpdateRuleRequest) error {
	var setClauses []string
	var args []interface{}

	if req.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *req.Name)
	}
	if req.FeedID != nil {
		setClauses = append(setClauses, "feed_id = ?")
		args = append(args, *req.FeedID)
	}
	if req.Field != nil {
		setClauses = append(setClauses, "field = ?")
		args = append(args, *req.Field)
	}
	if req.Operator != nil {
		setClauses = append(setClauses, "operator = ?")
		args = append(args, *req.Operator)
	}
	if req.Value != nil {
		setClauses = append(setClauses, "value = ?")
		args = append(args, *req.Value)
	}
	if req.Action != nil {
		setClauses = append(setClauses, "action = ?")
		args = append(args, *req.Action)
	}
	if req.Enabled != nil {
		setClauses = append(setClauses, "enabled = ?")
		args = append(args, *req.Enabled)
	}

	if len(setClauses) == 0 {
		return nil
	}

	query := "UPDATE filter_rules SET " + strings.Join(setClauses, ", ") + " WHERE id = ? AND user_id = ?"
	args = append(args, id, userID)
	_, err := q.db.Exec(query, args...)
	return err
}

func (q *Queries) DeleteRule(id, userID int64) error {
	_, err := q.db.Exec("DELETE FROM filter_rules WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (q *Queries) GetRulesForFeed(userID int64, feedID *int64) ([]models.FilterRule, error) {
	var rows_result []models.FilterRule
	var query string
	var args []interface{}

	if feedID != nil {
		query = `
			SELECT id, user_id, name, feed_id, field, operator, value, action, enabled, created_at
			FROM filter_rules
			WHERE user_id = ? AND enabled = 1 AND (feed_id IS NULL OR feed_id = ?)
			ORDER BY created_at`
		args = []interface{}{userID, *feedID}
	} else {
		query = `
			SELECT id, user_id, name, feed_id, field, operator, value, action, enabled, created_at
			FROM filter_rules
			WHERE user_id = ? AND enabled = 1 AND feed_id IS NULL
			ORDER BY created_at`
		args = []interface{}{userID}
	}

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r models.FilterRule
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.FeedID, &r.Field, &r.Operator, &r.Value, &r.Action, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rows_result = append(rows_result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rows_result, nil
}

func (q *Queries) GetHideRules(userID int64) ([]models.FilterRule, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, name, feed_id, field, operator, value, action, enabled, created_at
		FROM filter_rules
		WHERE user_id = ? AND enabled = 1 AND action = 'hide'
		ORDER BY created_at`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.FilterRule
	for rows.Next() {
		var r models.FilterRule
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.FeedID, &r.Field, &r.Operator, &r.Value, &r.Action, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func (q *Queries) ApplyAutoRules(userID, articleID, feedID int64, title, author, content string) error {
	rules, err := q.GetRulesForFeed(userID, &feedID)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if rule.Action != "auto_read" && rule.Action != "auto_star" {
			continue
		}

		var fieldValue string
		switch rule.Field {
		case "title":
			fieldValue = title
		case "author":
			fieldValue = author
		case "content":
			fieldValue = content
		}

		matched := false
		lower := strings.ToLower(fieldValue)
		lowerValue := strings.ToLower(rule.Value)

		switch rule.Operator {
		case "contains":
			matched = strings.Contains(lower, lowerValue)
		case "not_contains":
			matched = !strings.Contains(lower, lowerValue)
		case "regex":
			// Regex matching is handled via Go's regexp package
			if re, err := compileRegexCached(rule.Value); err == nil {
				matched = re.MatchString(fieldValue)
			}
		}

		if matched {
			switch rule.Action {
			case "auto_read":
				isRead := true
				if err := q.UpdateArticle(articleID, userID, &isRead, nil); err != nil {
					return err
				}
			case "auto_star":
				isStarred := true
				if err := q.UpdateArticle(articleID, userID, nil, &isStarred); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (q *Queries) getRuleByID(id, userID int64) (*models.FilterRule, error) {
	var r models.FilterRule
	err := q.db.QueryRow(`
		SELECT id, user_id, name, feed_id, field, operator, value, action, enabled, created_at
		FROM filter_rules WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&r.ID, &r.UserID, &r.Name, &r.FeedID, &r.Field, &r.Operator, &r.Value, &r.Action, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
