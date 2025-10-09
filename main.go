package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

type OCRResult struct {
	FullText string      `json:"text"`
	Blocks   []TextBlock `json:"blocks"`
}

type TextBlock struct {
	Text        string  `json:"text"`
	Confidence  float32 `json:"confidence"`
	BoundingBox []Point `json:"bounding_box"`
}

type Point struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

func init() {
	if err := godotenv.Load("./env/.env"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile("./env/cloud-vision-srv-scc.json"))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fileName := "./upload/receipt.jpg"
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Failed to open image: %v", err)
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	annotation, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		log.Fatalf("Failed to detect text: %v", err)
	}

	if annotation == nil {
		fmt.Println(`{"text": "", "blocks": []}`)
		return
	}

	result := OCRResult{
		FullText: annotation.Text,
	}

	for _, page := range annotation.Pages {
		for _, block := range page.Blocks {
			var blockText string
			for _, paragraph := range block.Paragraphs {
				for _, word := range paragraph.Words {
					for _, symbol := range word.Symbols {
						blockText += symbol.Text
					}
					blockText += " "
				}
			}

			bbox := make([]Point, 0, len(block.BoundingBox.Vertices))
			for _, v := range block.BoundingBox.Vertices {
				bbox = append(bbox, Point{X: v.X, Y: v.Y})
			}

			result.Blocks = append(result.Blocks, TextBlock{
				Text:        blockText,
				Confidence:  block.Confidence,
				BoundingBox: bbox,
			})
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	ToJson(ctx, jsonData)
}
