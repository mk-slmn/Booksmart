package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

type bookItem struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Author    *string `json:"author,omitempty"`
	Source    *string `json:"source,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func (a *App) listBooks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
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
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	where := ""
	args := []any{}
	if q != "" {
		where = "WHERE LOWER(b.title) LIKE ? OR LOWER(IFNULL(b.author,'')) LIKE ?"
		like := "%" + strings.ToLower(q) + "%"
		args = append(args, like, like)
	}

	query := `
SELECT b.id, b.title, b.author, b.source, b.created_at
FROM books b
` + where + `
ORDER BY b.created_at DESC, b.id DESC
LIMIT ? OFFSET ?;
`
	argsQ := append(args, limit, offset)

	rows, err := a.DB.Query(query, argsQ...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	items := make([]bookItem, 0, limit)
	for rows.Next() {
		var it bookItem
		if err := rows.Scan(&it.ID, &it.Title, &it.Author, &it.Source, &it.CreatedAt); err != nil {
			writeErr(w, http.StatusInternalServerError, "scan failed")
			return
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "row error")
		return
	}

	total, err := countBooks(a.DB, q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "count failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"meta": map[string]any{
			"limit":  limit,
			"offset": offset,
			"count":  len(items),
			"total":  total,
			"q":      q,
		},
	})
}

func countBooks(db *sql.DB, q string) (int, error) {
	if strings.TrimSpace(q) == "" {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM books`).Scan(&n)
		return n, err
	}
	q = "%" + strings.ToLower(strings.TrimSpace(q)) + "%"
	var n int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM books b
		WHERE LOWER(b.title) LIKE ? OR LOWER(IFNULL(b.author,'')) LIKE ?
	`, q, q).Scan(&n)
	return n, err
}
