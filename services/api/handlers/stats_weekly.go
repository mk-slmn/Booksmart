package handlers

import (
	"net/http"
	"strconv"
)

type StatDay struct {
	DayISO         string  `json:"day_iso"`
	MinutesRead    float64 `json:"minutes_read"`
	SessionsClosed int     `json:"sessions_closed"`
	PagesRead      *int    `json:"pages_read,omitempty"`
}

func (a *App) statsWeekly(w http.ResponseWriter, r *http.Request) {
	days := 7
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 30 {
				n = 30
			}
			days = n
		}
	}
	offset := -(days - 1)

	const q = `
WITH closed AS (
  SELECT
    DATE(ended_at) AS day,
    duration_seconds,
    CASE
      WHEN end_page IS NOT NULL AND start_page IS NOT NULL THEN (end_page - start_page)
      ELSE NULL
    END AS pages_read
  FROM sessions
  WHERE ended_at IS NOT NULL
    AND DATE(ended_at) >= DATE('now', ? || ' days')
),
agg AS (
  SELECT
    day,
    SUM(duration_seconds)/60.0 AS minutes_read,
    COUNT(*) AS sessions_closed,
    SUM(pages_read) AS pages_total
  FROM closed
  GROUP BY day
)
SELECT
  day,
  COALESCE(minutes_read, 0.0),
  COALESCE(sessions_closed, 0),
  pages_total
FROM agg
ORDER BY day DESC
LIMIT ?;
`
	rows, err := a.DB.Query(q, offset, days)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	out := make([]StatDay, 0, days)
	for rows.Next() {
		var d StatDay
		var pages *int
		if err := rows.Scan(&d.DayISO, &d.MinutesRead, &d.SessionsClosed, &pages); err != nil {
			writeErr(w, http.StatusInternalServerError, "scan failed")
			return
		}
		if pages != nil {
			if *pages < 0 {
				x := 0
				d.PagesRead = &x
			} else {
				d.PagesRead = pages
			}
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, "row error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"range_days": days,
		"items":      out,
	})
}
