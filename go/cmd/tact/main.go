package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fabiobrady/tact/internal/model"
	"github.com/fabiobrady/tact/internal/todo"
	"github.com/fabiobrady/tact/internal/tui"
	"github.com/spf13/cobra"
)

func defaultProject() string {
	dir, err := os.Getwd()
	if err != nil {
		return "default"
	}
	return filepath.Base(dir)
}

func main() {
	root := &cobra.Command{
		Use:   "tact",
		Short: "Control tower for AI coding sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
		SilenceUsage: true,
	}

	// --- todo subcommand ---
	todoCmd := &cobra.Command{Use: "todo", Short: "Manage shared todos"}

	var project, tags string

	addCmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a todo item",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")
			var t []string
			if tags != "" {
				for _, s := range strings.Split(tags, ",") {
					if s = strings.TrimSpace(s); s != "" {
						t = append(t, s)
					}
				}
			}
			item, err := todo.AddTodo(project, text, "", t)
			if err != nil {
				return err
			}
			fmt.Printf("Added [%s] to %s: %s\n", item.ID, project, text)
			return nil
		},
	}
	addCmd.Flags().StringVarP(&project, "project", "p", defaultProject(), "Project name")
	addCmd.Flags().StringVarP(&tags, "tags", "t", "", "Comma-separated tags")

	doneCmd := &cobra.Command{
		Use: "done [id]", Short: "Mark todo as done", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := todo.UpdateTodo(todo.Slug(project), args[0], model.TodoDone)
			if err != nil {
				return err
			}
			if ok {
				fmt.Println("Marked as done")
			} else {
				fmt.Fprintln(os.Stderr, "Not found")
			}
			return nil
		},
	}
	doneCmd.Flags().StringVarP(&project, "project", "p", defaultProject(), "Project name")

	startCmd := &cobra.Command{
		Use: "start [id]", Short: "Mark todo as in-progress", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := todo.UpdateTodo(todo.Slug(project), args[0], model.TodoInProgress)
			if err != nil {
				return err
			}
			if ok {
				fmt.Println("Started")
			} else {
				fmt.Fprintln(os.Stderr, "Not found")
			}
			return nil
		},
	}
	startCmd.Flags().StringVarP(&project, "project", "p", defaultProject(), "Project name")

	rmCmd := &cobra.Command{
		Use: "rm [id]", Short: "Remove a todo", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ok, err := todo.RemoveTodo(todo.Slug(project), args[0])
			if err != nil {
				return err
			}
			if ok {
				fmt.Println("Removed")
			} else {
				fmt.Fprintln(os.Stderr, "Not found")
			}
			return nil
		},
	}
	rmCmd.Flags().StringVarP(&project, "project", "p", defaultProject(), "Project name")

	listCmd := &cobra.Command{
		Use: "list", Short: "List todos",
		Run: func(cmd *cobra.Command, args []string) {
			if project != defaultProject() || project != "" {
				slug := todo.Slug(project)
				pt := todo.LoadProjectTodos(slug)
				printTodos(pt.Project, pt.Items)
			} else {
				for slug, pt := range todo.ListAllTodos() {
					name := pt.Project
					if name == "" {
						name = slug
					}
					printTodos(name, pt.Items)
				}
			}
		},
	}
	listCmd.Flags().StringVarP(&project, "project", "p", defaultProject(), "Project name")

	todoCmd.AddCommand(addCmd, doneCmd, startCmd, rmCmd, listCmd)
	root.AddCommand(todoCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func printTodos(project string, items []model.TodoItem) {
	if len(items) == 0 {
		fmt.Printf("  %s: (no items)\n", project)
		return
	}
	fmt.Printf("  %s:\n", project)
	for _, item := range items {
		tags := ""
		if len(item.Tags) > 0 {
			tags = " [" + strings.Join(item.Tags, ", ") + "]"
		}
		fmt.Printf("    %s [%s] %s%s\n", item.Status.Icon(), item.ID, item.Text, tags)
	}
}
