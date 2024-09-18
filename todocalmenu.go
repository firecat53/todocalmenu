package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
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

var createdDatePtr = flag.Bool("no-created-date", false, "Set CreatedDate when adding new task")
var optsPtr = flag.String("opts", "", "Additional Rofi/Dmenu options")
var thresholdPtr = flag.Bool("threshold", false, "Hide items before their threshold date")
var todoPtr = flag.String("todo", "todo", "Path to todo directory")
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
}

type TodoList struct {
	Todos []*Todo
}

func loadTodos(dirPath string) (*TodoList, error) {
	todoList := &TodoList{}
	files, err := ioutil.ReadDir(dirPath)
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
	data, err := ioutil.ReadFile(filePath)
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
	}

	if created := vtodo.GetProperty(ics.ComponentPropertyCreated); created != nil {
		todo.Created, _ = time.Parse("20060102T150405Z", created.Value)
	}

	if lastMod := vtodo.GetProperty(ics.ComponentPropertyLastModified); lastMod != nil {
		todo.LastMod, _ = time.Parse("20060102T150405Z", lastMod.Value)
	}

	if due := vtodo.GetProperty(ics.ComponentPropertyDue); due != nil {
		todo.DueDate, _ = time.Parse("20060102T150405Z", due.Value)
	}

	if priority := vtodo.GetProperty(ics.ComponentPropertyPriority); priority != nil {
		todo.Priority, _ = strconv.Atoi(priority.Value)
	}

	if categories := vtodo.GetProperty(ics.ComponentPropertyCategories); categories != nil {
		todo.Categories = strings.Split(categories.Value, ",")
	}

	return todo
}

func saveTodos(todoList *TodoList, dirPath string) error {
	// Keep track of existing files
	existingFiles, err := filepath.Glob(filepath.Join(dirPath, "*.ics"))
	if err != nil {
		return fmt.Errorf("error listing existing .ics files: %v", err)
	}
	existingMap := make(map[string]bool)
	for _, file := range existingFiles {
		existingMap[filepath.Base(file)] = true
	}

	for _, todo := range todoList.Todos {
		fileName := todo.UID + ".ics"
		filePath := filepath.Join(dirPath, fileName)

		if todo.LastMod.IsZero() {
			// This todo hasn't been modified, skip it
			delete(existingMap, fileName)
			continue
		}

		cal := ics.NewCalendar()
		vtodo := cal.AddTodo(todo.UID)

		vtodo.SetProperty(ics.ComponentPropertySummary, todo.Summary)
		vtodo.SetProperty(ics.ComponentPropertyDescription, todo.Description)
		vtodo.SetProperty(ics.ComponentPropertyStatus, todo.Status)
		vtodo.SetProperty(ics.ComponentPropertyCreated, todo.Created.Format("20060102T150405Z"))
		vtodo.SetProperty(ics.ComponentPropertyLastModified, todo.LastMod.Format("20060102T150405Z"))

		if !todo.DueDate.IsZero() {
			vtodo.SetProperty(ics.ComponentPropertyDue, todo.DueDate.Format("20060102T150405Z"))
		}

		if todo.Priority != 0 {
			vtodo.SetProperty(ics.ComponentPropertyPriority, strconv.Itoa(todo.Priority))
		}

		if len(todo.Categories) > 0 {
			vtodo.SetProperty(ics.ComponentPropertyCategories, strings.Join(todo.Categories, ","))
		}

		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := cal.SerializeTo(file); err != nil {
			return fmt.Errorf("error saving todo %s: %v", todo.UID, err)
		}

		delete(existingMap, fileName)
	}

	// Delete files for todos that no longer exist
	for file := range existingMap {
		if err := os.Remove(filepath.Join(dirPath, file)); err != nil {
			log.Printf("Error deleting file %s: %v", file, err)
		}
	}

	return nil
}

