package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

// TestAnalyzerInput represents the input to our test analyzer workflow
type TestAnalyzerInput struct {
	FailureData string `json:"failure_data"`
	RepoTree    string `json:"repo_tree"`
}

// TestAnalyzerOutput represents the output from our test analyzer workflow
type TestAnalyzerOutput struct {
	Analysis string `json:"analysis"`
}

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: GEMINI_API_KEY environment variable is not set")
		os.Exit(1)
	}

	// Read test failures from file
	failureData, err := ioutil.ReadFile("test_output.txt")
	if err != nil {
		fmt.Printf("Error reading test failures: %v\n", err)
		os.Exit(1)
	}

	// Get repo tree
	repoTreeOutput, err := getRepoTree()
	if err != nil {
		fmt.Printf("Error getting repository tree: %v\n", err)
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
		genkit.WithPlugins(&googlegenai.GoogleAI{}, &pinecone.Pinecone{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-pro"),
		genkit.WithPromptDir(promptDir),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Define the test analyzer flow
	testAnalyzerFlow := genkit.DefineFlow(g, "testAnalyzerFlow",
		func(ctx context.Context, input TestAnalyzerInput) (TestAnalyzerOutput, error) {
			// Step 1: Use failure data to prompt RAG
			docs, err := retrieveGoVibes(g, input.FailureData)
			if err != nil {
				log.Printf("Warning: could not retrieve Go Vibes docs: %v", err)
				log.Println("Continuing without RAG context...")
			}

			// Process documents
			docsContent := ""
			if docs != nil && len(docs) > 0 {
				for _, doc := range docs {
					log.Printf("Appending Go Vibes doc: %v", doc.Metadata["fileName"])
					if doc != nil && len(doc.Content) > 0 && doc.Content[0] != nil {
						docsContent += doc.Content[0].Text + "\n"
					}
				}
			}

			// Step 3: Call the prompt
			testAnalyzerPrompt := genkit.LookupPrompt(g, "test_analyzer")
			if testAnalyzerPrompt == nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not find prompt 'test_analyzer'")
			}

			// Prepare input data for the prompt
			inputData := map[string]interface{}{
				"failures":  input.FailureData,
				"repo_tree": input.RepoTree,
			}

			if docsContent != "" {
				inputData["docs"] = docsContent
			}

			// Execute the prompt with the test failure data
			resp, err := testAnalyzerPrompt.Execute(ctx,
				ai.WithInput(inputData),
			)
			if err != nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not execute prompt: %v", err)
			}

			// Return the analysis
			return TestAnalyzerOutput{
				Analysis: resp.Text(),
			}, nil
		},
	)

	// Run the flow with our input
	result, err := testAnalyzerFlow.Run(ctx, TestAnalyzerInput{
		FailureData: string(failureData),
		RepoTree:    repoTreeOutput,
	})
	if err != nil {
		log.Fatalf("could not run flow: %v", err)
	}

	// Output the results
	fmt.Println("::group::Gemini Test Failure Analysis")
	fmt.Println("⚠️ Test failures detected! Gemini AI suggestions:")
	fmt.Println("-------------------------------------------")
	fmt.Println(result.Analysis)
	fmt.Println("-------------------------------------------")
	fmt.Println("::endgroup::")
}

func retrieveGoVibes(g *genkit.Genkit, query string) ([]*ai.Document, error) {
	indexId := "go-vibes-docs"
	ctx := context.Background()

	embedder := googlegenai.GoogleAIEmbedder(g, "text-embedding-004")
	retriever, err := pinecone.DefineRetriever(ctx, g, pinecone.Config{
		IndexID:  indexId,
		Embedder: embedder,
	})
	if err != nil {
		return nil, err
	}

	resp, err := retriever.Retrieve(ctx, &ai.RetrieverRequest{
		Query: ai.DocumentFromText(query, nil),
		Options: &pinecone.RetrieverOptions{
			Namespace: "go-vibes-docs",
			Count:     10,
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Documents, nil
}

func getRepoTree() (string, error) {
	// Get the directory that contains the workspace root
	cmd := exec.Command("bash", "-c", "cd \"$(git rev-parse --show-toplevel 2>/dev/null || pwd)\" && tree")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to regular tree if git command fails
		cmd = exec.Command("tree")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get repository tree: %v", err)
		}
	}

	return string(output), nil
}
