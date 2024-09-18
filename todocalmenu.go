package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

var hideCreatedDatePtr = flag.Bool("hide-created-date", false, "Hide created date in the list view")
var optsPtr = flag.String("opts", "", "Additional Rofi/Dmenu options")
var thresholdPtr = flag.Bool("threshold", false, "Hide items before their threshold date")
var todoPtr = flag.String("todo", "./todos", "Path to todo directory")
var cmdPtr = flag.String("cmd", "dmenu", "Dmenu command to use (dmenu, rofi, wofi, etc)")

type Todo struct {
	UID         string
	Summary     string
	Description string
	Categories  []string
	Status      string
	Created     time.Time
	LastMod     time.Time
	DueDate     time.Time
	Priority    int
	StartDate   time.Time
	Modified    bool // New field to track changes in the current session
}

type TodoList struct {
	Todos []*Todo
}

func main() {
	flag.Parse()

	// Ensure the todo directory exists
	if err := os.MkdirAll(*todoPtr, 0755); err != nil {
		log.Fatalf("Failed to create todo directory: %v", err)
	}

	todoList, err := loadTodos(*todoPtr)
	if err != nil {
		log.Fatal(err.Error())
	}
	for edit := true; edit; {
		displayList, m := createMenu(todoList, false)
		out, _ := display(displayList.String(), *todoPtr)
		switch {
		case out == "Add Item":
			addItem(todoList)
		case out == "View Completed Items":
			viewCompletedItems(todoList)
		case out != "":
			t := todoList.Todos[m[out]]
			editItem(t, todoList)
		default:
			edit = false
		}
	}
	if err := saveTodos(todoList, *todoPtr); err != nil {
		log.Fatal(err.Error())
	}
}

func loadTodos(dirPath string) (*TodoList, error) {
	todoList := &TodoList{}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".ics" {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		cal, err := loadICSFile(filePath)
		if err != nil {
			log.Printf("Error loading %s: %v", filePath, err)
			continue
		}

		for _, component := range cal.Components {
			if todo, ok := component.(*ics.VTodo); ok {
				todoList.Todos = append(todoList.Todos, convertVTodoToTodo(todo))
			}
		}
	}

	if len(todoList.Todos) == 0 {
		log.Printf("Warning: No todos found in directory %s", dirPath)
	}

	return todoList, nil
}

func loadICSFile(filePath string) (*ics.Calendar, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ics.ParseCalendar(bytes.NewReader(data))
}

func convertVTodoToTodo(vtodo *ics.VTodo) *Todo {
	todo := &Todo{
		UID: vtodo.Id(),
	}

	if prop := vtodo.GetProperty(ics.ComponentPropertySummary); prop != nil {
		todo.Summary = prop.Value
	}
	if prop := vtodo.GetProperty(ics.ComponentPropertyDescription); prop != nil {
		todo.Description = prop.Value
	}
	if prop := vtodo.GetProperty(ics.ComponentPropertyStatus); prop != nil {
		todo.Status = prop.Value
	} else {
		todo.Status = "NEEDS-ACTION" // Default status if not set
	}
	if created := vtodo.GetProperty(ics.ComponentPropertyCreated); created != nil {
		todo.Created = parseDateTime(created.Value)
	}
	if lastMod := vtodo.GetProperty(ics.ComponentPropertyLastModified); lastMod != nil {
		todo.LastMod = parseDateTime(lastMod.Value)
	}
	if due := vtodo.GetProperty(ics.ComponentPropertyDue); due != nil {
		todo.DueDate = parseDateTime(due.Value)
	}
	if priority := vtodo.GetProperty(ics.ComponentPropertyPriority); priority != nil {
		todo.Priority, _ = strconv.Atoi(priority.Value)
	}
	if categories := vtodo.GetProperty(ics.ComponentPropertyCategories); categories != nil {
		todo.Categories = strings.Split(categories.Value, ",")
	}
	if start := vtodo.GetProperty(ics.ComponentPropertyDtStart); start != nil {
		todo.StartDate = parseDateTime(start.Value)
	}

	return todo
}

