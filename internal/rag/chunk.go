package rag

import "strings"

type Chunk struct {
	ID      string    `json:"id"`
	Source  string    `json:"source"`
	Path    string    `json:"path"`
	Heading string    `json:"heading,omitempty"`
	Text    string    `json:"text"`
	Vector  []float64 `json:"vector,omitempty"`
}

func ChunkDocument(doc Document, maxChars int) []Chunk {
	if maxChars <= 0 {
		maxChars = 1200
	}
	sections := splitByHeading(doc.Text)
	var chunks []Chunk
	for _, section := range sections {
		text := strings.TrimSpace(section.text)
		if text == "" {
			continue
		}
		for len(text) > maxChars {
			chunks = append(chunks, newChunk(doc, section.heading, text[:maxChars], len(chunks)))
			text = strings.TrimSpace(text[maxChars:])
		}
		chunks = append(chunks, newChunk(doc, section.heading, text, len(chunks)))
	}
	return chunks
}

type section struct {
	heading string
	text    string
}

func splitByHeading(text string) []section {
	lines := strings.Split(text, "\n")
	current := section{}
	sections := []section{}
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			if strings.TrimSpace(current.text) != "" {
				sections = append(sections, current)
			}
			current = section{heading: strings.TrimSpace(strings.TrimLeft(line, "# ")), text: line + "\n"}
			continue
		}
		current.text += line + "\n"
	}
	if strings.TrimSpace(current.text) != "" {
		sections = append(sections, current)
	}
	return sections
}

func newChunk(doc Document, heading string, text string, index int) Chunk {
	return Chunk{ID: doc.Source + ":" + doc.Path + ":" + string(rune(index+'0')), Source: doc.Source, Path: doc.Path, Heading: heading, Text: strings.TrimSpace(text)}
}
