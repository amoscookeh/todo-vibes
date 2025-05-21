package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

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
		genkit.WithPlugins(&googlegenai.GoogleAI{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-pro"),
		genkit.WithPromptDir(promptDir),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Look up the prompt from the Dotprompt file
	testAnalyzerPrompt := genkit.LookupPrompt(g, "test_analyzer")
	if testAnalyzerPrompt == nil {
		log.Fatalf("could not find prompt 'test_analyzer'")
	}

	// Execute the prompt with the test failure data
	resp, err := testAnalyzerPrompt.Execute(ctx,
		ai.WithInput(map[string]interface{}{
			"failures": string(failureData),
		}),
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
