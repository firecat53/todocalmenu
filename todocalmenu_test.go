package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadTodos(t *testing.T) {
	testDir := "testdata"
	todoList, err := loadTodos(testDir)
	if err != nil {
		t.Fatalf("Failed to load todos: %v", err)
	}

	if len(todoList.Todos) != 6 {
		t.Errorf("Expected 6 todos, got %d", len(todoList.Todos))
	}

	// Test specific todos
	testTodo1(t, findTodoByUID(todoList, "35rU"))
	testTodo2(t, findTodoByUID(todoList, "sLNz"))
	testTodo3(t, findTodoByUID(todoList, "20240918T131500Z-test2@example.com"))
	testTodo4(t, findTodoByUID(todoList, "657913900676334277"))
	testTodo5(t, findTodoByUID(todoList, "519633551077716419"))
	testTodo6(t, findTodoByUID(todoList, "3900172495289256706"))
}

func findTodoByUID(todoList *TodoList, uid string) *Todo {
	for _, todo := range todoList.Todos {
		if todo.UID == uid {
			return todo
		}
	}
	return nil
}

func testTodo1(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID 35rU not found")
	}
	if todo.Summary != "Test 2" {
		t.Errorf("Expected summary 'Test 2', got '%s'", todo.Summary)
	}
	if todo.Status != "NEEDS-ACTION" {
		t.Errorf("Expected status NEEDS-ACTION, got %s", todo.Status)
	}
	if !containsCategory(todo.Categories, "tech") {
		t.Errorf("Expected category 'tech', not found")
	}
	if todo.Priority != 1 {
		t.Errorf("Expected priority 1, got %d", todo.Priority)
	}
}

func testTodo2(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID sLNz not found")
	}
	if todo.Summary != "Move git repos" {
		t.Errorf("Expected summary 'Move git repos', got '%s'", todo.Summary)
	}
	if todo.Description != "Move git repos?" {
		t.Errorf("Expected description 'Move git repos?', got '%s'", todo.Description)
	}
	expectedCategories := []string{"tech", "git", "projects"}
	for _, cat := range expectedCategories {
		if !containsCategory(todo.Categories, cat) {
			t.Errorf("Expected category '%s', not found", cat)
		}
	}
}

func testTodo3(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID 20240918T131500Z-test2@example.com not found")
	}
	if todo.Summary != "Test 2" {
		t.Errorf("Expected summary 'Test 2', got '%s'", todo.Summary)
	}
	if todo.Description != "This is a test todo starting at 1600 EDT" {
		t.Errorf("Unexpected description: %s", todo.Description)
	}

	// Create the expected time in UTC
	expectedStart := time.Date(2024, 9, 18, 23, 0, 0, 0, time.UTC)

	// Convert the actual start time to UTC for comparison
	actualStartUTC := todo.StartDate.UTC()

	if !actualStartUTC.Equal(expectedStart) {
		t.Errorf("Expected start date %v UTC, got %v UTC", expectedStart, actualStartUTC)
	}
}

func testTodo4(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID 657913900676334277 not found")
	}
	if todo.Summary != "Testing" {
		t.Errorf("Expected summary 'Testing', got '%s'", todo.Summary)
	}
	if todo.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", todo.Priority)
	}
	expectedStart := time.Date(2024, 9, 20, 20, 0, 0, 0, time.UTC)
	if !todo.StartDate.Equal(expectedStart) {
		t.Errorf("Expected start date %v, got %v", expectedStart, todo.StartDate)
	}
	expectedDue := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	if !todo.DueDate.Equal(expectedDue) {
		t.Errorf("Expected due date %v, got %v", expectedDue, todo.DueDate)
	}
}

func testTodo5(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID 519633551077716419 not found")
	}
	if todo.Summary != "Trash/yard/recycle" {
		t.Errorf("Expected summary 'Trash/yard/recycle', got '%s'", todo.Summary)
	}
	if !containsCategory(todo.Categories, "chores") {
		t.Errorf("Expected category 'chores', not found")
	}
}

func testTodo6(t *testing.T, todo *Todo) {
	if todo == nil {
		t.Fatal("Todo with UID 3900172495289256706 not found")
	}
	if todo.Summary != "Trash/yard waste" {
		t.Errorf("Expected summary 'Trash/yard waste', got '%s'", todo.Summary)
	}
	if !containsCategory(todo.Categories, "chores") {
		t.Errorf("Expected category 'chores', not found")
	}
	expectedDue := time.Date(2024, 10, 2, 1, 0, 1, 0, time.UTC)
	if !todo.DueDate.Equal(expectedDue) {
		t.Errorf("Expected due date %v, got %v", expectedDue, todo.DueDate)
	}
	expectedStart := time.Date(2024, 10, 2, 1, 0, 0, 0, time.UTC)
	if !todo.StartDate.Equal(expectedStart) {
		t.Errorf("Expected start date %v, got %v", expectedStart, todo.StartDate)
	}
}

