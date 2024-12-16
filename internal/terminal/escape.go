package terminal

const (
	ESC                  = "\033"
	BACKSPACE            = "\b \b"
	RESET_CURSOR         = ESC + "[H"
	RESET_CURSOR_COLUMN  = ESC + "[G"
	CLEAR_REMAINING_LINE = ESC + "[K"
	CLEAR_LINE           = RESET_CURSOR_COLUMN + ESC + "[2K"
	CLEAR_SCREEN         = RESET_CURSOR + ESC + "[2J"

	BLACK         = ESC + "[0;30m"
	GRAY          = ESC + "[1;30m"
	RED           = ESC + "[0;31m"
	LIGHT_RED     = ESC + "[1;31m"
	GREEN         = ESC + "[0;32m"
	LIGHT_GREEN   = ESC + "[1;32m"
	YELLOW        = ESC + "[0;33m"
	LIGHT_YELLOW  = ESC + "[1;33m"
	BLUE          = ESC + "[0;34m"
	LIGHT_BLUE    = ESC + "[1;34m"
	MAGENTA       = ESC + "[0;35m"
	LIGHT_MAGENTA = ESC + "[1;35m"
	CYAN          = ESC + "[0;36m"
	LIGHT_CYAN    = ESC + "[1;36m"
	WHITE         = ESC + "[0;37m"
	LIGHT_WHITE   = ESC + "[1;37m"
	RESET         = ESC + "[0m"
)
