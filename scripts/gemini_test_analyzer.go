package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("::warning::GEMINI_API_KEY environment variable is not set")
		fmt.Println("Tests failed, but cannot provide AI suggestions without an API key.")
		fmt.Println("Please set the GEMINI_API_KEY secret in your GitHub repository.")
		os.Exit(0)
	}

	// Read test failures from file
	failureData, err := ioutil.ReadFile("test_failures.txt")
	if err != nil {
		fmt.Printf("::warning::Error reading test failures: %v\n", err)
		fmt.Println("Could not read test failures file. Make sure there are failing tests.")
		os.Exit(0)
	}

	if len(failureData) == 0 {
		fmt.Println("::warning::No test failures detected in the output")
		os.Exit(0)
	}

	// Extract failed test names and context
	contextData := extractTestContext(string(failureData))

	// Log the extracted context to monitor what's being sent
	fmt.Println("::group::Extracted Test Context")
	fmt.Println("📋 EXTRACTED FAILING TESTS:")
	for _, test := range contextData.failedTests {
		fmt.Println(" - " + test)
	}
	if len(contextData.failedTests) == 0 {
		fmt.Println(" (No specific test names extracted)")
	}

	fmt.Println("\n📋 FILE PATH (if detected):")
	if contextData.filePath != "" {
		fmt.Println(" - " + contextData.filePath)
	} else {
		fmt.Println(" (No file path detected)")
	}

	fmt.Println("\n📋 TEST CODE SUMMARY:")
	if len(contextData.testCode) > 500 {
		// Print first 500 chars with ellipsis
		fmt.Println(contextData.testCode[:500] + "...\n(truncated, full context sent to AI)")
	} else {
		fmt.Println(contextData.testCode)
	}

	fmt.Println("\n📋 IMPLEMENTATION CODE SUMMARY:")
	if len(contextData.implCode) > 500 {
		// Print first 500 chars with ellipsis
		fmt.Println(contextData.implCode[:500] + "...\n(truncated, full context sent to AI)")
	} else {
		fmt.Println(contextData.implCode)
	}
	fmt.Println("::endgroup::")

	// Create request payload for Gemini API with improved prompt
	prompt := fmt.Sprintf(`You are an expert Go developer tasked with analyzing and fixing test failures. Focus exclusively on identifying the root cause and providing actionable fixes.

FAILING TEST OUTPUT:
%s

TEST CODE:
%s

RELATED IMPLEMENTATION CODE:
%s

Analyze this test failure following these steps:
1. Identify the exact failing assertion or error in the test
2. Determine why the test is failing (be specific about line numbers and conditions)
3. Look for disconnects between what the test expects and what the implementation actually does
4. Provide a concise fix recommendation with code snippets, clearly indicating:
   - Which file needs to be modified
   - Exactly what code should be changed
   - The correct implementation (with complete code)

Your response should be structured as:
- Root Cause: [brief explanation of why the test is failing]
- Affected Files: [list files that need modification]
- Fix: [code snippets with clear before/after changes]
- Explanation: [brief explanation of why the fix works]

Focus only on the immediate test failure, not refactoring or improving the codebase beyond fixing the test.`,
		contextData.failureOutput,
		contextData.testCode,
		contextData.implCode)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("::warning::Error marshaling request: %v\n", err)
		os.Exit(0)
	}

	// Call Gemini API
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("::warning::Error calling Gemini API: %v\n", err)
		os.Exit(0)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("::warning::Error reading response: %v\n", err)
		os.Exit(0)
	}

	// Parse response
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Printf("::warning::Error parsing response: %v\n", err)
		os.Exit(0)
	}

	// Extract and print suggestions
	fmt.Println("::group::Gemini Test Failure Analysis")
	fmt.Println("⚠️ Test failures detected! Gemini AI suggestions:")
	fmt.Println("-------------------------------------------")

	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							fmt.Println(text)
						}
					}
				}
			}
		}
	} else {
		fmt.Println("No suggestions received from Gemini API")
		if result["error"] != nil {
			fmt.Printf("API Error: %v\n", result["error"])
		}
	}

	fmt.Println("-------------------------------------------")
	fmt.Println("::endgroup::")
}

type TestContext struct {
	failureOutput string
	testCode      string
	implCode      string
	failedTests   []string
	filePath      string
}

