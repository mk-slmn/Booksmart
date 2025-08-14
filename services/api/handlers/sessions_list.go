package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

type sessionListItem struct {
	ID              int64   `json:"id"`
	BookID          int64   `json:"book_id"`
	BookTitle       string  `json:"book_title"`
	Author          *string `json:"author,omitempty"`
	Source          *string `json:"source,omitempty"`
	DeviceID        string  `json:"device_id"`
	StartPage       int     `json:"start_page"`
	EndPage         *int    `json:"end_page,omitempty"`
	StartedAt       string  `json:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty"`
	DurationSeconds *int64  `json:"duration_seconds,omitempty"`
	CreatedAt       string  `json:"created_at"`
	Status          string  `json:"status"`
	LastActivity    string  `json:"last_activity"`
}

func (a *App) listSessions(w http.ResponseWriter, r *http.Request) {
	// pagination
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	device := strings.TrimSpace(r.URL.Query().Get("device_id"))
	bookTitle := strings.TrimSpace(r.URL.Query().Get("book_title"))

	conds := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if device != "" {
		conds = append(conds, "s.device_id = ?")
		args = append(args, device)
	}
	if bookTitle != "" {
		conds = append(conds, "b.title = ?")
		args = append(args, bookTitle)
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	const base = `
SELECT
  s.id,
  s.book_id,
  b.title,
  b.author,
  b.source,
  s.device_id,
  s.start_page,
  s.end_page,
  s.started_at,
  s.ended_at,
  s.duration_seconds,
  s.created_at,
  CASE WHEN s.ended_at IS NULL THEN 'open' ELSE 'closed' END AS status,
  COALESCE(s.ended_at, s.started_at) AS last_activity
FROM sessions s
JOIN books b ON b.id = s.book_id
`
	query := base + where + `
ORDER BY last_activity DESC, s.id DESC
LIMIT ? OFFSET ?;`

	argsQ := append(args, limit, offset)

	rows, err := a.DB.Query(query, argsQ...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := make([]sessionListItem, 0, limit)
	for rows.Next() {
		var it sessionListItem
		if err := rows.Scan(
			&it.ID,
			&it.BookID,
			&it.BookTitle,
			&it.Author,
			&it.Source,
			&it.DeviceID,
			&it.StartPage,
			&it.EndPage,
			&it.StartedAt,
			&it.EndedAt,
			&it.DurationSeconds,
			&it.CreatedAt,
			&it.Status,
			&it.LastActivity,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, "scan failed")
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "row error")
		return
	}

	total, err := countSessions(a, device, bookTitle)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "count failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta": map[string]any{
			"limit":      limit,
			"offset":     offset,
			"count":      len(items),
			"total":      total,
			"device_id":  device,
			"book_title": bookTitle,
		},
	})
}

func countSessions(a *App, device, bookTitle string) (int, error) {
	conds := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if device != "" {
		conds = append(conds, "device_id = ?")
		args = append(args, device)
	}
	if bookTitle != "" {
		conds = append(conds, "book_id IN (SELECT id FROM books WHERE title = ?)")
		args = append(args, bookTitle)
	}

	sql := `SELECT COUNT(*) FROM sessions`
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}

	var n int
	if err := a.DB.QueryRow(sql, args...).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