func parseDateTime(value string) time.Time {
	var t time.Time
	var err error

	if strings.HasSuffix(value, "Z") {
		// UTC time
		t, err = time.Parse("20060102T150405Z", value)
		if err == nil {
			return t.Local() // Convert UTC to local time
		}
	} else if strings.Contains(value, "TZID=") {
		// TZID format
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 {
			tzParts := strings.SplitN(parts[0], "=", 2)
			if len(tzParts) == 2 {
				tzName := tzParts[1]
				dateTimeStr := parts[1]
				loc, err := time.LoadLocation(tzName)
				if err == nil {
					t, err = time.ParseInLocation("20060102T150405", dateTimeStr, loc)
					if err == nil {
						return t.Local() // Convert to local time
					}
				}
			}
		}
	} else {
		// Handle other formats
		switch {
		case len(value) == 8: // YYYYMMDD format
			t, err = time.ParseInLocation("20060102", value, time.Local)
		case len(value) == 15: // YYYYMMDDTHHMMSS format
			t, err = time.ParseInLocation("20060102T150405", value, time.Local)
		}
	}

	if err != nil {
		log.Printf("Error parsing date-time: %v", err)
		return time.Time{} // Return zero time if parsing fails
	}

	return t
}

func saveTodos(todoList *TodoList, dirPath string) error {
	for _, todo := range todoList.Todos {
		if !todo.Modified {
			continue // Skip unmodified todos
		}

		fileName := todo.UID + ".ics"
		filePath := filepath.Join(dirPath, fileName)

		// Read existing calendar if file exists
		var cal *ics.Calendar
		var err error
		if _, err := os.Stat(filePath); err == nil {
			cal, err = loadICSFile(filePath)
			if err != nil {
				return fmt.Errorf("error loading existing file %s: %v", filePath, err)
			}
		} else {
			cal = ics.NewCalendar()
		}

		// Find existing VTODO or create new one
		var vtodo *ics.VTodo
		for _, component := range cal.Components {
			if t, ok := component.(*ics.VTodo); ok && t.Id() == todo.UID {
				vtodo = t
				break
			}
		}
		if vtodo == nil {
			vtodo = cal.AddTodo(todo.UID)
		}

		// Update only the fields we manage
		setPropertyIfNotEmpty(vtodo, ics.ComponentPropertySummary, todo.Summary)
		setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyDescription, todo.Description)
		setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyStatus, todo.Status)
		setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyLastModified, todo.LastMod.UTC().Format("20060102T150405Z"))

		// Convert DTSTART to UTC and save
		if !todo.StartDate.IsZero() {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyDtStart, todo.StartDate.UTC().Format("20060102T150405Z"))
		} else {
			removeProperty(vtodo, ics.ComponentPropertyDtStart)
		}

		// Convert DUE to UTC and save
		if !todo.DueDate.IsZero() {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyDue, todo.DueDate.UTC().Format("20060102T150405Z"))
		} else {
			removeProperty(vtodo, ics.ComponentPropertyDue)
		}

		if todo.Priority > 0 {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyPriority, strconv.Itoa(todo.Priority))
		} else {
			removeProperty(vtodo, ics.ComponentPropertyPriority)
		}

		if len(todo.Categories) > 0 {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyCategories, strings.Join(todo.Categories, ","))
		} else {
			removeProperty(vtodo, ics.ComponentPropertyCategories)
		}

		// Preserve CREATED if it exists, otherwise set it
		if created := vtodo.GetProperty(ics.ComponentPropertyCreated); created == nil {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyCreated, todo.Created.UTC().Format("20060102T150405Z"))
		}

		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := cal.SerializeTo(file); err != nil {
			return fmt.Errorf("error saving todo %s: %v", todo.UID, err)
		}

		todo.Modified = false // Reset the modified flag after saving
	}

	return nil
}

func setPropertyIfNotEmpty(vtodo *ics.VTodo, property ics.ComponentProperty, value string) {
	if value != "" {
		vtodo.SetProperty(property, value)
	} else {
		removeProperty(vtodo, property)
	}
}

func removeProperty(vtodo *ics.VTodo, property ics.ComponentProperty) {
	for i, prop := range vtodo.Properties {
		if prop.IANAToken == string(property) {
			// Remove the property
			vtodo.Properties = append(vtodo.Properties[:i], vtodo.Properties[i+1:]...)
			return
		}
	}
}

