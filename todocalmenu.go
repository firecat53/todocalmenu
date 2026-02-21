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
	"github.com/teambition/rrule-go"
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
	RRULE       string // Recurrence rule (RFC 5545)
	Modified    bool   // New field to track changes in the current session
	ListName    string // Subdirectory basename; empty in single-list mode
	ListDir     string // Full path to the directory containing this todo's .ics file
}

var listNames []string            // Empty = single-list mode; populated = multi-list mode
var listDisplayNames map[string]string // Dir name -> display name (from displayname file)

type TodoList struct {
	Todos []*Todo
}

func main() {
	flag.Parse()

	// Ensure the todo directory exists
	if err := os.MkdirAll(*todoPtr, 0755); err != nil {
		log.Fatalf("Failed to create todo directory: %v", err)
	}

	// Discover lists (multi-list vs single-list mode)
	var err error
	listNames, err = discoverLists(*todoPtr)
	if err != nil {
		log.Fatal(err.Error())
	}

	todoList, err := loadAllTodos(*todoPtr)
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
			editItem(t, todoList, false)
		default:
			edit = false
		}
	}
	if err := saveTodos(todoList); err != nil {
		log.Fatal(err.Error())
	}
}

// dirContainsVTodos checks if a directory contains .ics files with VTODO components.
func dirContainsVTodos(dirPath string) bool {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return false
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".ics" {
			continue
		}
		cal, err := loadICSFile(filepath.Join(dirPath, file.Name()))
		if err != nil {
			continue
		}
		for _, component := range cal.Components {
			if _, ok := component.(*ics.VTodo); ok {
				return true
			}
		}
	}
	return false
}

// readDisplayName reads a "displayname" file from dirPath and returns its
// contents as the list display name. Returns empty string if file doesn't exist.
func readDisplayName(dirPath string) string {
	data, err := os.ReadFile(filepath.Join(dirPath, "displayname"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// listDisplayName returns the display name for a directory name,
// falling back to the directory name itself.
func listDisplayName(dirName string) string {
	if dn, ok := listDisplayNames[dirName]; ok {
		return dn
	}
	return dirName
}

// discoverLists scans rootDir for todo list subdirectories.
// Returns nil if rootDir contains .ics files directly (single-list mode)
// or if nothing is found (empty directory for new users).
// Also populates listDisplayNames from displayname files in each subdirectory.
func discoverLists(rootDir string) ([]string, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %v", err)
	}

	// If any .ics files exist directly, use single-list mode
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".ics" {
			return nil, nil
		}
	}

	// Scan subdirectories for VTODO-containing .ics files
	var names []string
	listDisplayNames = make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subDir := filepath.Join(rootDir, entry.Name())
		if dirContainsVTodos(subDir) {
			names = append(names, entry.Name())
			if dn := readDisplayName(subDir); dn != "" {
				listDisplayNames[entry.Name()] = dn
			}
		}
	}

	if len(names) == 0 {
		return nil, nil
	}

	sort.Strings(names)
	return names, nil
}

func loadTodos(dirPath string, listName string) (*TodoList, error) {
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
				t := convertVTodoToTodo(todo)
				t.ListName = listName
				t.ListDir = dirPath
				todoList.Todos = append(todoList.Todos, t)
			}
		}
	}

	if len(todoList.Todos) == 0 {
		log.Printf("Warning: No todos found in directory %s", dirPath)
	}

	return todoList, nil
}

