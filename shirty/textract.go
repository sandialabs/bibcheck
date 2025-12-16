// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cwpearson/bibliography-checker/config"
)

type TextractResponse struct {
	Id               any    `json:"id"`
	Text             string `json:"text"`
	Filepath         string `json:"filepath"`
	ExtractTimestamp string `json:"extract_timestamp"`
	ExtractUser      string `json:"extract_user"`
	Metadata         any    `json:"metadata"`
	Sections         []any  `json:"sections"`
}

func (w *Workflow) textractImpl(requestBody io.Reader, contentType string) (*TextractResponse, error) {
	// Create the request
	req, err := http.NewRequest("POST", w.baseUrl+"/extract/textract/create", requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", config.UserAgent())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var textractResp TextractResponse
	err = json.Unmarshal(body, &textractResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &textractResp, nil
}

func (w *Workflow) TextractContent(data []byte) (*TextractResponse, error) {
	// Create a buffer to write our multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create the form file field
	part, err := writer.CreateFormFile("file", "document.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Write the byte data to the form
	_, err = part.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	return w.textractImpl(&requestBody, writer.FormDataContentType())
}

func (w *Workflow) Textract(filePath string) (*TextractResponse, error) {
	// Create a buffer to write our multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create the form file field
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy the file content to the form
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	return w.textractImpl(&requestBody, writer.FormDataContentType())
}