func addItem(todoList *TodoList) {
	// Add new todo item
	todo := &Todo{
		UID:     generateUID(),
		Created: time.Now(),
		LastMod: time.Now(),
		Status:  "NEEDS-ACTION", // Set default status
	}

	var e error
	todo.Summary, e = display("", "Todo Title: ")
	if e != nil {
		return
	}

	if todo.Summary != "" {
		editItem(todo, todoList)
		if todo.Summary != "" && todo.Modified {
			todo.LastMod = time.Now() // Update LastMod when adding
			todoList.Todos = append(todoList.Todos, todo)
		}
	}
}

func editItem(todo *Todo, todoList *TodoList) {
	originalTodo := *todo   // Make a copy of the original todo
	isNew := todo.UID == "" // Check if this is a new item
	for edit := true; edit; {
		var displayList strings.Builder
		var tdd string
		if todo.DueDate.IsZero() {
			tdd = ""
		} else {
			tdd = todo.DueDate.Format("2006-01-02")
		}
		var comp string
		if len(todo.Summary) == 0 {
			comp = ""
		} else if todo.Status == "COMPLETED" {
			comp = "Restore item (uncomplete)\n\n"
		} else {
			comp = "Complete item\n\n"
		}
		fmt.Fprintf(&displayList,
			"Save item\n%s"+
				"Title: %s\n"+
				"Priority: %d\n"+
				"Categories (comma separated): %s\n"+
				"Due date yyyy-mm-dd: %s\n"+
				"Start date yyyy-mm-dd: %s\n"+
				"Start time hh:mm: %s\n"+
				"Description: %s\n\n"+
				"Delete item",
			comp, todo.Summary, todo.Priority, strings.Join(todo.Categories, ","),
			tdd, formatDate(todo.StartDate), formatTime(todo.StartDate), todo.Description,
		)
		out, e := display(displayList.String(), todo.Summary)
		// Cancel new item if ESC is hit without saving
		if e != nil {
			if isNew {
				// Do not add the new item to the list if ESC is hit
				return
			} else {
				*todo = originalTodo // Revert changes for existing item
			}
			return
		}
		switch {
		case out == "Save item":
			todo.Modified = true      // Set the modified flag
			todo.LastMod = time.Now() // Update LastMod when saving
			edit = false
		case strings.HasPrefix(out, "Title"):
			tn, e := display(todo.Summary, "Todo Title: ")
			if e == nil {
				todo.Summary = tn
				todo.Modified = true // Set the modified flag
			}
		case strings.HasPrefix(out, "Priority"):
			p, e := display(fmt.Sprintf("%d", todo.Priority), "Priority (0-9, 0 to unset):")
			if e == nil {
				pn, err := strconv.Atoi(p)
				if err == nil && pn >= 0 && pn <= 9 {
					if pn == 0 {
						todo.Priority = 0 // Unset priority
					} else {
						todo.Priority = pn
					}
					todo.Modified = true
				} else {
					display("", "Priority must be a number between 0 and 9")
				}
			}
		case strings.HasPrefix(out, "Categories"):
			existingCats := getExistingCategories(todoList)
			catOptions := strings.Join(existingCats, "\n") + "\n<Enter new category>"
			cats, e := display(catOptions, "Select or enter new category (comma separated):")
			if e == nil {
				if cats == "<Enter new category>" {
					newCats, _ := display("", "Enter new category (comma separated):")
					todo.Categories = strings.Split(newCats, ",")
				} else {
					todo.Categories = strings.Split(cats, ",")
				}
				for i, cat := range todo.Categories {
					todo.Categories[i] = strings.TrimSpace(cat)
				}
				if len(todo.Categories) == 1 && todo.Categories[0] == "" {
					todo.Categories = []string{} // Clear categories if empty
				}
				todo.Modified = true
			}
		case strings.HasPrefix(out, "Due date"):
			d, e := display(tdd, "Due Date (yyyy-mm-dd):")
			if e == nil {
				if d == "" {
					todo.DueDate = time.Time{} // Clear the due date
				} else {
					td, err := time.ParseInLocation("2006-01-02", d, time.Local)
					if err != nil {
						display("", "Bad date format. Should be yyyy-mm-dd.")
					} else {
						todo.DueDate = td
						todo.Modified = true
					}
				}
			}
		case strings.HasPrefix(out, "Start date"):
			d, e := display(formatDate(todo.StartDate), "Start Date (yyyy-mm-dd):")
			if e == nil {
				updateStartDate(todo, d)
			}
		case strings.HasPrefix(out, "Start time"):
			t, e := display(formatTime(todo.StartDate), "Start Time (hh:mm or hhmm):")
			if e == nil {
				updateStartTime(todo, t)
			}
		case strings.HasPrefix(out, "Description"):
			desc, e := display(todo.Description, "Description:")
			if e == nil {
				todo.Description = desc
				todo.Modified = true // Set the modified flag
			}
		case strings.HasPrefix(out, "Complete item"):
			todo.Status = "COMPLETED"
			todo.LastMod = time.Now()
			todo.Modified = true // Set the modified flag
		case strings.HasPrefix(out, "Restore item"):
			todo.Status = "NEEDS-ACTION"
			todo.LastMod = time.Now()
			todo.Modified = true // Set the modified flag
		case strings.HasPrefix(out, "Delete item"):
			confirm, _ := display("", fmt.Sprintf("Delete item: %s. y/N?", todo.Summary))
			if strings.ToLower(confirm) == "y" {
				if deleteTodo(todo, todoList) {
					edit = false
					return
				}
			}
		}
	}
}

