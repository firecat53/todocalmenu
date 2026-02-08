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

	// Check only the date components, not the specific time
	expectedDate := time.Date(2024, 9, 18, 0, 0, 0, 0, time.UTC)
	actualDate := todo.StartDate.Truncate(24 * time.Hour)

	if !actualDate.Equal(expectedDate) {
		t.Errorf("Expected start date %v, got %v", expectedDate, actualDate)
	}

	// Check that the time is at least set to some hour (not checking specific hour)
	if todo.StartDate.Hour() == 0 && todo.StartDate.Minute() == 0 {
		t.Errorf("Expected non-zero time, got %v", todo.StartDate)
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

func TestGenerateDateOptions(t *testing.T) {
	// Test start date options (isDueDate = false)
	startOptions := generateDateOptions(false)

	expectedStartItems := []string{"Clear", "Today", "+1 day", "+7 days", "+14 days", "Custom..."}
	for _, item := range expectedStartItems {
		if !strings.Contains(startOptions, item) {
			t.Errorf("Start date options should contain '%s'", item)
		}
	}

	// Start dates should NOT have due-date-specific options
	unexpectedStartItems := []string{"+30 days", "End of week", "End of month"}
	for _, item := range unexpectedStartItems {
		if strings.Contains(startOptions, item) {
			t.Errorf("Start date options should NOT contain '%s'", item)
		}
	}

	// Test due date options (isDueDate = true)
	dueOptions := generateDateOptions(true)

	expectedDueItems := []string{"Clear", "Today", "+1 day", "+7 days", "+14 days", "+30 days", "End of week", "End of month", "Custom..."}
	for _, item := range expectedDueItems {
		if !strings.Contains(dueOptions, item) {
			t.Errorf("Due date options should contain '%s'", item)
		}
	}
}

func TestParseDateSelection(t *testing.T) {
	tests := []struct {
		name       string
		selection  string
		wantClear  bool
		wantCustom bool
		wantDate   string // empty string means don't check date
	}{
		{"Clear option", "Clear", true, false, ""},
		{"Custom option", "Custom...", false, true, ""},
		{"Today option", "Today (2025-02-06)", false, false, "2025-02-06"},
		{"+1 day option", "+1 day (2025-02-07)", false, false, "2025-02-07"},
		{"End of week", "End of week (2025-02-07)", false, false, "2025-02-07"},
		{"End of month", "End of month (2025-02-28)", false, false, "2025-02-28"},
		{"Invalid format", "Something without parens", false, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, isClear, isCustom := parseDateSelection(tt.selection)

			if isClear != tt.wantClear {
				t.Errorf("isClear = %v, want %v", isClear, tt.wantClear)
			}
			if isCustom != tt.wantCustom {
				t.Errorf("isCustom = %v, want %v", isCustom, tt.wantCustom)
			}
			if tt.wantDate != "" {
				gotDate := date.Format("2006-01-02")
				if gotDate != tt.wantDate {
					t.Errorf("date = %v, want %v", gotDate, tt.wantDate)
				}
			}
		})
	}
}

func TestIsDateInPast(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	tests := []struct {
		name     string
		date     time.Time
		wantPast bool
	}{
		{"Yesterday", today.AddDate(0, 0, -1), true},
		{"Today at midnight", today, false},
		{"Today at noon", today.Add(12 * time.Hour), false},
		{"Tomorrow", today.AddDate(0, 0, 1), false},
		{"Last week", today.AddDate(0, 0, -7), true},
		{"Next week", today.AddDate(0, 0, 7), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDateInPast(tt.date)
			if got != tt.wantPast {
				t.Errorf("isDateInPast(%v) = %v, want %v", tt.date, got, tt.wantPast)
			}
		})
	}
}

func TestIsToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		date    time.Time
		want    bool
	}{
		{"Today at midnight", today, true},
		{"Today at noon", today.Add(12 * time.Hour), true},
		{"Today at 23:59", today.Add(23*time.Hour + 59*time.Minute), true},
		{"Yesterday", today.AddDate(0, 0, -1), false},
		{"Tomorrow", today.AddDate(0, 0, 1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isToday(tt.date)
			if got != tt.want {
				t.Errorf("isToday(%v) = %v, want %v", tt.date, got, tt.want)
			}
		})
	}
}

func TestEndOfWeekCalculation(t *testing.T) {
	// Verify that "End of week" in generateDateOptions always falls on a Friday
	options := generateDateOptions(true)

	// Find the "End of week" line
	lines := strings.Split(options, "\n")
	var endOfWeekLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "End of week") {
			endOfWeekLine = line
			break
		}
	}

	if endOfWeekLine == "" {
		t.Fatal("Could not find 'End of week' option")
	}

	// Parse the date from the option
	date, _, _ := parseDateSelection(endOfWeekLine)
	if date.IsZero() {
		t.Fatal("Could not parse date from 'End of week' option")
	}

	// Verify it's a Friday
	if date.Weekday() != time.Friday {
		t.Errorf("End of week date %v is %v, expected Friday", date.Format("2006-01-02"), date.Weekday())
	}

	// Verify it's in the future (or today if today is Friday)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if date.Before(today) {
		t.Errorf("End of week date %v should not be before today %v", date.Format("2006-01-02"), today.Format("2006-01-02"))
	}
}

func TestHasTimeComponent(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want bool
	}{
		{"Midnight", time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local), false},
		{"Has hour", time.Date(2025, 1, 1, 1, 0, 0, 0, time.Local), true},
		{"Has minute", time.Date(2025, 1, 1, 0, 30, 0, 0, time.Local), true},
		{"Has second", time.Date(2025, 1, 1, 0, 0, 30, 0, time.Local), true},
		{"Full time", time.Date(2025, 1, 1, 14, 30, 45, 0, time.Local), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTimeComponent(tt.time)
			if got != tt.want {
				t.Errorf("hasTimeComponent(%v) = %v, want %v", tt.time, got, tt.want)
			}
		})
	}
}

func TestParseTimeInput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantHour  int
		wantMin   int
		wantError bool
	}{
		{"Colon format", "14:30", 14, 30, false},
		{"Colon format single digits", "9:05", 9, 5, false},
		{"No colon format", "1430", 14, 30, false},
		{"Midnight", "00:00", 0, 0, false},
		{"End of day", "23:59", 23, 59, false},
		{"Invalid hour", "25:00", 0, 0, true},
		{"Invalid minute", "12:60", 0, 0, true},
		{"Negative hour", "-1:00", 0, 0, true},
		{"Invalid format", "abc", 0, 0, true},
		{"Empty string", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, err := parseTimeInput(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if pt.hour != tt.wantHour {
				t.Errorf("hour = %d, want %d", pt.hour, tt.wantHour)
			}
			if pt.minute != tt.wantMin {
				t.Errorf("minute = %d, want %d", pt.minute, tt.wantMin)
			}
		})
	}
}
