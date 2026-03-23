package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fabiobrady/tact/internal/model"
)

var slugRe = regexp.MustCompile(`[^a-z0-9_-]+`)

// Slug converts a project name to a filesystem-safe slug.
func Slug(name string) string {
	s := strings.ToLower(name)
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func todoPath(slug string) string {
	return filepath.Join(model.TodosDir, slug+".json")
}

// LoadProjectTodos reads a project's todo file.
// No locking needed: SaveProjectTodos writes atomically via os.Rename.
func LoadProjectTodos(slug string) model.ProjectTodos {
	p := todoPath(slug)
	f, err := os.Open(p)
	if err != nil {
		return model.ProjectTodos{Project: slug}
	}
	defer f.Close()

	var todos model.ProjectTodos
	if json.NewDecoder(f).Decode(&todos) != nil {
		return model.ProjectTodos{Project: slug}
	}
	return todos
}

// SaveProjectTodos writes a project's todo file atomically via a temp file
// and os.Rename, so readers always see a complete file.
func SaveProjectTodos(todos model.ProjectTodos) error {
	model.EnsureDirs()
	slug := Slug(todos.Project)
	if slug == "" {
		slug = todos.Project
	}
	p := todoPath(slug)
	todos.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(model.TodosDir, ".tact-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op on success (file renamed), cleans up on failure

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, p)
}

// AddTodo adds a new todo item to a project.
func AddTodo(project, text, sourceSession string, tags []string) (model.TodoItem, error) {
	slug := Slug(project)
	todos := LoadProjectTodos(slug)
	todos.Project = project
	item := model.TodoItem{
		ID:            model.NewTodoID(),
		Text:          text,
		Status:        model.TodoPending,
		SourceSession: sourceSession,
		Project:       project,
		Tags:          tags,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	todos.Items = append(todos.Items, item)
	return item, SaveProjectTodos(todos)
}

// UpdateTodo changes a todo item's status. Returns false if the ID was not found.
func UpdateTodo(slug, todoID string, status model.TodoStatus) (bool, error) {
	todos := LoadProjectTodos(slug)
	for i := range todos.Items {
		if todos.Items[i].ID == todoID {
			todos.Items[i].Status = status
			todos.Items[i].UpdatedAt = time.Now()
			return true, SaveProjectTodos(todos)
		}
	}
	return false, nil
}

// RemoveTodo deletes a todo item. Returns false if the ID was not found.
func RemoveTodo(slug, todoID string) (bool, error) {
	todos := LoadProjectTodos(slug)
	n := len(todos.Items)
	filtered := make([]model.TodoItem, 0, n)
	for _, item := range todos.Items {
		if item.ID != todoID {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == n {
		return false, nil
	}
	todos.Items = filtered
	return true, SaveProjectTodos(todos)
}

// ListAllTodos loads all project todo lists.
func ListAllTodos() map[string]model.ProjectTodos {
	result := make(map[string]model.ProjectTodos)
	matches, _ := filepath.Glob(filepath.Join(model.TodosDir, "*.json"))
	for _, m := range matches {
		slug := strings.TrimSuffix(filepath.Base(m), ".json")
		result[slug] = LoadProjectTodos(slug)
	}
	return result
}
