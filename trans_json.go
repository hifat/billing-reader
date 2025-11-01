package main

import (
	"context"
	"log"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

type Receipt struct {
	Name             string  `json:"name"`
	NameEdited       string  `json:"nameEdited"`
	PurchasePrice    float64 `json:"purchasePrice"`
	PurchaseQuantity float64 `json:"purchaseQuantity"`
	Remark           string  `json:"remark"`
}

func ToJson(ctx context.Context, jtxt []byte) ([]Receipt, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")

	g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{
		APIKey: apiKey,
	}))

	prompt := `
Extract product information from the receipt data. Follow these rules strictly:

Field Extraction:
- "name": Extract the exact product name as written in the receipt
- "nameEdited": Correct any spelling errors or typos in the product name
- "purchasePrice": Extract the price per unit as a number
- "purchaseQuantity": Extract the quantity as a number
- "remark": Extract ONLY if there is an actual customer note or special instruction written on the receipt. DO NOT add your own comments or explanations.

Critical Rules:
- If a field has no data, use empty string ("") for text or 0 for numbers
- DO NOT put correction explanations in the remark field
- DO NOT add your own observations or processing notes
- The remark field must ONLY contain text that appears on the original receipt as a customer note or special instruction
- If there is no customer note on the receipt, leave remark empty
- Keep numbers as decimal format (use . not ,)

Receipt Data: %s
	`

	resp, err := genkit.Generate(ctx, g,
		ai.WithPrompt(prompt, jtxt),
		ai.WithModelName("googleai/gemini-2.5-flash-lite"),
		ai.WithOutputType([]Receipt{}),
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	items := []Receipt{}
	if err := resp.Output(&items); err != nil {
		log.Fatalf("Failed to unmarshals items: %v", err)
		return nil, err
	}

	return items, nil
}
