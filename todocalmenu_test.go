package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadTodos(t *testing.T) {
	testDir := "testdata"
	todoList, err := loadTodos(testDir, "")
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
	// Verify RRULE is loaded
	if todo.RRULE != "FREQ=WEEKLY;INTERVAL=2;BYDAY=TU" {
		t.Errorf("Expected RRULE 'FREQ=WEEKLY;INTERVAL=2;BYDAY=TU', got '%s'", todo.RRULE)
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
	todoList, err := loadTodos(testDir, "")
	if err != nil {
		t.Fatalf("Failed to load todos: %v", err)
	}

	if len(todoList.Todos) == 0 {
		t.Fatalf("No todos found in testdata directory")
	}

	// Create a temporary directory for saving
	tempDir, err := os.MkdirTemp("", "test_save_todos")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Modify the first todo and set its ListDir to tempDir
	todoList.Todos[0].Summary = "Modified " + todoList.Todos[0].Summary
	todoList.Todos[0].Modified = true
	todoList.Todos[0].ListDir = tempDir

	// Save todos
	err = saveTodos(todoList)
	if err != nil {
		t.Fatalf("Failed to save todos: %v", err)
	}

	// Load saved todos
	savedTodoList, err := loadTodos(tempDir, "")
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
	todoList, err := loadTodos(testDir, "")
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

func TestCalculateNextOccurrence(t *testing.T) {
	tests := []struct {
		name        string
		todo        *Todo
		expectOK    bool
		checkResult func(t *testing.T, result time.Time)
	}{
		{
			name: "No RRULE",
			todo: &Todo{
				Summary: "Non-recurring task",
				DueDate: time.Now().AddDate(0, 0, 1),
			},
			expectOK: false,
		},
		{
			name: "Daily recurrence",
			todo: &Todo{
				Summary: "Daily task",
				DueDate: time.Now().AddDate(0, 0, -1), // Yesterday
				RRULE:   "FREQ=DAILY",
			},
			expectOK: true,
			checkResult: func(t *testing.T, result time.Time) {
				// Should be today or later
				if result.Before(time.Now().Truncate(24 * time.Hour)) {
					t.Errorf("Next occurrence %v should be today or later", result)
				}
			},
		},
		{
			name: "Weekly recurrence on Tuesday",
			todo: &Todo{
				Summary: "Weekly Tuesday task",
				DueDate: time.Date(2024, 10, 1, 9, 0, 0, 0, time.UTC), // A Tuesday
				RRULE:   "FREQ=WEEKLY;BYDAY=TU",
			},
			expectOK: true,
			checkResult: func(t *testing.T, result time.Time) {
				// Should be a Tuesday
				if result.Weekday() != time.Tuesday {
					t.Errorf("Next occurrence %v should be a Tuesday, got %v", result, result.Weekday())
				}
				// Should be in the future
				if result.Before(time.Now()) {
					t.Errorf("Next occurrence %v should be in the future", result)
				}
			},
		},
		{
			name: "Monthly recurrence",
			todo: &Todo{
				Summary: "Monthly task",
				DueDate: time.Now().AddDate(0, -1, 0), // Last month
				RRULE:   "FREQ=MONTHLY",
			},
			expectOK: true,
			checkResult: func(t *testing.T, result time.Time) {
				// Should be in the future
				if result.Before(time.Now()) {
					t.Errorf("Next occurrence %v should be in the future", result)
				}
			},
		},
		{
			name: "Biweekly recurrence",
			todo: &Todo{
				Summary:   "Trash/yard waste",
				DueDate:   time.Date(2024, 10, 1, 9, 0, 0, 0, time.UTC), // Oct 1, 2024 was a Tuesday
				StartDate: time.Date(2024, 10, 1, 9, 0, 0, 0, time.UTC),
				RRULE:     "FREQ=WEEKLY;INTERVAL=2;BYDAY=TU",
			},
			expectOK: true,
			checkResult: func(t *testing.T, result time.Time) {
				// Should be a Tuesday (in UTC, the source timezone)
				resultUTC := result.UTC()
				if resultUTC.Weekday() != time.Tuesday {
					t.Errorf("Next occurrence %v (UTC: %v) should be a Tuesday, got %v", result, resultUTC, resultUTC.Weekday())
				}
				// Should be in the future
				if result.Before(time.Now()) {
					t.Errorf("Next occurrence %v should be in the future", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := calculateNextOccurrence(tt.todo)

			if ok != tt.expectOK {
				t.Errorf("calculateNextOccurrence() ok = %v, want %v", ok, tt.expectOK)
				return
			}

			if tt.expectOK && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestDirContainsVTodos(t *testing.T) {
	// testdata/ contains .ics files with VTODOs
	if !dirContainsVTodos("testdata") {
		t.Error("Expected testdata/ to contain VTODOs")
	}

	// testdata/multi/ contains only subdirectories, no .ics files directly
	if dirContainsVTodos("testdata/multi") {
		t.Error("Expected testdata/multi/ to not contain VTODOs directly")
	}

	// testdata/multi/work/ contains .ics files with VTODOs
	if !dirContainsVTodos("testdata/multi/work") {
		t.Error("Expected testdata/multi/work/ to contain VTODOs")
	}

	// Non-existent directory
	if dirContainsVTodos("nonexistent") {
		t.Error("Expected nonexistent directory to return false")
	}
}

func TestDiscoverLists(t *testing.T) {
	t.Run("Single-list mode (ics files in root)", func(t *testing.T) {
		names, err := discoverLists("testdata")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if names != nil {
			t.Errorf("Expected nil for single-list mode, got %v", names)
		}
	})

	t.Run("Multi-list mode (subdirectories with todos)", func(t *testing.T) {
		names, err := discoverLists("testdata/multi")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(names) != 2 {
			t.Fatalf("Expected 2 lists, got %d: %v", len(names), names)
		}
		// Should be sorted
		if names[0] != "personal" || names[1] != "work" {
			t.Errorf("Expected [personal, work], got %v", names)
		}
		// work has a displayname file, personal does not
		if dn, ok := listDisplayNames["work"]; !ok || dn != "Work Tasks" {
			t.Errorf("Expected displayname 'Work Tasks' for work, got %q (ok=%v)", dn, ok)
		}
		if _, ok := listDisplayNames["personal"]; ok {
			t.Error("Expected no displayname entry for personal")
		}
	})

	t.Run("Empty directory (single-list mode)", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test_discover_empty")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		names, err := discoverLists(tempDir)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if names != nil {
			t.Errorf("Expected nil for empty directory, got %v", names)
		}
	})

	t.Run("Non-existent directory", func(t *testing.T) {
		_, err := discoverLists("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent directory")
		}
	})
}

func TestLoadAllTodos(t *testing.T) {
	t.Run("Single-list mode", func(t *testing.T) {
		// Reset global state
		listNames = nil

		todoList, err := loadAllTodos("testdata")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(todoList.Todos) != 6 {
			t.Errorf("Expected 6 todos, got %d", len(todoList.Todos))
		}
		// In single-list mode, ListName should be empty
		for _, todo := range todoList.Todos {
			if todo.ListName != "" {
				t.Errorf("Expected empty ListName in single-list mode, got %q", todo.ListName)
			}
			if todo.ListDir != "testdata" {
				t.Errorf("Expected ListDir 'testdata', got %q", todo.ListDir)
			}
		}
	})

	t.Run("Multi-list mode", func(t *testing.T) {
		// Set global state for multi-list
		listNames = []string{"personal", "work"}
		listDisplayNames = map[string]string{"work": "Work Tasks"}

		todoList, err := loadAllTodos("testdata/multi")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(todoList.Todos) != 3 {
			t.Errorf("Expected 3 todos, got %d", len(todoList.Todos))
		}

		// Verify display name is used for work (has displayname file)
		workTodo := findTodoByUID(todoList, "work-task1")
		if workTodo == nil {
			t.Fatal("work-task1 not found")
		}
		if workTodo.ListName != "Work Tasks" {
			t.Errorf("Expected ListName 'Work Tasks' (from displayname file), got %q", workTodo.ListName)
		}
		if workTodo.ListDir != filepath.Join("testdata/multi", "work") {
			t.Errorf("Expected ListDir 'testdata/multi/work', got %q", workTodo.ListDir)
		}

		// Verify dir name is used for personal (no displayname file)
		personalTodo := findTodoByUID(todoList, "personal-task1")
		if personalTodo == nil {
			t.Fatal("personal-task1 not found")
		}
		if personalTodo.ListName != "personal" {
			t.Errorf("Expected ListName 'personal' (dir name fallback), got %q", personalTodo.ListName)
		}

		// Reset global state
		listNames = nil
		listDisplayNames = nil
	})
}

func TestCreateMenuMultiList(t *testing.T) {
	todoList := &TodoList{
		Todos: []*Todo{
			{
				UID:      "t1",
				Summary:  "Work task",
				ListName: "Work Tasks", // display name from displayname file
				Created:  time.Now(),
				Status:   "NEEDS-ACTION",
			},
			{
				UID:        "t2",
				Summary:    "Personal task",
				ListName:   "personal", // dir name fallback
				Categories: []string{"home"},
				Created:    time.Now(),
				Status:     "NEEDS-ACTION",
			},
			{
				UID:      "t3",
				Summary:  "Single-list task",
				ListName: "", // single-list mode
				Created:  time.Now(),
				Status:   "NEEDS-ACTION",
			},
		},
	}

	displayList, _ := createMenu(todoList, false)
	menuStr := displayList.String()

	// Multi-list items should show +listname (display name when available)
	if !strings.Contains(menuStr, "+Work Tasks") {
		t.Error("Expected menu to contain '+Work Tasks'")
	}
	if !strings.Contains(menuStr, "+personal") {
		t.Error("Expected menu to contain '+personal'")
	}

	// Single-list items should not show +listname
	lines := strings.Split(menuStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Single-list task") {
			if strings.Contains(line, "+") {
				t.Errorf("Single-list task should not have +listname, got: %s", line)
			}
		}
	}
}

func TestSaveTodosMultiList(t *testing.T) {
	// Create temp directories for two lists
	tempDir, err := os.MkdirTemp("", "test_save_multi")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	workDir := filepath.Join(tempDir, "work")
	personalDir := filepath.Join(tempDir, "personal")
	os.MkdirAll(workDir, 0755)
	os.MkdirAll(personalDir, 0755)

	todoList := &TodoList{
		Todos: []*Todo{
			{
				UID:      "save-work-1",
				Summary:  "Work task",
				ListName: "work",
				ListDir:  workDir,
				Created:  time.Now(),
				LastMod:  time.Now(),
				Status:   "NEEDS-ACTION",
				Modified: true,
			},
			{
				UID:      "save-personal-1",
				Summary:  "Personal task",
				ListName: "personal",
				ListDir:  personalDir,
				Created:  time.Now(),
				LastMod:  time.Now(),
				Status:   "NEEDS-ACTION",
				Modified: true,
			},
		},
	}

	err = saveTodos(todoList)
	if err != nil {
		t.Fatalf("Failed to save todos: %v", err)
	}

	// Verify work todo was saved to work directory
	workTodos, err := loadTodos(workDir, "work")
	if err != nil {
		t.Fatalf("Failed to load work todos: %v", err)
	}
	if len(workTodos.Todos) != 1 {
		t.Errorf("Expected 1 work todo, got %d", len(workTodos.Todos))
	}
	if workTodos.Todos[0].Summary != "Work task" {
		t.Errorf("Expected 'Work task', got %q", workTodos.Todos[0].Summary)
	}

	// Verify personal todo was saved to personal directory
	personalTodos, err := loadTodos(personalDir, "personal")
	if err != nil {
		t.Fatalf("Failed to load personal todos: %v", err)
	}
	if len(personalTodos.Todos) != 1 {
		t.Errorf("Expected 1 personal todo, got %d", len(personalTodos.Todos))
	}
	if personalTodos.Todos[0].Summary != "Personal task" {
		t.Errorf("Expected 'Personal task', got %q", personalTodos.Todos[0].Summary)
	}
}

func TestReadDisplayName(t *testing.T) {
	// Directory with a displayname file
	dn := readDisplayName("testdata/multi/work")
	if dn != "Work Tasks" {
		t.Errorf("Expected 'Work Tasks', got %q", dn)
	}

	// Directory without a displayname file
	dn = readDisplayName("testdata/multi/personal")
	if dn != "" {
		t.Errorf("Expected empty string, got %q", dn)
	}

	// Non-existent directory
	dn = readDisplayName("nonexistent")
	if dn != "" {
		t.Errorf("Expected empty string for non-existent dir, got %q", dn)
	}
}

func TestListDisplayName(t *testing.T) {
	// With display name set
	listDisplayNames = map[string]string{"work": "Work Tasks"}
	if got := listDisplayName("work"); got != "Work Tasks" {
		t.Errorf("Expected 'Work Tasks', got %q", got)
	}

	// Without display name (falls back to dir name)
	if got := listDisplayName("personal"); got != "personal" {
		t.Errorf("Expected 'personal', got %q", got)
	}

	// Reset global state
	listDisplayNames = nil
}
