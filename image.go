package aimini

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
)

// ImageSource describes an image file sent to queue.add.
type ImageSource interface {
	open() (io.ReadCloser, string, string, error)
}

type imageFile struct {
	path        string
	contentType string
}

type imageBytes struct {
	data        []byte
	filename    string
	contentType string
}

// ImageFromFile sends an image from a local path.
func ImageFromFile(path string) ImageSource {
	return &imageFile{path: path}
}

// ImageFromBytes sends an image from memory. Filename should include an image
// extension when possible, for example "input.jpg".
func ImageFromBytes(data []byte, filename, contentType string) ImageSource {
	cp := make([]byte, len(data))
	copy(cp, data)
	return &imageBytes{data: cp, filename: filename, contentType: contentType}
}

func (i *imageFile) open() (io.ReadCloser, string, string, error) {
	if i == nil || i.path == "" {
		return nil, "", "", fmt.Errorf("aimini: image path is required")
	}

	file, err := os.Open(i.path)
	if err != nil {
		return nil, "", "", err
	}

	filename := filepath.Base(i.path)
	contentType := i.contentType
	if contentType == "" {
		contentType = contentTypeFromFilename(filename)
	}
	return file, filename, contentType, nil
}

func (i *imageBytes) open() (io.ReadCloser, string, string, error) {
	if i == nil || len(i.data) == 0 {
		return nil, "", "", fmt.Errorf("aimini: image bytes are required")
	}

	filename := i.filename
	if filename == "" {
		filename = "image"
	}

	contentType := i.contentType
	if contentType == "" {
		contentType = contentTypeFromFilename(filename)
	}

	return io.NopCloser(bytes.NewReader(i.data)), filename, contentType, nil
}

func contentTypeFromFilename(filename string) string {
	if ext := filepath.Ext(filename); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}
