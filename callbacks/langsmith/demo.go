package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino-ext/callbacks/langsmith"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
)

func main() {
	// Create a langsmith handler
	// In a real application, you would get the API key from environment variables or a config file.
	cbh, err := langsmith.NewLangsmithHandler(&langsmith.Config{
		APIKey: "YOUR_LANGSMITH_API_KEY",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Set langsmith as a global callback handler
	callbacks.AppendGlobalHandlers(cbh)

	// Set trace-level information using the context
	ctx := context.Background()
	ctx = langsmith.SetTrace(ctx,
		langsmith.WithProjectName("my-awesome-project"),
		langsmith.WithMetadata(map[string]interface{}{"environment": "production"}),
	)

	// Build and compile an eino graph
	g := compose.NewGraph[string, string]()
	// ... add nodes and edges to your graph
	runner, _ := g.Compile(ctx)

	// Invoke the runner
	result, _ := runner.Invoke(ctx, "some input")

	// Process the result
	log.Printf("Got result: %s", result)
}
