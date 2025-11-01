package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/option"
)

func ExtractFromImage(ctx context.Context, credentials string, filePath string) ([]Receipt, error) {
	client, err := vision.NewImageAnnotatorClient(ctx, option.WithCredentialsFile(credentials))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
		return nil, err
	}
	defer client.Close()

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open image: %v", err)
		return nil, err
	}
	defer file.Close()

	image, err := vision.NewImageFromReader(file)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
		return nil, err
	}

	annotation, err := client.DetectDocumentText(ctx, image, nil)
	if err != nil {
		log.Fatalf("Failed to detect text: %v", err)
		return nil, err
	}

	if annotation == nil {
		return nil, nil
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
		return nil, err
	}

	return ToJson(ctx, jsonData)
}
