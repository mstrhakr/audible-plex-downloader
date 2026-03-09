CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    asin TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    author_asin TEXT NOT NULL DEFAULT '',
    narrator TEXT NOT NULL DEFAULT '',
    publisher TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    duration INTEGER NOT NULL DEFAULT 0,
    series TEXT NOT NULL DEFAULT '',
    series_position TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    purchase_date TIMESTAMP,
    release_date TIMESTAMP,
    drm_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'new',
    file_path TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_books_asin ON books(asin);
CREATE INDEX IF NOT EXISTS idx_books_status ON books(status);
CREATE INDEX IF NOT EXISTS idx_books_author ON books(author);

CREATE TABLE IF NOT EXISTS download_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    asin TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    progress REAL NOT NULL DEFAULT 0.0,
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_download_queue_status ON download_queue(status);
CREATE INDEX IF NOT EXISTS idx_download_queue_book_id ON download_queue(book_id);

CREATE TABLE IF NOT EXISTS sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    books_found INTEGER NOT NULL DEFAULT 0,
    books_added INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running',
    error TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL DEFAULT '',
    marketplace TEXT NOT NULL DEFAULT '',
    credentials BLOB,
    is_active INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
