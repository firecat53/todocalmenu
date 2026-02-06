# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Todocalmenu is a minimal dmenu/rofi launcher for viewing and managing iCalendar (RFC 5545) todo items. It supports dmenu, rofi, wofi, fuzzel, tofi, yofi, wmenu, and bemenu. Written in Go, single-file application.

## Build & Development Commands

```bash
# Run tests
go test
go test -v

# Run with test data
go run todocalmenu.go -todo ./testdata

# Build
go build

# Using Nix (primary dev environment)
nix develop      # Enter dev shell with Go, delve, gopls, etc.
nix build        # Build the binary
```

## Architecture

The application is a single Go file (`todocalmenu.go`, ~740 lines) with a straightforward structure:

**Core Flow:**
1. Parse CLI flags → Load `.ics` files from todo directory → Enter interactive menu loop → Save modified todos on exit

**Key Data Structures:**
- `Todo` struct: Internal representation with UID, Summary, Description, Categories, Status, dates, Priority, and Modified flag
- `TodoList` struct: Container for Todo items

**Main Functions:**
- `loadTodos()`/`loadICSFile()`: Read and parse `.ics` files using golang-ical library
- `display()`: Execute external dmenu/rofi command, pipe items, capture selection
- `createMenu()`: Build sorted display list (sorts by: due date presence → due date → priority → creation date)
- `editItem()`: Interactive property editor via menu
- `saveTodos()`: Write only Modified=true todos back to disk

**Design Patterns:**
- Menu-driven infinite loop until user exits
- External process integration via `exec.Command` with stdin/stdout pipes
- Lazy save: only writes files marked as modified
- All datetimes converted to local time for display, stored as UTC

## Dependencies

- `github.com/arran4/golang-ical`: iCalendar RFC 5545 parsing

## Test Data

Sample `.ics` files in `testdata/` directory used by tests.