func updateStartDate(todo *Todo, dateStr string) {
	if dateStr == "" {
		todo.StartDate = time.Time{}
	} else {
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err == nil {
			if !todo.StartDate.IsZero() {
				todo.StartDate = time.Date(date.Year(), date.Month(), date.Day(),
					todo.StartDate.Hour(), todo.StartDate.Minute(), 0, 0, time.Local)
			} else {
				todo.StartDate = date
			}
			todo.Modified = true
		}
	}
}

func updateStartTime(todo *Todo, timeStr string) {
	if timeStr == "" {
		return
	}
	var hour, min int
	var err error
	if strings.Contains(timeStr, ":") {
		_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
	} else {
		_, err = fmt.Sscanf(timeStr, "%02d%02d", &hour, &min)
	}
	if err == nil && hour >= 0 && hour < 24 && min >= 0 && min < 60 {
		if todo.StartDate.IsZero() {
			todo.StartDate = time.Now().Local()
		}
		todo.StartDate = time.Date(todo.StartDate.Year(), todo.StartDate.Month(), todo.StartDate.Day(),
			hour, min, 0, 0, time.Local)
		todo.Modified = true
	}
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("15:04")
}

func deleteTodo(todo *Todo, todoList *TodoList) bool {
	// Remove the todo from the todoList
	for i, t := range todoList.Todos {
		if t.UID == todo.UID {
			todoList.Todos = append(todoList.Todos[:i], todoList.Todos[i+1:]...)
			break
		}
	}

	// Delete the corresponding .ics file
	filePath := filepath.Join(*todoPtr, todo.UID+".ics")
	err := os.Remove(filePath)
	if err != nil {
		log.Printf("Error deleting file %s: %v", filePath, err)
		return false
	}

	log.Printf("Todo item deleted: %s", todo.Summary)
	return true
}

func viewCompletedItems(todoList *TodoList) {
	for {
		displayList, m := createMenu(todoList, true)
		out, _ := display(displayList.String(), "Completed Items")

		if out == "Delete All Completed" {
			confirm, _ := display("", "Delete ALL Completed Items? (y/N)")
			if strings.ToLower(confirm) == "y" {
				var completedTodos []*Todo
				var remainingTodos []*Todo

				// First, separate completed and non-completed todos
				for _, todo := range todoList.Todos {
					if todo.Status == "COMPLETED" {
						completedTodos = append(completedTodos, todo)
					} else {
						remainingTodos = append(remainingTodos, todo)
					}
				}

				// Now delete all completed todos
				for _, todo := range completedTodos {
					if deleteTodo(todo, todoList) {
						log.Printf("Deleted completed item: %s", todo.Summary)
					} else {
						// If deletion failed, keep the todo in the list
						remainingTodos = append(remainingTodos, todo)
					}
				}

				todoList.Todos = remainingTodos
			}
			return
		} else if out != "" {
			t := todoList.Todos[m[out]]
			editItem(t, todoList)
		} else {
			return
		}
	}
}

