package todo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
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

// LoadProjectTodos reads a project's todo file with a shared lock.
func LoadProjectTodos(slug string) model.ProjectTodos {
	p := todoPath(slug)
	f, err := os.Open(p)
	if err != nil {
		return model.ProjectTodos{Project: slug}
	}
	defer f.Close()
	syscall.Flock(int(f.Fd()), syscall.LOCK_SH)
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	var todos model.ProjectTodos
	if json.NewDecoder(f).Decode(&todos) != nil {
		return model.ProjectTodos{Project: slug}
	}
	return todos
}

// SaveProjectTodos writes a project's todo file with an exclusive lock.
func SaveProjectTodos(todos model.ProjectTodos) error {
	model.EnsureDirs()
	slug := Slug(todos.Project)
	if slug == "" {
		slug = todos.Project
	}
	p := todoPath(slug)
	todos.UpdatedAt = time.Now()

	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := json.MarshalIndent(todos, "", "  ")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
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

// UpdateTodo changes a todo item's status.
func UpdateTodo(slug, todoID string, status model.TodoStatus) bool {
	todos := LoadProjectTodos(slug)
	for i := range todos.Items {
		if todos.Items[i].ID == todoID {
			todos.Items[i].Status = status
			todos.Items[i].UpdatedAt = time.Now()
			SaveProjectTodos(todos)
			return true
		}
	}
	return false
}

// RemoveTodo deletes a todo item.
func RemoveTodo(slug, todoID string) bool {
	todos := LoadProjectTodos(slug)
	n := len(todos.Items)
	filtered := make([]model.TodoItem, 0, n)
	for _, item := range todos.Items {
		if item.ID != todoID {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == n {
		return false
	}
	todos.Items = filtered
	SaveProjectTodos(todos)
	return true
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
