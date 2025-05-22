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
	"github.com/firebase/genkit/go/plugins/pinecone"
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
		genkit.WithPlugins(&googlegenai.GoogleAI{}, &pinecone.Pinecone{}),
		genkit.WithDefaultModel("googleai/gemini-1.5-pro"),
		genkit.WithPromptDir(promptDir),
	)
	if err != nil {
		log.Fatalf("could not initialize Genkit: %v", err)
	}

	// Fetch relevant Go Vibes docs
	fmt.Println(string(failureData))
	goVibesDocs, err := retrieveGoVibes(g, string(failureData))
	if err != nil {
		fmt.Printf("Warning: could not retrieve Go Vibes docs: %v\n", err)
		fmt.Println("Continuing without RAG context...")
	}

	docs := ""
	log.Printf("Go Vibes docs: %v", docs)
	if goVibesDocs != nil && len(goVibesDocs) > 0 {
		for _, doc := range goVibesDocs {
			log.Printf("Appending Go Vibes doc: %v", doc.Metadata["fileName"])
			if doc != nil && len(doc.Content) > 0 && doc.Content[0] != nil {
				docs += doc.Content[0].Text + "\n"
			}
		}
	}

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

	// Execute the prompt with the test failure data
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
