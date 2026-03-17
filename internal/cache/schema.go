package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type Table struct {
	Schema      string   `json:"schema"`
	Name        string   `json:"name"`
	Columns     []Column `json:"columns"`
	Partitioned bool     `json:"partitioned,omitempty"`
}

type Index struct {
	Schema  string `json:"schema"`
	Table   string `json:"table"`
	Name    string `json:"name"`
	Columns []string `json:"columns,omitempty"`
}

type Function struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	ReturnType string `json:"return_type"`
	Arguments  string `json:"arguments"`
}

type SchemaSnapshot struct {
	Profile     string     `json:"profile"`
	RefreshedAt time.Time  `json:"refreshed_at"`
	Schemas     []string   `json:"schemas"`
	Tables      []Table    `json:"tables"`
	Indexes     []Index    `json:"indexes"`
	Functions   []Function `json:"functions"`
}

type SchemaCache struct {
	mu       sync.RWMutex
	snapshot *SchemaSnapshot
	diskPath string
}

func NewSchemaCache(diskPath string) *SchemaCache {
	return &SchemaCache{diskPath: diskPath}
}

func (c *SchemaCache) Load() error {
	data, err := os.ReadFile(c.diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap SchemaSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return err
	}
	c.mu.Lock()
	c.snapshot = &snap
	c.mu.Unlock()
	return nil
}

func (c *SchemaCache) Set(snap *SchemaSnapshot) error {
	c.mu.Lock()
	c.snapshot = snap
	c.mu.Unlock()
	return c.persist(snap)
}

func (c *SchemaCache) persist(snap *SchemaSnapshot) error {
	if err := os.MkdirAll(filepath.Dir(c.diskPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.diskPath, data, 0o644)
}

func (c *SchemaCache) Get() *SchemaSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

func (c *SchemaCache) TableNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	names := make([]string, 0, len(c.snapshot.Tables))
	for _, t := range c.snapshot.Tables {
		names = append(names, t.Name)
	}
	return names
}

func (c *SchemaCache) ColumnNamesForTable(tableName string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	for _, t := range c.snapshot.Tables {
		if t.Name == tableName {
			cols := make([]string, 0, len(t.Columns))
			for _, col := range t.Columns {
				cols = append(cols, col.Name)
			}
			return cols
		}
	}
	return nil
}

func (c *SchemaCache) AllColumnNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var cols []string
	for _, t := range c.snapshot.Tables {
		for _, col := range t.Columns {
			if _, ok := seen[col.Name]; !ok {
				seen[col.Name] = struct{}{}
				cols = append(cols, col.Name)
			}
		}
	}
	return cols
}

func (c *SchemaCache) SchemaNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	return c.snapshot.Schemas
}

func (c *SchemaCache) FunctionNames() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	names := make([]string, 0, len(c.snapshot.Functions))
	for _, f := range c.snapshot.Functions {
		names = append(names, f.Name)
	}
	return names
}
