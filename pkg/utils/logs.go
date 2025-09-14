// Package utils provides some simple 'prettier' logs for a basic go usage
// DEPRECATED: This logging module is not used in the codebase. 
// The application uses logrus for all logging functionality.
// This file is kept for reference only and may be removed in future versions.
package utils

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
)

var colorGreen = color.New(color.FgGreen).Add(color.Bold).SprintFunc()
var colorRed = color.New(color.FgRed).Add(color.Bold).SprintFunc()
var colorYellow = color.New(color.FgYellow).Add(color.Bold).SprintFunc()
var colorBlue = color.New(color.FgBlue).Add(color.Bold).SprintFunc()
var colorCyan = color.New(color.FgCyan).SprintFunc()
var colorMagenta = color.New(color.FgMagenta).Add(color.Bold).SprintFunc()

// Route function gives us the current route used, with the number of current
// goroutines and the IP of the caller
func Route(r *http.Request) {
	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`

	if r.Header != nil && r.Header[`X-Forwarded-For`] != nil && r.Header[`X-Forwarded-For`][0] != `` {
		IP := r.Header[`X-Forwarded-For`][0]
		log.Printf("%s %s %s", colorMagenta(str0), colorBlue(`[`+IP+`]`), r.RequestURI)
	} else {
		IP := strings.Split(r.RemoteAddr, ":")[0]
		log.Printf("%s %s %s", colorMagenta(str0), colorBlue(`[`+IP+`]`), r.RequestURI)
	}
}

// Webook function is fired when using a webhook (for Facebook for instance) and
// gives us the name of this webhook, the field, item and verb used
func Webhook(name, field, item, verb string) {
	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[Webhook - ` + name + `]`
	str2 := verb + `/` + item + `/` + field

	log.Printf("%s %s %s", colorMagenta(str0), colorBlue(str1), colorBlue(str2))
}

// ErrorCrash function is fired upon a crash and logs us where the crash occurs
func ErrorCrash(err interface{}) {
	pc, _, line, _ := runtime.Caller(3)

	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[ERROR]`
	str2 := `(` + runtime.FuncForPC(pc).Name() + `:` + strconv.Itoa(line) + `)`
	t := time.Now().Format("2006/01/02 15:04:05")

	spew.Printf("%s %s %s %s %s\n", t, colorMagenta(str0), colorRed(str1), colorCyan(str2), colorRed(err))
}

// ErrorCrash function logs an error
func Error(err ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)

	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[ERROR]`
	str2 := `(` + runtime.FuncForPC(pc).Name() + `:` + strconv.Itoa(line) + `)`
	t := time.Now().Format("2006/01/02 15:04:05")

	if len(err) == 1 {
		spew.Config.Indent = "    "
		spew.Printf("%s %s %s %s %s\n", t, colorMagenta(str0), colorRed(str1), colorCyan(str2), colorRed(err[0]))
	} else {
		spew.Config.Indent = "    "
		fmt.Printf("%s", colorRed("----------------------------------\n"))
		spew.Printf("%s %s %s %s\n", t, colorMagenta(str0), colorRed(str1), colorCyan(str2))
		for _, each := range err {
			spew.Dump(each)
		}
		fmt.Printf("%s", colorRed("----------------------------------\n"))
	}
}

// Success function logs a success message
func Success(success interface{}) {
	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[SUCCESS]`
	t := time.Now().Format("2006/01/02 15:04:05")

	spew.Printf("%s %s %s %s\n", t, colorMagenta(str0), colorGreen(str1), colorCyan(success))
}

// Warning function logs a warning message
func Warning(warning interface{}) {
	pc, _, line, _ := runtime.Caller(1)

	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[WARNING]`
	str2 := `(` + runtime.FuncForPC(pc).Name() + `:` + strconv.Itoa(line) + `)`
	t := time.Now().Format("2006/01/02 15:04:05")

	spew.Printf("%s %s %s %s %s\n", t, colorMagenta(str0), colorYellow(str1), colorCyan(str2), colorYellow(warning))
}

// Info function logs an info message
func Info(info ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)

	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[INFO]`
	str2 := `(` + runtime.FuncForPC(pc).Name() + `:` + strconv.Itoa(line) + `)`
	t := time.Now().Format("2006/01/02 15:04:05")

	spew.Printf("%s %s %s %s %s\n", t, colorMagenta(str0), colorBlue(str1), colorCyan(str2), colorBlue(info))
}

// Debug function logs a debug message
func Debug(debug string) {
	pc, _, line, _ := runtime.Caller(1)

	str0 := `[` + strconv.Itoa(runtime.NumGoroutine()) + `]`
	str1 := `[DEBUG]`
	str2 := `(` + runtime.FuncForPC(pc).Name() + `:` + strconv.Itoa(line) + `)`
	t := time.Now().Format("2006/01/02 15:04:05")

	spew.Printf("%s %s %s %s %s\n", t, colorMagenta(str0), colorBlue(str1), colorCyan(str2), colorBlue(debug))
}

// Pretty function disasemble a variable and display it's struct and values
func Pretty(variable ...interface{}) {
	spew.Config.Indent = "    "
	fmt.Printf("%s", colorYellow("----------------------------------\n"))
	for _, each := range variable {
		spew.Dump(each)
	}
	fmt.Printf("%s", colorYellow("----------------------------------\n"))
}
