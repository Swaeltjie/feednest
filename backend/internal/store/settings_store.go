package store

func (q *Queries) GetSettings(userID int64) (map[string]string, error) {
	rows, err := q.db.Query("SELECT key, value FROM settings WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

func (q *Queries) SetSetting(userID int64, key, value string) error {
	_, err := q.db.Exec(
		"INSERT INTO settings (user_id, key, value) VALUES (?, ?, ?) ON CONFLICT(user_id, key) DO UPDATE SET value = ?",
		userID, key, value, value,
	)
	return err
}

func (q *Queries) UpdateSettings(userID int64, settings map[string]string) error {
	for key, value := range settings {
		if err := q.SetSetting(userID, key, value); err != nil {
			return err
		}
	}
	return nil
}