func display(list string, title string) (result string, e error) {
	var out, outErr bytes.Buffer
	flag.Parse()
	userOpts := strings.Split(*optsPtr, " ")

	// Default options for supported launchers
	defaultOpts := []string{"-i", "-p", title}
	switch *cmdPtr {
	case "rofi":
		defaultOpts = []string{"-i", "-dmenu", "-p", title}
	case "wofi", "fuzzel":
		defaultOpts = []string{"-i", "--dmenu", "-p", title}
	case "tofi":
		defaultOpts = []string{"-i", "--prompt-text", title}
	}

	// Combine default options with user options
	opts := append(defaultOpts, userOpts...)

	// Remove empty strings from opts
	var finalOpts []string
	for _, opt := range opts {
		if opt != "" {
			finalOpts = append(finalOpts, opt)
		}
	}

	cmd := exec.Command(*cmdPtr, finalOpts...)
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	cmd.Stdin = strings.NewReader(list)
	err := cmd.Run()
	if err != nil {
		if outErr.String() != "" {
			log.Fatal(outErr.String())
		} else {
			// Skip this error when hitting Esc to go back to previous menu
			if err.Error() == "exit status 1" {
				return "", errors.New("escape")
			}
			log.Fatal(err.Error())
		}
	}
	result = strings.TrimRight(out.String(), "\n")
	return
}

func createMenu(todoList *TodoList, showCompleted bool) (*strings.Builder, map[string]int) {
	displayList := &strings.Builder{}
	if !showCompleted {
		displayList.WriteString("Add Item\n")
		displayList.WriteString("View Completed Items\n")
	} else {
		displayList.WriteString("Delete All Completed\n")
	}

	// Updated sorting logic
	sort.Slice(todoList.Todos, func(i, j int) bool {
		a, b := todoList.Todos[i], todoList.Todos[j]

		// 1. Items with due date come first
		if !a.DueDate.IsZero() && b.DueDate.IsZero() {
			return true
		}
		if a.DueDate.IsZero() && !b.DueDate.IsZero() {
			return false
		}

		// 2. Sort by due date (ascending)
		if !a.DueDate.IsZero() && !b.DueDate.IsZero() {
			return a.DueDate.Before(b.DueDate)
		}

		// 3. Priority (lower number = higher priority, 0 means no priority)
		if a.Priority != b.Priority {
			if a.Priority == 0 {
				return false
			}
			if b.Priority == 0 {
				return true
			}
			return a.Priority < b.Priority
		}

		// 4. Created date (descending)
		return a.Created.After(b.Created)
	})

	m := make(map[string]int)
	now := time.Now()
	for i, todo := range todoList.Todos {
		if (todo.Status == "COMPLETED") != showCompleted {
			continue
		}
		if *thresholdPtr && !showCompleted {
			if !todo.StartDate.IsZero() {
				nowInStartTZ := now.In(todo.StartDate.Location())
				if todo.StartDate.After(nowInStartTZ) {
					continue // Skip items with future start dates when threshold option is set
				}
			}
		}

		// Format: "(priority) created-date summary @category due:due date"
		var displayStr strings.Builder

		// Priority
		if todo.Priority > 0 {
			fmt.Fprintf(&displayStr, "(%d) ", todo.Priority)
		} else {
			displayStr.WriteString("    ")
		}

		// Created date (only if not hidden)
		if !*hideCreatedDatePtr {
			fmt.Fprintf(&displayStr, "%s ", todo.Created.Format("2006-01-02"))
		}

		// Summary
		displayStr.WriteString(todo.Summary)

		// Category
		if len(todo.Categories) > 0 {
			for _, category := range todo.Categories {
				fmt.Fprintf(&displayStr, " @%s", category)
			}
		}

		// Due date (convert to local time for display)
		if !todo.DueDate.IsZero() {
			localDueDate := todo.DueDate.In(time.Local)
			fmt.Fprintf(&displayStr, " due:%s", localDueDate.Format("2006-01-02"))
		}

		displayList.WriteString(displayStr.String() + "\n")
		m[displayStr.String()] = i
	}

	return displayList, m
}

func generateUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func getExistingCategories(todoList *TodoList) []string {
	catMap := make(map[string]bool)
	for _, todo := range todoList.Todos {
		for _, cat := range todo.Categories {
			catMap[cat] = true
		}
	}
	var cats []string
	for cat := range catMap {
		cats = append(cats, cat)
	}
	sort.Strings(cats)
	return cats
}