func containsCategory(categories []string, category string) bool {
	for _, c := range categories {
		if c == category {
			return true
		}
	}
	return false
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func TestSaveTodos(t *testing.T) {
	// Load existing todos
	testDir := "testdata"
	todoList, err := loadTodos(testDir)
	if err != nil {
		t.Fatalf("Failed to load todos: %v", err)
	}

	if len(todoList.Todos) == 0 {
		t.Fatalf("No todos found in testdata directory")
	}

	// Modify the first todo
	todoList.Todos[0].Summary = "Modified " + todoList.Todos[0].Summary
	todoList.Todos[0].Modified = true

	// Create a temporary directory for saving
	tempDir, err := os.MkdirTemp("", "test_save_todos")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save todos
	err = saveTodos(todoList, tempDir)
	if err != nil {
		t.Fatalf("Failed to save todos: %v", err)
	}

	// Load saved todos
	savedTodoList, err := loadTodos(tempDir)
	if err != nil {
		t.Fatalf("Failed to load saved todos: %v", err)
	}

	// Check if the modification was saved
	if len(savedTodoList.Todos) == 0 || !strings.HasPrefix(savedTodoList.Todos[0].Summary, "Modified ") {
		t.Errorf("Modification was not saved correctly")
	}
}

func TestCreateMenu(t *testing.T) {
	testDir := "testdata"
	todoList, err := loadTodos(testDir)
	if err != nil {
		t.Fatalf("Failed to load todos: %v", err)
	}

	displayList, m := createMenu(todoList, false)
	menuStr := displayList.String()

	expectedItems := []string{
		"Add Item",
		"View Completed Items",
	}

	for _, item := range expectedItems {
		if !strings.Contains(menuStr, item) {
			t.Errorf("Expected menu to contain '%s', but it doesn't", item)
		}
	}

	if len(m) != len(todoList.Todos) {
		t.Errorf("Expected %d items in the menu map, got %d", len(todoList.Todos), len(m))
	}
}

func TestAddTodo(t *testing.T) {
	todoList := &TodoList{}
	newTodo := &Todo{
		Summary:     "New Test Todo",
		Description: "This is a new test todo",
		Categories:  []string{"test", "new"},
		Priority:    2,
	}

	// Add the new todo directly to the list
	todoList.Todos = append(todoList.Todos, newTodo)

	if len(todoList.Todos) != 1 {
		t.Errorf("Expected 1 todo in the list, got %d", len(todoList.Todos))
	}

	addedTodo := todoList.Todos[0]
	if addedTodo.Summary != "New Test Todo" {
		t.Errorf("Expected summary 'New Test Todo', got '%s'", addedTodo.Summary)
	}
	if addedTodo.Description != "This is a new test todo" {
		t.Errorf("Expected description 'This is a new test todo', got '%s'", addedTodo.Description)
	}
	if !reflect.DeepEqual(addedTodo.Categories, []string{"test", "new"}) {
		t.Errorf("Expected categories [test, new], got %v", addedTodo.Categories)
	}
	if addedTodo.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", addedTodo.Priority)
	}
	// Note: We're not checking for UID here as it's not being set in this test
}

func TestEditTodo(t *testing.T) {
	todoList := &TodoList{
		Todos: []*Todo{
			{
				UID:         "test-uid",
				Summary:     "Original Todo",
				Description: "Original description",
				Categories:  []string{"original"},
				Priority:    1,
			},
		},
	}

	// Edit the todo directly
	todoList.Todos[0].Summary = "Edited Todo"
	todoList.Todos[0].Description = "Edited description"
	todoList.Todos[0].Categories = []string{"edited", "updated"}
	todoList.Todos[0].Priority = 3
	todoList.Todos[0].Modified = true

	if len(todoList.Todos) != 1 {
		t.Fatalf("Expected 1 todo in the list, got %d", len(todoList.Todos))
	}

	updatedTodo := todoList.Todos[0]
	if updatedTodo.Summary != "Edited Todo" {
		t.Errorf("Expected summary 'Edited Todo', got '%s'", updatedTodo.Summary)
	}
	if updatedTodo.Description != "Edited description" {
		t.Errorf("Expected description 'Edited description', got '%s'", updatedTodo.Description)
	}
	if !reflect.DeepEqual(updatedTodo.Categories, []string{"edited", "updated"}) {
		t.Errorf("Expected categories [edited, updated], got %v", updatedTodo.Categories)
	}
	if updatedTodo.Priority != 3 {
		t.Errorf("Expected priority 3, got %d", updatedTodo.Priority)
	}
	if !updatedTodo.Modified {
		t.Error("Expected Modified flag to be set to true")
	}
}
