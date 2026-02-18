package db

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

func Init(dbPath string) error {
	// Ensure directory exists
	dataDir := filepath.Dir(dbPath)
	if dataDir != "" && dataDir != "." {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}
	db, err := sqlx.Connect("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return migrate()
}

func migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    avatar TEXT DEFAULT '',
    prompt TEXT NOT NULL,
    is_user_playable BOOLEAN DEFAULT FALSE,
    model_name TEXT DEFAULT 'gpt-3.5-turbo',
    temperature REAL DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 1000,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    setting TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS room_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    participant_type TEXT DEFAULT 'ai',
    is_user BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    UNIQUE(room_id, character_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    participant_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (participant_id) REFERENCES room_participants(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    room_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    message_from INTEGER NOT NULL,
    message_to INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    api_endpoint TEXT DEFAULT 'https://api.openai.com/v1',
    api_key TEXT DEFAULT '',
    default_model TEXT DEFAULT 'gpt-3.5-turbo'
);

INSERT OR IGNORE INTO config (id) VALUES (1);
`

	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: add updated_at to messages if not exists
	_, _ = DB.Exec(`ALTER TABLE messages ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP`)

	// Migration: create llm_call_logs table
	_, err = DB.Exec(`
CREATE TABLE IF NOT EXISTS llm_call_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    room_id INTEGER NOT NULL,
    call_type TEXT NOT NULL,
    model_name TEXT NOT NULL,
    temperature REAL DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 1000,
    request_body TEXT NOT NULL,
    response_body TEXT,
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    latency_ms INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_llm_logs_message_id ON llm_call_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_llm_logs_created_at ON llm_call_logs(created_at);
`)
	if err != nil {
		return err
	}

	// Migration: rename system_prompt to setting in rooms table
	// SQLite doesn't support RENAME COLUMN directly, so we need to check and migrate
	var hasSettingColumn bool
	err = DB.Get(&hasSettingColumn, `SELECT COUNT(*) > 0 FROM pragma_table_info('rooms') WHERE name = 'setting'`)
	if err != nil {
		// Fallback: try to add setting column anyway
		hasSettingColumn = false
	}

	if !hasSettingColumn {
		// Add setting column
		_, _ = DB.Exec(`ALTER TABLE rooms ADD COLUMN setting TEXT DEFAULT ''`)
		// Copy data from system_prompt to setting if exists
		_, _ = DB.Exec(`UPDATE rooms SET setting = system_prompt WHERE setting = '' AND system_prompt IS NOT NULL`)
	}

	// Migration: create orchestrator_decisions table
	_, err = DB.Exec(`
CREATE TABLE IF NOT EXISTS orchestrator_decisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id INTEGER NOT NULL,
    room_id INTEGER NOT NULL,
    step_order INTEGER NOT NULL,
    step_type TEXT NOT NULL,
    input_data TEXT NOT NULL,
    output_data TEXT NOT NULL,
    llm_call_log_id INTEGER DEFAULT 0,
    reason TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE CASCADE,
    FOREIGN KEY (llm_call_log_id) REFERENCES llm_call_logs(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_decisions_message_id ON orchestrator_decisions(message_id);
CREATE INDEX IF NOT EXISTS idx_decisions_step_order ON orchestrator_decisions(message_id, step_order);
CREATE INDEX IF NOT EXISTS idx_decisions_llm_call ON orchestrator_decisions(llm_call_log_id);
`)
	if err != nil {
		return err
	}

	return nil
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
