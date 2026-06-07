package store

type Item struct {
	ID          int64  `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Source      string `json:"source"`
	Member      string `json:"member"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	EntryDate   string `json:"entry_date"`
	Priority    string `json:"priority"`
	Link        string `json:"link"`
	Status      string `json:"status"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
	ResolvedAt  string `json:"resolved_at"`
}

type SourceHealth struct {
	Source       string `json:"source"`
	LastRun      string `json:"last_run"`
	DurationMs   int64  `json:"duration_ms"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	EntriesCount int    `json:"entries_count"`
	RunID        string `json:"run_id"`
}

type Run struct {
	ID            string
	StartedAt     string
	FinishedAt    string
	SourcesTotal  int
	SourcesOK     int
	SourcesFailed int
}

type DailyCount struct {
	Date  string
	Count int
}

type Metric struct {
	ID         int64   `json:"id"`
	Source     string  `json:"source"`
	Name       string  `json:"name"`
	Value      float64 `json:"value"`
	RecordedAt string  `json:"recorded_at"`
}

type DailyMetric struct {
	Date  string
	Value float64
}

type ChatSession struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	StartedAt    string `json:"started_at"`
	LastActiveAt string `json:"last_active_at"`
}

type ChatMessage struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type AgentMemory struct {
	ID           int64  `json:"id"`
	CreatedAt    string `json:"created_at"`
	LastScrapeAt string `json:"last_scrape_at"`
	Content      string `json:"content"`
}

type AckedAlert struct {
	Fingerprint string `json:"fingerprint"`
	Title       string `json:"title"`
	AckedAt     string `json:"acked_at"`
}

type TokenSummary struct {
	TodayTokens int            `json:"today_tokens"`
	WeekTokens  int            `json:"week_tokens"`
	TotalCalls  int            `json:"total_calls"`
	AvgPerDay   int            `json:"avg_per_day"`
	BySource    map[string]int `json:"by_source"`
}

type DailyTokens struct {
	Date  string `json:"date"`
	Agent int    `json:"agent"`
	Chat  int    `json:"chat"`
}

type Member struct {
	ID                  int64  `json:"id"`
	Fingerprint         string `json:"fingerprint"`
	Name                string `json:"name"`
	Email               string `json:"email"`
	Aliases             string `json:"aliases"`
	Role                string `json:"role"`
	Department          string `json:"department"`
	Branch              string `json:"branch"`
	Registration        string `json:"registration"`
	ManagerID           *int64 `json:"manager_id"`
	ManagerRegistration string `json:"manager_registration"`
	ManagerRef          string `json:"-"`
	Source              string `json:"source"`
	FirstSeenAt         string `json:"first_seen_at"`
	LastSeenAt          string `json:"last_seen_at"`
}

type PruneResult struct {
	Items    int64 `json:"items"`
	Messages int64 `json:"messages"`
	Sessions int64 `json:"sessions"`
	Memories int64 `json:"memories"`
	Metrics  int64 `json:"metrics"`
	Runs     int64 `json:"runs"`
	Acks     int64 `json:"acks"`
	Tokens   int64 `json:"tokens"`
}
