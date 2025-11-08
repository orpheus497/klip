package ui

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/term"
)

// sanitizeInput removes control characters and ANSI escape sequences from user input
// This prevents terminal injection attacks and ensures clean input
func sanitizeInput(s string) string {
	// Remove control characters (0x00-0x1F, 0x7F) except tab and newline
	var result strings.Builder
	for _, r := range s {
		// Keep printable characters and safe whitespace (space, tab, newline)
		if unicode.IsPrint(r) || r == '\t' || r == '\n' || r == ' ' {
			result.WriteRune(r)
		}
	}

	s = result.String()

	// Remove ANSI escape sequences (e.g., \x1b[31m for colors)
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	s = ansiEscape.ReplaceAllString(s, "")

	// Remove CSI sequences (e.g., \x1b]0;title\x07)
	csiSequence := regexp.MustCompile(`\x1b\][^\x07]*\x07`)
	s = csiSequence.ReplaceAllString(s, "")

	// Remove other escape sequences
	otherEscape := regexp.MustCompile(`\x1b[^\[]*`)
	s = otherEscape.ReplaceAllString(s, "")

	return strings.TrimSpace(s)
}

// PromptString prompts for a string input
func PromptString(prompt string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Sanitize input to remove control characters and escape sequences
	input = sanitizeInput(input)

	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}

	return input, nil
}

// PromptInt prompts for an integer input
func PromptInt(prompt string, defaultValue int) (int, error) {
	defaultStr := ""
	if defaultValue > 0 {
		defaultStr = strconv.Itoa(defaultValue)
	}

	input, err := PromptString(prompt, defaultStr)
	if err != nil {
		return 0, err
	}

	if input == "" && defaultValue > 0 {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid integer: %w", err)
	}

	return value, nil
}

// PromptBool prompts for a boolean input
func PromptBool(prompt string, defaultValue bool) (bool, error) {
	suffix := " [y/N]"
	if defaultValue {
		suffix = " [Y/n]"
	}

	fmt.Printf("%s%s: ", prompt, suffix)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	// Sanitize and normalize input
	input = strings.ToLower(sanitizeInput(input))

	if input == "" {
		return defaultValue, nil
	}

	switch input {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %s", input)
	}
}

// PromptPassword prompts for a password input (hidden)
func PromptPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)

	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	fmt.Println() // Print newline after password input

	// Sanitize password input (though it won't be displayed)
	password := sanitizeInput(string(passwordBytes))

	return password, nil
}

// PromptChoice prompts for a choice from a list
func PromptChoice(prompt string, choices []string, defaultIndex int) (int, error) {
	PrintInfo(prompt)

	for i, choice := range choices {
		marker := " "
		if i == defaultIndex {
			marker = Success("●")
		}
		fmt.Printf("  %s %d. %s\n", marker, i+1, choice)
	}

	defaultStr := ""
	if defaultIndex >= 0 && defaultIndex < len(choices) {
		defaultStr = strconv.Itoa(defaultIndex + 1)
	}

	input, err := PromptString("Select", defaultStr)
	if err != nil {
		return 0, err
	}

	if input == "" && defaultIndex >= 0 {
		return defaultIndex, nil
	}

	selection, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid selection: %w", err)
	}

	if selection < 1 || selection > len(choices) {
		return 0, fmt.Errorf("selection out of range")
	}

	return selection - 1, nil
}

// PromptMultiChoice prompts for multiple choices from a list
func PromptMultiChoice(prompt string, choices []string) ([]int, error) {
	PrintInfo(prompt)

	for i, choice := range choices {
		fmt.Printf("  %d. %s\n", i+1, choice)
	}

	fmt.Println()
	fmt.Printf("Enter selections (comma-separated, e.g., 1,3,5): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)

	if input == "" {
		return []int{}, nil
	}

	parts := strings.Split(input, ",")
	selections := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		selection, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid selection: %s", part)
		}

		if selection < 1 || selection > len(choices) {
			return nil, fmt.Errorf("selection out of range: %d", selection)
		}

		selections = append(selections, selection-1)
	}

	return selections, nil
}

// PromptPath prompts for a file or directory path
func PromptPath(prompt string, defaultValue string) (string, error) {
	path, err := PromptString(prompt, defaultValue)
	if err != nil {
		return "", err
	}

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = strings.Replace(path, "~", homeDir, 1)
		}
	}

	return path, nil
}

// PromptRequired prompts for required input (cannot be empty)
func PromptRequired(prompt string) (string, error) {
	for {
		input, err := PromptString(prompt, "")
		if err != nil {
			return "", err
		}

		if input != "" {
			return input, nil
		}

		PrintError("This field is required. Please enter a value.")
	}
}

// PromptValidated prompts for input with validation
func PromptValidated(prompt string, validator func(string) error) (string, error) {
	for {
		input, err := PromptString(prompt, "")
		if err != nil {
			return "", err
		}

		if err := validator(input); err != nil {
			PrintError("Invalid input: %v", err)
			continue
		}

		return input, nil
	}
}

// PromptMenu displays a menu and returns the selected option
func PromptMenu(title string, options []MenuOption) (string, error) {
	PrintHeader(title)
	PrintEmptyLine()

	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, Bold(option.Label))
		if option.Description != "" {
			fmt.Printf("     %s\n", Dim(option.Description))
		}
	}

	PrintEmptyLine()
	fmt.Print(Info("Select an option: "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(options) {
		return "", fmt.Errorf("invalid selection")
	}

	return options[selection-1].Value, nil
}

// MenuOption represents a menu option
type MenuOption struct {
	Label       string
	Description string
	Value       string
}

// WaitForEnter waits for the user to press Enter
func WaitForEnter() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