func main() {
	flag.Parse()
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

func addItem(todoList *TodoList) {
	// Add new todo item
	todo := &Todo{
		UID:     generateUID(),
		Created: time.Now(),
		LastMod: time.Now(),
	}

	var e error
	todo.Summary, e = display("", "Todo Title: ")
	if e != nil {
		return
	}

	if todo.Summary != "" {
		if *createdDatePtr {
			// Zero Created Date if -no-created-date is set
			todo.Created = time.Time{}
		}
		editItem(todo, todoList)
		if todo.Summary != "" {
			todo.LastMod = time.Now() // Update LastMod when adding
			todoList.Todos = append(todoList.Todos, todo)
		}
	}
}

func editItem(todo *Todo, todoList *TodoList) {
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
				"Description: %s\n\n"+
				"Delete item",
			comp, todo.Summary, todo.Priority, strings.Join(todo.Categories, ","),
			tdd, todo.Description,
		)
		out, e := display(displayList.String(), todo.Summary)
		// Cancel new item if ESC is hit without saving
		if e != nil {
			if todo.UID == "" {
				todo.Summary = ""
			}
			return
		}
		switch {
		case out == "Save item":
			edit = false
		case strings.HasPrefix(out, "Title"):
			tn, e := display(todo.Summary, "Todo Title: ")
			if e == nil {
				todo.Summary = tn
			}
		case strings.HasPrefix(out, "Priority"):
			p, e := display(fmt.Sprintf("%d", todo.Priority), "Priority (1-9):")
			if e == nil {
				pn, err := strconv.Atoi(p)
				if err == nil && pn >= 1 && pn <= 9 {
					todo.Priority = pn
				} else {
					display("", "Priority must be a number between 1 and 9")
				}
			}
		case strings.HasPrefix(out, "Categories"):
			cats, e := display(strings.Join(todo.Categories, ","), "Categories (comma separated):")
			if e == nil {
				todo.Categories = strings.Split(cats, ",")
			}
		case strings.HasPrefix(out, "Due date"):
			d, e := display(tdd, "Due Date (yyyy-mm-dd):")
			td, err := time.Parse("2006-01-02", d)
			if e == nil {
				if err != nil && d != "" {
					display("", "Bad date format. Should be yyyy-mm-dd.")
				} else {
					todo.DueDate = td
				}
			}
		case strings.HasPrefix(out, "Description"):
			desc, e := display(todo.Description, "Description:")
			if e == nil {
				todo.Description = desc
			}
		case strings.HasPrefix(out, "Complete item"):
			todo.Status = "COMPLETED"
			todo.LastMod = time.Now()
		case strings.HasPrefix(out, "Restore item"):
			todo.Status = "NEEDS-ACTION"
			todo.LastMod = time.Now()
		case strings.HasPrefix(out, "Delete item"):
			if deleteTodo(todo, todoList) {
				edit = false
				return
			}
		}
		todo.LastMod = time.Now()
	}
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
				var remainingTodos []*Todo
				for _, todo := range todoList.Todos {
					if todo.Status != "COMPLETED" {
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
	// Displays list in dmenu, returns selection
	var out, outErr bytes.Buffer
	flag.Parse()
	opts := strings.Split(*optsPtr, " ")
	o := []string{"-i", "-p", title}
	if *cmdPtr == "rofi" {
		o = []string{"-i", "-dmenu", "-p", title}
	}
	// Remove empty "" from dmenu args that would cause a dmenu error
	if opts[0] != "" {
		opts = append(o, opts...)
	} else {
		opts = o
	}
	cmd := exec.Command(*cmdPtr, opts...)
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
		log.Fatal(err.Error())
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

		// 1. Priority (lower number = higher priority, 0 means no priority)
		if (a.Priority != 0 || b.Priority != 0) && a.Priority != b.Priority {
			if a.Priority == 0 {
				return false
			}
			if b.Priority == 0 {
				return true
			}
			return a.Priority < b.Priority
		}

		// 2. Due date
		if !a.DueDate.Equal(b.DueDate) {
			return a.DueDate.Before(b.DueDate)
		}

		// 3. Category (first category alphabetically)
		if len(a.Categories) > 0 && len(b.Categories) > 0 && a.Categories[0] != b.Categories[0] {
			return a.Categories[0] < b.Categories[0]
		}

		// 4. Alphabetically by summary
		return a.Summary < b.Summary
	})

	m := make(map[string]int)
	for i, todo := range todoList.Todos {
		if (todo.Status == "COMPLETED") != showCompleted {
			continue
		}
		if *thresholdPtr && !showCompleted {
			// Implement threshold date checking if needed
		}

		// Format: "(priority) yyyy-mm-dd summary @category"
		var displayStr strings.Builder

		// Priority
		if todo.Priority > 0 {
			fmt.Fprintf(&displayStr, "(%d) ", todo.Priority)
		} else {
			displayStr.WriteString("    ")
		}

		// Due date
		if !todo.DueDate.IsZero() {
			fmt.Fprintf(&displayStr, "%s ", todo.DueDate.Format("2006-01-02"))
		} else {
			displayStr.WriteString("          ")
		}

		// Summary
		displayStr.WriteString(todo.Summary)

		// Category
		if len(todo.Categories) > 0 {
			fmt.Fprintf(&displayStr, " @%s", todo.Categories[0])
		}

		displayList.WriteString(displayStr.String() + "\n")
		m[displayStr.String()] = i
	}

	return displayList, m
}

func generateUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
