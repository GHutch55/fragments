-- Users table (simple session-based accounts)
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    display_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Folders for organizing snippets (optional hierarchy)
CREATE TABLE folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    parent_id INTEGER, -- for nested folders (nullable)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE SET NULL,
    UNIQUE(user_id, name, parent_id) -- prevent duplicate folder names in same location
);

-- Main snippets table
CREATE TABLE snippets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    folder_id INTEGER, -- nullable, snippets can exist without folders
    title TEXT NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'text', -- 'javascript', 'python', 'go', etc.
    is_favorite BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE SET NULL
);

-- Tags for flexible organization
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    color TEXT, -- hex color for UI (optional)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name) -- users can't have duplicate tag names
);

-- Many-to-many relationship between snippets and tags
CREATE TABLE snippet_tags (
    snippet_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (snippet_id, tag_id),
    FOREIGN KEY (snippet_id) REFERENCES snippets(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_snippets_user_id ON snippets(user_id);
CREATE INDEX idx_snippets_folder_id ON snippets(folder_id);
CREATE INDEX idx_snippets_language ON snippets(language);
CREATE INDEX idx_snippets_created_at ON snippets(created_at DESC);
CREATE INDEX idx_folders_user_id ON folders(user_id);
CREATE INDEX idx_tags_user_id ON tags(user_id);

-- Full-text search support (SQLite FTS5)
CREATE VIRTUAL TABLE snippets_fts USING fts5(
    title, 
    description, 
    content,
    content_id UNINDEXED -- reference back to snippets.id
);

-- Trigger to keep FTS table in sync
CREATE TRIGGER snippets_fts_insert AFTER INSERT ON snippets BEGIN
    INSERT INTO snippets_fts(title, description, content, content_id) 
    VALUES (new.title, new.description, new.content, new.id);
END;

CREATE TRIGGER snippets_fts_update AFTER UPDATE ON snippets BEGIN
    UPDATE snippets_fts 
    SET title = new.title, description = new.description, content = new.content
    WHERE content_id = new.id;
END;

CREATE TRIGGER snippets_fts_delete AFTER DELETE ON snippets BEGIN
    DELETE FROM snippets_fts WHERE content_id = old.id;
END;
