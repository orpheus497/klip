// Package ui provides user interface components for klip
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	// Color functions for different message types
	Success = color.New(color.FgGreen).SprintFunc()
	Error   = color.New(color.FgRed).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()
	Bold    = color.New(color.Bold).SprintFunc()
	Dim     = color.New(color.Faint).SprintFunc()
)

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Success("✓"), message)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", Error("✗"), message)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Warning("!"), message)
}

// PrintInfo prints an informational message
func PrintInfo(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", Info("ℹ"), message)
}

// PrintHeader prints a section header
func PrintHeader(text string) {
	fmt.Println()
	fmt.Println(Bold(text))
	fmt.Println(strings.Repeat("=", len(text)))
}

// PrintSubHeader prints a subsection header
func PrintSubHeader(text string) {
	fmt.Println()
	fmt.Println(Bold(text))
	fmt.Println(strings.Repeat("-", len(text)))
}

// PrintTable prints data in a table format
func PrintTable(headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	headerRow := ""
	for i, header := range headers {
		if i > 0 {
			headerRow += " │ "
		}
		headerRow += Bold(padRight(header, widths[i]))
	}
	fmt.Println(headerRow)

	// Print separator
	separator := ""
	for i, width := range widths {
		if i > 0 {
			separator += "─┼─"
		}
		separator += strings.Repeat("─", width)
	}
	fmt.Println(separator)

	// Print rows
	for _, row := range rows {
		rowStr := ""
		for i := 0; i < len(widths); i++ {
			if i > 0 {
				rowStr += " │ "
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			rowStr += padRight(cell, widths[i])
		}
		fmt.Println(rowStr)
	}
}

// PrintKeyValue prints key-value pairs
func PrintKeyValue(key, value string) {
	fmt.Printf("%s: %s\n", Bold(key), value)
}

// PrintList prints a bulleted list
func PrintList(items []string) {
	for _, item := range items {
		fmt.Printf("  • %s\n", item)
	}
}

// PrintNumberedList prints a numbered list
func PrintNumberedList(items []string) {
	for i, item := range items {
		fmt.Printf("  %d. %s\n", i+1, item)
	}
}

// PrintSeparator prints a separator line
func PrintSeparator() {
	fmt.Println(strings.Repeat("─", 80))
}

// PrintEmptyLine prints an empty line
func PrintEmptyLine() {
	fmt.Println()
}

// padRight pads a string with spaces on the right
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeft pads a string with spaces on the left
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}

// Confirm prompts the user for confirmation (Y/n)
func Confirm(prompt string) bool {
	fmt.Printf("%s [Y/n]: ", prompt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "" || response == "y" || response == "yes"
}

// ConfirmDefaultNo prompts the user for confirmation (y/N)
func ConfirmDefaultNo(prompt string) bool {
	fmt.Printf("%s [y/N]: ", prompt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// PrintJSON prints data as formatted JSON
func PrintJSON(data interface{}) error {
	// Note: This is a simplified version; in production, use encoding/json
	fmt.Printf("%+v\n", data)
	return nil
}

// ClearLine clears the current line
func ClearLine() {
	fmt.Print("\r\033[K")
}

// PrintInline prints without a newline
func PrintInline(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
