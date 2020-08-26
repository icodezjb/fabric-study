package log

import (
	"fmt"
	"os"
)

var (
	root            = &logger{name: "", h: new(swapHandler), enable: true}
	StdoutHandler   = StreamHandler(os.Stdout, LogfmtFormat())
	TerminalHandler = StreamHandler(os.Stdout, TerminalFormat(true))
)

func init() {
	//root.SetHandler(TerminalHandler)
	//root.SetHandler(DiscardHandler())
	root.SetHandler(LvlFilterHandler(LvlDebug, TerminalHandler))

}

func New(name string) Logger {
	return root.New(name)
}

func NewLoggerFile(name, path string) Logger {
	l := root.New(name)
	fileHandler, err := FileHandler(path, LogfmtFormat())
	if err != nil {
		panic(err)
	}
	l.SetHandler(fileHandler)
	return l
}

func NewWithHandle(name string, h Handler) Logger {
	l := root.New(name)
	l.SetHandler(h)
	return l
}

func Root() Logger {
	return root
}

// The following functions bypass the exported logger methods (logger.Debug,
// etc.) to keep the call depth the same for all paths to logger.write so
// runtime.Caller(2) always refers to the call site in client code.

// Debug is a convenient alias for Root().Debug
func Debug(format string, arg ...interface{}) {
	if root.enable {
		root.write(LvlDebug, fmt.Sprintf(format, arg...), skipLevel)
	}
}

// Info is a convenient alias for Root().Info
func Info(format string, arg ...interface{}) {
	if root.enable {
		root.write(LvlInfo, fmt.Sprintf(format, arg...), skipLevel)
	}
}

// Warn is a convenient alias for Root().Warn
func Warn(format string, arg ...interface{}) {
	if root.enable {
		root.write(LvlWarn, fmt.Sprintf(format, arg...), skipLevel)
	}
}

// Error is a convenient alias for Root().Error
func Error(format string, arg ...interface{}) {
	if root.enable {
		root.write(LvlError, fmt.Sprintf(format, arg...), skipLevel)
	}
}

// Crit is a convenient alias for Root().Crit
func Crit(format string, arg ...interface{}) {
	if root.enable {
		root.write(LvlCrit, fmt.Sprintf(format, arg...), skipLevel)
		os.Exit(1)
	}
}
