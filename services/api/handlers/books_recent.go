package handlers

import (
	"net/http"
	"strconv"
)

type recentBook struct {
	ID           int64   `json:"id"`
	Title        string  `json:"title"`
	Author       *string `json:"author,omitempty"`
	Source       *string `json:"source,omitempty"`
	LastActivity *string `json:"last_activity,omitempty"`
}

func (a *App) recentBooks(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 50 {
				n = 50
			}
			limit = n
		}
	}

	const q = `
SELECT
  b.id,
  b.title,
  b.author,
  b.source,
  CASE
    WHEN MAX(s.ended_at) IS NOT NULL THEN MAX(s.ended_at)
    ELSE MAX(s.started_at)
  END AS last_activity
FROM books b
LEFT JOIN sessions s ON s.book_id = b.id
GROUP BY b.id
ORDER BY
  last_activity IS NULL,  -- false first (has activity), true last (never read)
  last_activity DESC,     -- newest first
  b.created_at DESC       -- tie-breaker for never-read books
LIMIT ?;`

	rows, err := a.DB.Query(q, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := make([]recentBook, 0, limit)
	for rows.Next() {
		var it recentBook
		var last *string
		if err := rows.Scan(&it.ID, &it.Title, &it.Author, &it.Source, &last); err != nil {
			writeErr(w, http.StatusInternalServerError, "scan failed")
			return
		}
		it.LastActivity = last
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "row error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta":  map[string]any{"limit": limit, "count": len(items)},
	})
}
