## Todocalmenu

A minimal dmenu/rofi launcher (also bemenu, wofi, fuzzel, tofi, yofi, and wmenu) app to view and manage a directory of
[icalendar](https://icalendar.org/iCalendar-RFC-5545/3-6-2-to-do-component.html)
todo's. 

Can be used, for example, with a combination of [Nextcloud
Tasks](https://github.com/nextcloud/tasks),
[vdirsyncer](https://github.com/pimutils/vdirsyncer), and
[Tasks](https://tasks.org/) for Android.


### Installation

* `go get github.com/firecat53/todocalmenu.go` OR [download binary](https://github.com/firecat53/todocalmenu/releases)

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

### Testing

* `go test`