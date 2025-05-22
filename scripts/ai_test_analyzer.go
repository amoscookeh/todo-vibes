package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

type TestContext struct {
	failureOutput string
	testCode      string
	implCode      string
	failedTests   []string
	filePath      string
}

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable is not set")
		os.Exit(1)
	}

	// Read test failures from file
	failureData, err := ioutil.ReadFile("test_failures.txt")
	if err != nil {
		fmt.Printf("Error reading test failures: %v\n", err)
		os.Exit(1)
	}

	// Extract test context
	contextData := extractTestContext(string(failureData))

	// Log the extracted context
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
		fmt.Println(contextData.testCode[:500] + "...\n(truncated, full context sent to AI)")
	} else {
		fmt.Println(contextData.testCode)
	}

	fmt.Println("\n📋 IMPLEMENTATION CODE SUMMARY:")
	if len(contextData.implCode) > 500 {
		fmt.Println(contextData.implCode[:500] + "...\n(truncated, full context sent to AI)")
	} else {
		fmt.Println(contextData.implCode)
	}
	fmt.Println("::endgroup::")

	// Initialize Genkit context
	ctx := context.Background()

	// Get current working directory to locate prompts
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current directory: %v", err)
	}

	promptDir := filepath.Join(workDir, "prompts")

	// Initialize Genkit with Google AI plugin and Dotprompt
	g, err := genkit.Init(ctx,
		genkit.WithPlugins(&googlegenai.GoogleAI{}, &pinecone.Pinecone{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-pro"),
		genkit.WithPromptDir(promptDir),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Fetch relevant Go Vibes docs
	goVibesDocs, err := retrieveGoVibes(g, string(failureData))
	if err != nil {
		fmt.Printf("Warning: could not retrieve Go Vibes docs: %v\n", err)
		fmt.Println("Continuing without RAG context...")
	}

	docs := ""
	if goVibesDocs != nil && len(goVibesDocs) > 0 {
		for _, doc := range goVibesDocs {
			if doc != nil && len(doc.Content) > 0 && doc.Content[0] != nil {
				docs += doc.Content[0].Text + "\n"
			}
		}
	}

	log.Printf("Go Vibes docs: %v", docs)

	// Look up the prompt from the Dotprompt file
	testAnalyzerPrompt := genkit.LookupPrompt(g, "test_analyzer")
	if testAnalyzerPrompt == nil {
		log.Fatalf("could not find prompt 'test_analyzer'")
	}

	inputData := map[string]interface{}{
		"failures": string(failureData),
	}

	if docs != "" {
		inputData["docs"] = docs
	}

	// Add test and implementation code to input data
	inputData["testCode"] = contextData.testCode
	inputData["implCode"] = contextData.implCode

	// Execute the prompt with the test failure data and context
	resp, err := testAnalyzerPrompt.Execute(ctx,
		ai.WithInput(inputData),
	)
	if err != nil {
		log.Fatalf("could not execute prompt: %v", err)
	}

	// Extract and print suggestions
	fmt.Println("::group::Gemini Test Failure Analysis")
	fmt.Println("⚠️ Test failures detected! Gemini AI suggestions:")
	fmt.Println("-------------------------------------------")

	// Print the response text
	fmt.Println(resp.Text())

	fmt.Println("-------------------------------------------")
	fmt.Println("::endgroup::")
}

func retrieveGoVibes(g *genkit.Genkit, query string) ([]*ai.Document, error) {
	indexId := "go-vibes"
	ctx := context.Background()

	embedder := googlegenai.GoogleAIEmbedder(g, "text-embedding-004")
	retriever, err := pinecone.DefineRetriever(ctx, g, pinecone.Config{
		IndexID:  indexId,
		Embedder: embedder,
	})
	if err != nil {
		return nil, err
	}

	fmt.Println(query)
	resp, err := retriever.Retrieve(ctx, &ai.RetrieverRequest{
		Query:   ai.DocumentFromText(query, nil),
		Options: nil,
	})
	if err != nil {
		return nil, err
	}

	return resp.Documents, nil
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
