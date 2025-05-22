package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	Analysis  string `json:"analysis"`
	CodeDiffs string `json:"code_diffs"`
}

// FileContentInput represents the input for the file content tool
type FileContentInput struct {
	FilePath string `json:"file_path" jsonschema_description:"Relative path to the file from repo root"`
}

// RelevantFilesOutput represents the output from the first stage of analysis
type RelevantFilesOutput struct {
	FilePaths []string `json:"file_paths"`
}

// Run the program
func run() {
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

	// Define the file content tool using cat command
	fileContentTool := genkit.DefineTool(
		g, "getFileContent", "Fetches the content of a file using the cat command",
		func(ctx *ai.ToolContext, input FileContentInput) (string, error) {
			// Use cat command to read file content
			cmd := exec.Command("cat", fmt.Sprintf("%s/../%s", workDir, input.FilePath))
			fmt.Println("Reading file:", input.FilePath)
			output, err := cmd.Output()
			fmt.Println(string(output))
			if err != nil {
				return "", fmt.Errorf("could not read file %s: %v", input.FilePath, err)
			}
			return string(output), nil
		},
	)

	// Define the test analyzer flow
	testAnalyzerFlow := genkit.DefineFlow(g, "testAnalyzerFlow",
		func(ctx context.Context, input TestAnalyzerInput) (TestAnalyzerOutput, error) {
			// Step 1: Find relevant files based on error message using Dotprompt
			findFilesPrompt := genkit.LookupPrompt(g, "find_files")
			if findFilesPrompt == nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not find prompt 'find_files'")
			}

			// Execute find files prompt
			findFilesResp, err := findFilesPrompt.Execute(ctx,
				ai.WithInput(map[string]interface{}{
					"failures":  input.FailureData,
					"repo_tree": input.RepoTree,
				}),
			)
			if err != nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not execute find files prompt: %v", err)
			}

			// Extract relevant files from structured output
			var relevantFiles RelevantFilesOutput
			if err := findFilesResp.Output(&relevantFiles); err != nil {
				log.Printf("Warning: could not extract structured output: %v", err)

				// Fallback to parsing the response text
				var filePaths []string
				if err := json.Unmarshal([]byte(findFilesResp.Text()), &filePaths); err != nil {
					// Try extracting from markdown code block
					text := findFilesResp.Text()
					if start := strings.Index(text, "```json"); start != -1 {
						start += 7
						if end := strings.Index(text[start:], "```"); end != -1 {
							jsonContent := text[start : start+end]
							err = json.Unmarshal([]byte(jsonContent), &filePaths)
						}
					}

					if err != nil || len(filePaths) == 0 {
						// Fallback: try to parse file paths line by line
						lines := strings.Split(findFilesResp.Text(), "\n")
						for _, line := range lines {
							line = strings.TrimSpace(line)
							if strings.HasSuffix(line, ".go") && !strings.Contains(line, " ") {
								filePaths = append(filePaths, line)
							}
						}
					}
				}

				relevantFiles = RelevantFilesOutput{
					FilePaths: filePaths,
				}
			}

			fmt.Println("Relevant files identified:", relevantFiles.FilePaths)

			// Step 2: Use RAG to get any relevant documentation
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

			// Step 3: For each relevant file, fetch its content
			fileContents := make(map[string]string)
			for _, filePath := range relevantFiles.FilePaths {
				// Execute the file content tool directly
				result, err := fileContentTool.RunRaw(ctx, map[string]interface{}{
					"file_path": filePath,
				})
				if err != nil {
					log.Printf("Warning: could not read file %s: %v", filePath, err)
					continue
				}
				content, ok := result.(string)
				if !ok {
					log.Printf("Warning: unexpected result type for file %s", filePath)
					continue
				}
				fileContents[filePath] = content
			}

			// Convert file contents to string format for prompt
			fileContentStr := ""
			for path, content := range fileContents {
				fileContentStr += fmt.Sprintf("File: %s\n```go\n%s\n```\n\n", path, content)
			}

			// Step 4: Call the analysis prompt with file contents using Dotprompt
			analyzeFilesPrompt := genkit.LookupPrompt(g, "test_analyzer")
			if analyzeFilesPrompt == nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not find prompt 'analyze_files'")
			}

			// Prepare input data for the analysis prompt
			inputData := map[string]interface{}{
				"failures":      input.FailureData,
				"file_contents": fileContentStr,
			}

			if docsContent != "" {
				inputData["docs"] = docsContent
			}

			// Execute the analysis prompt
			resp, err := analyzeFilesPrompt.Execute(ctx, ai.WithInput(inputData))
			if err != nil {
				return TestAnalyzerOutput{}, fmt.Errorf("could not execute analysis prompt: %v", err)
			}

			// Return the analysis as plain text
			return TestAnalyzerOutput{
				Analysis:  resp.Text(),
				CodeDiffs: "", // Included in the analysis text
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

func main() {
	run()
}
