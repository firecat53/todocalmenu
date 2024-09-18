## Todocalmenu

A minimal dmenu/rofi launcher app to view and manage a directory of
[icalendar](https://icalendar.org/iCalendar-RFC-5545/3-6-2-to-do-component.html)
todo's. 

Can be used, for example, with a combination of [Nextcloud
Tasks](https://github.com/nextcloud/tasks),
[vdirsyncer](https://github.com/pimutils/vdirsyncer), and
[Tasks](https://tasks.org/) for Android.


### Installation

- `go get github.com/firecat53/todocalmenu.go` OR [download binary](https://github.com/firecat53/todocalmenu/releases)

### Usage

- Command line options:

          -cmd string
                Dmenu command to use (dmenu, rofi, wofi, etc) (default "dmenu")
          -no-created-date
                Don't automatically add a date when creating a new item
          -opts string
                Additional Rofi/Dmenu options (default "")
          -todo string
                Path to todo directory (default "todo.txt")
          -threshold
                Hide items before their threshold date (default false)

- Configure the launcher using appropriate command line options and pass using
  the `-opts` flag to todocalmenu.
  *NOTE* The `-i` flag is passed to both Dmenu and Rofi by default. The `-dmenu`
  flag is passed to Rofi. Examples:
  
        todocalmenu -cmd rofi -todo /home/user/todos -opts "-theme todocalmenu"
        todocalmenu -todo /home/user/todos -opts
            "-fn SourceCodePro-Regular:12 -b -l 10 -nf blue -nb black"