func extractTestContext(output string) TestContext {
	context := TestContext{
		failureOutput: output,
		testCode:      "No test code found",
		implCode:      "No implementation code found",
		failedTests:   []string{},
	}

	// Extract failed test names using regex
	re := regexp.MustCompile(`--- FAIL: (Test\w+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) > 1 {
			context.failedTests = append(context.failedTests, match[1])
		}
	}

	// Extract file path if present
	pathRe := regexp.MustCompile(`([^:\s]+_test\.go)`)
	pathMatches := pathRe.FindStringSubmatch(output)
	if len(pathMatches) > 1 {
		context.filePath = pathMatches[1]
	}

	// Get test code and implementation code
	context.testCode = findTestCode(context.failedTests)
	context.implCode = findImplementationCode(context.testCode, context.failedTests)

	return context
}

func findTestCode(testNames []string) string {
	// If no test names found, try to find all test files
	if len(testNames) == 0 {
		return getAllTestCode()
	}

	var result strings.Builder
	result.WriteString("// Failing tests:\n")

	// Find test files in current directory and subdirectories
	testFiles, err := findFiles(".", "_test.go")
	if err != nil {
		fmt.Printf("Warning: Error finding test files: %v\n", err)
		return "Could not locate test code"
	}

	// For each test file, look for the failing test function
	for _, file := range testFiles {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		for _, testName := range testNames {
			// Look for the test function definition
			re := regexp.MustCompile(`func\s+(` + testName + `)\s*\([^)]*\)\s*{(?:[^{}]|{[^{}]*})*}`)
			match := re.FindString(fileContent)
			if match != "" {
				result.WriteString(fmt.Sprintf("// From file: %s\n", file))
				result.WriteString(match)
				result.WriteString("\n\n")
			}
		}
	}

	if result.Len() > 25 { // "// Failing tests:\n" is 17 chars
		return result.String()
	}
	return getAllTestCode()
}

func findImplementationCode(testCode string, testNames []string) string {
	// Extract function/method names that might be used in the test
	var funcs []string

	// Look for common patterns in test code like repository.Create, service.Update, etc.
	re := regexp.MustCompile(`([a-zA-Z0-9_]+)\.([A-Z][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(testCode, -1)
	for _, match := range matches {
		if len(match) > 2 {
			funcs = append(funcs, match[2])
		}
	}

	// Also extract function names from test names (TestCreateTodo -> CreateTodo)
	for _, testName := range testNames {
		if strings.HasPrefix(testName, "Test") {
			funcName := strings.TrimPrefix(testName, "Test")
			funcs = append(funcs, funcName)
		}
	}

	// Find non-test go files
	files, err := findFiles(".", ".go")
	if err != nil {
		return "Could not find implementation files"
	}

	var implFiles []string
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") {
			implFiles = append(implFiles, file)
		}
	}

	var result strings.Builder
	// For each implementation file, look for functions that match the patterns we found
	for _, file := range implFiles {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		var fileMatched bool

		for _, funcName := range funcs {
			if strings.Contains(fileContent, funcName) {
				fileMatched = true
				break
			}
		}

		if fileMatched {
			result.WriteString(fmt.Sprintf("// From file: %s\n", file))
			// Get up to 200 lines to avoid overwhelming
			lines := strings.Split(fileContent, "\n")
			maxLines := 200
			if len(lines) > maxLines {
				lines = lines[:maxLines]
				result.WriteString(strings.Join(lines, "\n"))
				result.WriteString("\n// ... (file truncated for brevity)")
			} else {
				result.WriteString(fileContent)
			}
			result.WriteString("\n\n")
		}
	}

	if result.Len() > 0 {
		return result.String()
	}

	// If no matches found, return a small sample of implementation files
	return getRandomImplementationSample(implFiles)
}

func getAllTestCode() string {
	files, err := findFiles(".", "_test.go")
	if err != nil {
		return "Could not locate test code"
	}

	var result strings.Builder
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		result.WriteString(fmt.Sprintf("// From file: %s\n", file))
		// Get up to 200 lines to avoid overwhelming
		lines := strings.Split(string(content), "\n")
		maxLines := 200
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			result.WriteString(strings.Join(lines, "\n"))
			result.WriteString("\n// ... (file truncated for brevity)")
		} else {
			result.WriteString(string(content))
		}
		result.WriteString("\n\n")
	}

	if result.Len() > 0 {
		return result.String()
	}
	return "No test code found"
}

func getRandomImplementationSample(files []string) string {
	if len(files) == 0 {
		return "No implementation files found"
	}

	var result strings.Builder
	// Take up to 3 files
	maxFiles := 3
	if len(files) > maxFiles {
		files = files[:maxFiles]
	}

	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		result.WriteString(fmt.Sprintf("// Sample from file: %s\n", file))
		// Get up to 100 lines per file to avoid overwhelming
		lines := strings.Split(string(content), "\n")
		maxLines := 100
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			result.WriteString(strings.Join(lines, "\n"))
			result.WriteString("\n// ... (file truncated for brevity)")
		} else {
			result.WriteString(string(content))
		}
		result.WriteString("\n\n")
	}

	return result.String()
}

func findFiles(root, suffix string) ([]string, error) {
	var files []string

	// Try using find command first (more efficient)
	cmd := exec.Command("find", root, "-type", "f", "-name", fmt.Sprintf("*%s", suffix))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
	}

	// Fall back to Go's filepath.Walk
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, suffix) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