func loadAllTodos(rootDir string) (*TodoList, error) {
	if len(listNames) == 0 {
		// Single-list mode
		return loadTodos(rootDir, "")
	}

	// Multi-list mode
	combined := &TodoList{}
	for _, name := range listNames {
		subDir := filepath.Join(rootDir, name)
		tl, err := loadTodos(subDir, listDisplayName(name))
		if err != nil {
			log.Printf("Error loading list %s: %v", name, err)
			continue
		}
		combined.Todos = append(combined.Todos, tl.Todos...)
	}

	if len(combined.Todos) == 0 {
		log.Printf("Warning: No todos found in any list under %s", rootDir)
	}

	return combined, nil
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
	if rruleProp := vtodo.GetProperty(ics.ComponentPropertyRrule); rruleProp != nil {
		todo.RRULE = rruleProp.Value
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

func saveTodos(todoList *TodoList) error {
	for _, todo := range todoList.Todos {
		if !todo.Modified {
			continue // Skip unmodified todos
		}

		fileName := todo.UID + ".ics"
		filePath := filepath.Join(todo.ListDir, fileName)

		// Read existing calendar if file exists
		var cal *ics.Calendar
		if _, err := os.Stat(filePath); err == nil {
			var loadErr error
			cal, loadErr = loadICSFile(filePath)
			if loadErr != nil {
				return fmt.Errorf("error loading existing file %s: %v", filePath, loadErr)
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

		// Convert DTSTART to UTC and save (date-only if no time set)
		if !todo.StartDate.IsZero() {
			if hasTimeComponent(todo.StartDate) {
				setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyDtStart, todo.StartDate.UTC().Format("20060102T150405Z"))
			} else {
				vtodo.SetProperty(ics.ComponentPropertyDtStart, todo.StartDate.Format("20060102"), ics.WithValue("DATE"))
			}
		} else {
			removeProperty(vtodo, ics.ComponentPropertyDtStart)
		}

		// Convert DUE to UTC and save (date-only if no time set)
		if !todo.DueDate.IsZero() {
			if hasTimeComponent(todo.DueDate) {
				setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyDue, todo.DueDate.UTC().Format("20060102T150405Z"))
			} else {
				vtodo.SetProperty(ics.ComponentPropertyDue, todo.DueDate.Format("20060102"), ics.WithValue("DATE"))
			}
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

		// Handle RRULE
		if todo.RRULE != "" {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyRrule, todo.RRULE)
		} else {
			removeProperty(vtodo, ics.ComponentPropertyRrule)
		}

		// Preserve CREATED if it exists, otherwise set it
		if created := vtodo.GetProperty(ics.ComponentPropertyCreated); created == nil {
			setPropertyIfNotEmpty(vtodo, ics.ComponentPropertyCreated, todo.Created.UTC().Format("20060102T150405Z"))
		}

		var buf bytes.Buffer
		if err := cal.SerializeTo(&buf); err != nil {
			return fmt.Errorf("error saving todo %s: %v", todo.UID, err)
		}
		// Fix golang-ical escaping commas in CATEGORIES values
		output := fixCategoriesEscaping(buf.String())
		if err := os.WriteFile(filePath, []byte(output), 0644); err != nil {
			return err
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

func fixCategoriesEscaping(icsData string) string {
	lines := strings.SplitAfter(icsData, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "CATEGORIES:") {
			lines[i] = strings.ReplaceAll(line, `\,`, ",")
		}
	}
	return strings.Join(lines, "")
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

	if todo.Summary == "" {
		return
	}

	// Set ListDir based on mode
	if len(listNames) > 0 {
		// Multi-list mode: prompt for list selection
		// Build display name list and reverse mapping
		var displayOptions []string
		displayToDirName := make(map[string]string)
		for _, dirName := range listNames {
			dn := listDisplayName(dirName)
			displayOptions = append(displayOptions, dn)
			displayToDirName[dn] = dirName
		}
		options := strings.Join(displayOptions, "\n") + "\nNew List"
		selection, e := display(options, "Select List:")
		if e != nil {
			return
		}
		if selection == "New List" {
			newName, e := display("", "List Name:")
			if e != nil || newName == "" {
				return
			}
			newDir := filepath.Join(*todoPtr, newName)
			if err := os.MkdirAll(newDir, 0755); err != nil {
				log.Printf("Error creating list directory: %v", err)
				return
			}
			listNames = append(listNames, newName)
			sort.Strings(listNames)
			todo.ListName = newName
			todo.ListDir = newDir
		} else {
			dirName := displayToDirName[selection]
			todo.ListName = selection
			todo.ListDir = filepath.Join(*todoPtr, dirName)
		}
	} else {
		// Single-list mode
		todo.ListDir = *todoPtr
	}

	editItem(todo, todoList, true)
	if todo.Summary != "" && todo.Modified {
		todo.LastMod = time.Now() // Update LastMod when adding
		todoList.Todos = append(todoList.Todos, todo)
	}
}

func generateDateOptions(isDueDate bool) string {
	now := time.Now()
	today := now.Format("2006-01-02")
	plus1 := now.AddDate(0, 0, 1).Format("2006-01-02")
	plus7 := now.AddDate(0, 0, 7).Format("2006-01-02")
	plus14 := now.AddDate(0, 0, 14).Format("2006-01-02")

	var options []string
	if !isDueDate {
		options = append(options, "Same as Due Date")
	}
	options = append(options, fmt.Sprintf("Today (%s)", today))
	options = append(options, fmt.Sprintf("+1 day (%s)", plus1))
	options = append(options, fmt.Sprintf("+7 days (%s)", plus7))
	options = append(options, fmt.Sprintf("+14 days (%s)", plus14))

	if isDueDate {
		plus30 := now.AddDate(0, 0, 30).Format("2006-01-02")
		options = append(options, fmt.Sprintf("+30 days (%s)", plus30))

		// End of week (Friday)
		daysUntilFriday := (5 - int(now.Weekday()) + 7) % 7
		if daysUntilFriday == 0 {
			daysUntilFriday = 7 // If today is Friday, use next Friday
		}
		endOfWeek := now.AddDate(0, 0, daysUntilFriday).Format("2006-01-02")
		options = append(options, fmt.Sprintf("End of week (%s)", endOfWeek))

		// End of month
		endOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.Local).Format("2006-01-02")
		options = append(options, fmt.Sprintf("End of month (%s)", endOfMonth))
	}

	options = append(options, "Custom...")
	options = append(options, "Clear")
	return strings.Join(options, "\n")
}

func parseDateSelection(selection string) (date time.Time, isClear bool, isCustom bool) {
	if selection == "Clear" {
		return time.Time{}, true, false
	}
	if selection == "Custom..." {
		return time.Time{}, false, true
	}

	// Extract date from format "Label (YYYY-MM-DD)"
	start := strings.LastIndex(selection, "(")
	end := strings.LastIndex(selection, ")")
	if start != -1 && end != -1 && end > start {
		dateStr := selection[start+1 : end]
		parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err == nil {
			return parsed, false, false
		}
	}
	return time.Time{}, false, true // Fallback to custom if parsing fails
}

func isDateInPast(date time.Time) bool {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	return dateOnly.Before(today)
}

func isToday(date time.Time) bool {
	now := time.Now()
	return date.Year() == now.Year() && date.Month() == now.Month() && date.Day() == now.Day()
}

func hasTimeComponent(t time.Time) bool {
	return t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0
}

// calculateNextOccurrence calculates the next occurrence based on RRULE
// Returns the next occurrence time and whether it was successfully calculated
func calculateNextOccurrence(todo *Todo) (time.Time, bool) {
	if todo.RRULE == "" {
		return time.Time{}, false
	}

	// Build the RRULE string with DTSTART
	var dtstart time.Time
	if !todo.DueDate.IsZero() {
		dtstart = todo.DueDate
	} else if !todo.StartDate.IsZero() {
		dtstart = todo.StartDate
	} else {
		dtstart = time.Now()
	}

	// Parse the RRULE with DTSTART
	rruleStr := fmt.Sprintf("DTSTART:%s\nRRULE:%s", dtstart.UTC().Format("20060102T150405Z"), todo.RRULE)
	ruleSet, err := rrule.StrToRRuleSet(rruleStr)
	if err != nil {
		log.Printf("Error parsing RRULE: %v", err)
		return time.Time{}, false
	}

	// Get the next occurrence after now
	next := ruleSet.After(time.Now(), false)
	if next.IsZero() {
		// No more occurrences (e.g., COUNT or UNTIL limit reached)
		return time.Time{}, false
	}

	return next.Local(), true
}

func editItem(todo *Todo, todoList *TodoList, isNew bool) {
	originalTodo := *todo // Make a copy of the original todo
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
		} else if todo.RRULE != "" {
			comp = "Complete (reschedule to next)\nComplete permanently\n\n"
		} else {
			comp = "Complete item\n\n"
		}
		fmt.Fprintf(&displayList,
			"Save item\n%s"+
				"Title: %s\n"+
				"Priority: %d\n"+
				"Categories (comma separated): %s\n"+
				"Due date yyyy-mm-dd: %s\n"+
				"Due time hh:mm: %s\n"+
				"Start date yyyy-mm-dd: %s\n"+
				"Start time hh:mm: %s\n"+
				"Description: %s\n\n"+
				"Delete item",
			comp, todo.Summary, todo.Priority, strings.Join(todo.Categories, ","),
			tdd, formatTime(todo.DueDate), formatDate(todo.StartDate), formatTime(todo.StartDate), todo.Description,
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
			for {
				options := generateDateOptions(true)
				selection, e := display(options, "Due Date:")
				if e != nil {
					break
				}
				date, isClear, isCustom := parseDateSelection(selection)
				if isClear {
					todo.DueDate = time.Time{}
					todo.Modified = true
					break
				}
				if isCustom {
					d, e := display(tdd, "Due Date (yyyy-mm-dd):")
					if e != nil {
						break
					}
					if d == "" {
						todo.DueDate = time.Time{}
						todo.Modified = true
						break
					}
					td, err := time.ParseInLocation("2006-01-02", d, time.Local)
					if err != nil {
						display("", "Bad date format. Should be yyyy-mm-dd.")
						continue
					}
					date = td
				}
				if isNew && isDateInPast(date) {
					display("", "Cannot set due date in the past for new items.")
					continue
				}
				// Preserve existing time if any
				if !todo.DueDate.IsZero() {
					date = time.Date(date.Year(), date.Month(), date.Day(),
						todo.DueDate.Hour(), todo.DueDate.Minute(), 0, 0, time.Local)
				}
				todo.DueDate = date
				todo.Modified = true
				break
			}
		case strings.HasPrefix(out, "Due time"):
			for {
				t, e := display(formatTime(todo.DueDate), "Due Time (hh:mm or hhmm):")
				if e != nil {
					break
				}
				if t == "" {
					break
				}
				newTime, err := parseTimeInput(t)
				if err != nil {
					display("", "Bad time format. Should be hh:mm or hhmm.")
					continue
				}
				// Validate for new items: time must be >= now
				if isNew {
					now := time.Now()
					// Use existing date or today if not set
					baseDate := todo.DueDate
					if baseDate.IsZero() {
						baseDate = now
					}
					proposedDateTime := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
						newTime.hour, newTime.minute, 0, 0, time.Local)
					if proposedDateTime.Before(now) {
						display("", "Cannot set due time in the past for new items.")
						continue
					}
				}
				updateDueTimeFromParsed(todo, newTime)
				break
			}
		case strings.HasPrefix(out, "Start date"):
			for {
				options := generateDateOptions(false)
				selection, e := display(options, "Start Date:")
				if e != nil {
					break
				}
				// Handle "Same as Due Date" option
				if selection == "Same as Due Date" {
					todo.StartDate = todo.DueDate // Copies the due date (including time), or clears if due date is not set
					todo.Modified = true
					break
				}
				date, isClear, isCustom := parseDateSelection(selection)
				if isClear {
					todo.StartDate = time.Time{}
					todo.Modified = true
					break
				}
				if isCustom {
					d, e := display(formatDate(todo.StartDate), "Start Date (yyyy-mm-dd):")
					if e != nil {
						break
					}
					if d == "" {
						todo.StartDate = time.Time{}
						todo.Modified = true
						break
					}
					td, err := time.ParseInLocation("2006-01-02", d, time.Local)
					if err != nil {
						display("", "Bad date format. Should be yyyy-mm-dd.")
						continue
					}
					date = td
				}
				if isNew && isDateInPast(date) {
					display("", "Cannot set start date in the past for new items.")
					continue
				}
				// Preserve existing time if any
				if !todo.StartDate.IsZero() {
					date = time.Date(date.Year(), date.Month(), date.Day(),
						todo.StartDate.Hour(), todo.StartDate.Minute(), 0, 0, time.Local)
				}
				todo.StartDate = date
				todo.Modified = true
				break
			}
		case strings.HasPrefix(out, "Start time"):
			for {
				t, e := display(formatTime(todo.StartDate), "Start Time (hh:mm or hhmm):")
				if e != nil {
					break
				}
				if t == "" {
					break
				}
				newTime, err := parseTimeInput(t)
				if err != nil {
					display("", "Bad time format. Should be hh:mm or hhmm.")
					continue
				}
				// Validate for new items: time must be >= now
				if isNew {
					now := time.Now()
					// Use existing date or today if not set
					baseDate := todo.StartDate
					if baseDate.IsZero() {
						baseDate = now
					}
					proposedDateTime := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
						newTime.hour, newTime.minute, 0, 0, time.Local)
					if proposedDateTime.Before(now) {
						display("", "Cannot set start time in the past for new items.")
						continue
					}
				}
				updateStartTimeFromParsed(todo, newTime)
				break
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
		case strings.HasPrefix(out, "Complete (reschedule"):
			// Recurring task: reschedule to next occurrence
			nextDue, ok := calculateNextOccurrence(todo)
			if ok {
				// Calculate the offset between old due and start dates
				var offset time.Duration
				if !todo.DueDate.IsZero() && !todo.StartDate.IsZero() {
					offset = todo.DueDate.Sub(todo.StartDate)
				}
				// Update due date to next occurrence
				todo.DueDate = nextDue
				// Update start date if it was set (maintain the same offset)
				if !todo.StartDate.IsZero() {
					todo.StartDate = nextDue.Add(-offset)
				}
				todo.LastMod = time.Now()
				todo.Modified = true
			} else {
				// No more occurrences, treat as permanent completion
				todo.Status = "COMPLETED"
				todo.LastMod = time.Now()
				todo.Modified = true
			}
		case strings.HasPrefix(out, "Complete permanently"):
			// Permanently complete recurring task
			todo.Status = "COMPLETED"
			todo.RRULE = "" // Remove recurrence
			todo.LastMod = time.Now()
			todo.Modified = true
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

type parsedTime struct {
	hour   int
	minute int
}

func parseTimeInput(timeStr string) (parsedTime, error) {
	var hour, min int
	var err error
	if strings.Contains(timeStr, ":") {
		_, err = fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
	} else {
		_, err = fmt.Sscanf(timeStr, "%02d%02d", &hour, &min)
	}
	if err != nil || hour < 0 || hour >= 24 || min < 0 || min >= 60 {
		return parsedTime{}, errors.New("invalid time")
	}
	return parsedTime{hour: hour, minute: min}, nil
}

func updateStartTimeFromParsed(todo *Todo, pt parsedTime) {
	if todo.StartDate.IsZero() {
		todo.StartDate = time.Now().Local()
	}
	todo.StartDate = time.Date(todo.StartDate.Year(), todo.StartDate.Month(), todo.StartDate.Day(),
		pt.hour, pt.minute, 0, 0, time.Local)
	todo.Modified = true
}

func updateDueTimeFromParsed(todo *Todo, pt parsedTime) {
	if todo.DueDate.IsZero() {
		todo.DueDate = time.Now().Local()
	}
	todo.DueDate = time.Date(todo.DueDate.Year(), todo.DueDate.Month(), todo.DueDate.Day(),
		pt.hour, pt.minute, 0, 0, time.Local)
	todo.Modified = true
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func formatTime(t time.Time) string {
	if t.IsZero() || !hasTimeComponent(t) {
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
	filePath := filepath.Join(todo.ListDir, todo.UID+".ics")
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
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
			editItem(t, todoList, false)
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

		// List name (multi-list mode only)
		if todo.ListName != "" {
			fmt.Fprintf(&displayStr, " +%s", todo.ListName)
		}

		// Due date (convert to local time for display)
		if !todo.DueDate.IsZero() {
			localDueDate := todo.DueDate.In(time.Local)
			fmt.Fprintf(&displayStr, " due:%s", localDueDate.Format("2006-01-02"))
		}

		// Recurring indicator
		if todo.RRULE != "" {
			displayStr.WriteString(" [R]")
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
