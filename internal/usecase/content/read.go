package content

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const _defaultWPM = 180

type ReadTimeCalculator interface {
	Calculate(content string) string
	CalculateMinutes(content string) int
}

type calculator struct {
	wordsPerMinute int
	codeBlockRegex *regexp.Regexp
	imageRegex     *regexp.Regexp
}

func NewCalculator() ReadTimeCalculator {
	return &calculator{
		wordsPerMinute: _defaultWPM,
		codeBlockRegex: regexp.MustCompile("```[\\s\\S]*?```|`[^`]*`"),
		imageRegex:     regexp.MustCompile(`!\[.*?\]\(.*?\)`),
	}
}

func (c *calculator) WithWordsPerMinute(wordsPerMinute int) *calculator {
	if wordsPerMinute > 0 {
		c.wordsPerMinute = wordsPerMinute
	}
	return c
}

func (c *calculator) Calculate(content string) string {
	readTime := c.CalculateMinutes(content)
	return FormatReadTime(readTime)
}

func (c *calculator) CalculateMinutes(content string) int {
	if content == "" {
		return 0
	}

	processed := c.preprocessContent(content)

	wordCount := c.countWords(processed)

	if wordCount == 0 {
		return 0
	}

	minutes := float64(wordCount) / float64(c.wordsPerMinute)

	adjustedMinutes := c.adjustForComplexity(content, minutes)

	if adjustedMinutes < 1.0 {
		return 1
	}

	return int(adjustedMinutes + 0.999)
}

func (c *calculator) preprocessContent(content string) string {
	processed := c.codeBlockRegex.ReplaceAllString(content, "")

	processed = c.imageRegex.ReplaceAllString(processed, "")

	htmlRegex := regexp.MustCompile(`<[^>]+>`)
	processed = htmlRegex.ReplaceAllString(processed, "")

	return processed
}

func (c *calculator) countWords(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	wordCount := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
			continue
		}

		if unicode.In(r, unicode.Han) {
			wordCount++
			inWord = false
			continue
		}

		if !inWord {
			wordCount++
			inWord = true
		}
	}

	return wordCount
}

func (c *calculator) adjustForComplexity(content string, baseMinutes float64) float64 {
	adjusted := baseMinutes

	codeBlocks := len(c.codeBlockRegex.FindAllString(content, -1))
	if codeBlocks > 0 {
		adjusted += float64(codeBlocks) * 0.5
	}

	images := len(c.imageRegex.FindAllString(content, -1))
	adjusted += float64(images) * 0.25

	mathFormulaRegex := regexp.MustCompile(`\$\$[^\$]+\$\$|\$[^\$]+\$`)
	formulas := len(mathFormulaRegex.FindAllString(content, -1))
	adjusted += float64(formulas) * 0.3

	paragraphs := strings.Split(content, "\n\n")
	for _, p := range paragraphs {
		lines := strings.Count(p, "\n") + 1
		if lines > 10 {
			adjusted += 0.2
		}
	}

	return adjusted
}

func FormatReadTime(minutes int) string {
	if minutes <= 0 {
		return "少于 1 分钟阅读"
	}

	return fmt.Sprintf("%d 分钟阅读", minutes)
}
