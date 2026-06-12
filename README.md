## Todocalmenu

> NOTE: Code moved to https://git.firecat53.me/firecat53/todocalmenu. Issues and
> PRs still accepted here for now. Github repo maintained as a read-only mirror.

A minimal dmenu/rofi launcher (also bemenu, wofi, fuzzel, tofi, yofi, and wmenu) app to view and manage a directory of
[icalendar](https://icalendar.org/iCalendar-RFC-5545/3-6-2-to-do-component.html)
todo's. 

Can be used, for example, with a combination of [Nextcloud
Tasks](https://github.com/nextcloud/tasks),
[vdirsyncer](https://github.com/pimutils/vdirsyncer), and
[Tasks](https://tasks.org/) for Android.


### Installation

* `go install github.com/firecat53/todocalmenu@latest` OR [download binary](https://github.com/firecat53/todocalmenu/releases)

### Usage

* Command line options:

          -cmd string
                Dmenu command to use (dmenu, rofi, wofi, etc) (default "dmenu")
          -hide-created-date
                Don't display the created date (default false)
          -opts string
                Additional Rofi/Dmenu options (default "")
          -todo string
                Path to todo directory (default "./todos")
          -threshold
                Hide items before their threshold (Start) date (default false)

* Configure the launcher using appropriate command line options and pass using
  the `-opts` flag to todocalmenu.
  *NOTE* The `-i` and `dmenu` flags are passed to all launchers by default when necessary. Supported launchers by default are dmenu, rofi, wofi, fuzzel, tofi, yofi, wmenu and bemenu. Others may work but may not have the correct flags passed by default.
  
        todocalmenu -cmd rofi -todo /home/user/todos -opts "-theme todocalmenu"
        todocalmenu -todo /home/user/todos -opts
            "-fn SourceCodePro-Regular:12 -b -l 10 -nf blue -nb black"

### Display Format

    (priority) created-date summary @category +listname due:due-date [R]

- `@category` - categories assigned to the item (one per category)
- `+listname` - list name (only shown in multi-list mode)

### Multiple Lists

If the `-todo` directory contains `.ics` files directly, todocalmenu operates in
single-list mode. If it contains subdirectories with `.ics` files (e.g. from
CalDAV sync), each subdirectory is treated as a separate list and items are
tagged with `+listname` in the display. A `displayname` file in a subdirectory
(common with CalDAV) is used as the list name if present, otherwise the
directory name is used.

### Recurring Tasks

Recurring tasks are indicated with `[R]` in the task list. When completing a
recurring task, you can choose to:

- **Complete (reschedule to next)** - Reschedules the task to the next
  occurrence based on its recurrence rule
- **Complete permanently** - Marks the task as completed and removes the
  recurrence

**TODO**: Creating and editing recurrence rules is not yet supported. Use
another calendar app (e.g., Tasks.org, Nextcloud Tasks) to set up recurring
tasks.

### Testing

* `go test`
