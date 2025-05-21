package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Println("::warning::GEMINI_API_KEY environment variable is not set")
		fmt.Println("Tests failed, but cannot provide AI suggestions without an API key.")
		fmt.Println("Please set the GEMINI_API_KEY secret in your GitHub repository.")
		os.Exit(0) // Exit gracefully to not fail the workflow
	}

	// Read test failures from file
	failureData, err := ioutil.ReadFile("test_failures.txt")
	if err != nil {
		fmt.Printf("::warning::Error reading test failures: %v\n", err)
		fmt.Println("Could not read test failures file. Make sure there are failing tests.")
		os.Exit(0) // Exit gracefully
	}

	if len(failureData) == 0 {
		fmt.Println("::warning::No test failures detected in the output")
		os.Exit(0)
	}

	// Create request payload for Gemini API
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": fmt.Sprintf("I have the following Go test failures. Please analyze and suggest fixes:\n\n%s", string(failureData)),
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
