package repositorychecks

import (
	"fmt"
	"go/scanner"
	"go/token"
	"path"
	"strings"
)

// The dash characters engineering datum 0.8 forbids, written as code points so
// this file does not contain the characters it hunts for.
const (
	enDashCharacter rune = 0x2013
	emDashCharacter rune = 0x2014
)

// minimumHyphenRunLength is the shortest run a document converter leaves in place
// of a dash, and longFlagHyphenCount is how many hyphens open a long command line
// flag. They are equal, which is exactly why a flag needs the extra word boundary
// test before it is exempt.
const (
	minimumHyphenRunLength = 2
	longFlagHyphenCount    = 2
)

// hyphenCharacter is the character whose runs of two or more a document
// converter leaves behind in place of a dash.
const hyphenCharacter = '-'

// flagTokenPrecedingCharacters are the characters that may come immediately
// before a command line flag token such as a double hyphen followed by a word,
// which datum 0.8 exempts. They cover a flag at the start of a word, inside
// backticks or quotes, inside brackets, and after an equals sign.
const flagPrecedingCharacters = " \t`'\"([="

// ScanProse reports dash characters and converter mangled hyphen runs in one
// tracked text file.
//
// In a Go file only comments and string literals are prose, so hyphen runs are
// looked for there alone; the decrement operator is code, not a mangled dash.
// Dash characters are reported anywhere, because they have no meaning in Go
// outside prose.
func ScanProse(relativePath string, fileContent []byte) []Violation {
	violations := scanDashCharacters(relativePath, string(fileContent))

	if path.Ext(relativePath) == ".go" {
		return append(violations, scanGoProseForHyphenRuns(relativePath, fileContent)...)
	}

	return append(violations, scanTextForHyphenRuns(relativePath, string(fileContent), 1)...)
}

func scanDashCharacters(relativePath string, textContent string) []Violation {
	var violations []Violation

	for lineIndex, lineText := range strings.Split(textContent, "\n") {
		for _, forbiddenCharacter := range []rune{enDashCharacter, emDashCharacter} {
			if strings.ContainsRune(lineText, forbiddenCharacter) {
				violations = append(violations, Violation{
					Path:       relativePath,
					LineNumber: lineIndex + 1,
					Rule:       RuleDashCharacter,
					Detail: fmt.Sprintf(
						"forbidden dash character U+%04X; use a comma, colon, semicolon, parentheses or a new sentence",
						forbiddenCharacter,
					),
				})
			}
		}
	}

	return violations
}

// scanTextForHyphenRuns checks each line of text, numbering lines from
// firstLineNumber so a Go comment or string can report the line it sits on.
func scanTextForHyphenRuns(relativePath string, textContent string, firstLineNumber int) []Violation {
	var violations []Violation

	for lineOffset, lineText := range strings.Split(textContent, "\n") {
		if isMarkdownTableSeparatorRow(lineText) {
			continue
		}

		for _, runLength := range forbiddenHyphenRunLengths(lineText) {
			violations = append(violations, Violation{
				Path:       relativePath,
				LineNumber: firstLineNumber + lineOffset,
				Rule:       RuleHyphenRun,
				Detail:     fmt.Sprintf("run of %d hyphens in prose, a converter mangled dash", runLength),
			})
		}
	}

	return violations
}

// forbiddenHyphenRunLengths returns the length of every run of two or more
// hyphens in the line that is not an exempt command line flag token.
func forbiddenHyphenRunLengths(lineText string) []int {
	var runLengths []int

	lineRunes := []rune(lineText)

	for runStart := 0; runStart < len(lineRunes); {
		if lineRunes[runStart] != hyphenCharacter {
			runStart++

			continue
		}

		runEnd := runStart
		for runEnd < len(lineRunes) && lineRunes[runEnd] == hyphenCharacter {
			runEnd++
		}

		runLength := runEnd - runStart
		if runLength >= minimumHyphenRunLength && !isCommandLineFlagToken(lineRunes, runStart, runLength) {
			runLengths = append(runLengths, runLength)
		}

		runStart = runEnd
	}

	return runLengths
}

// isCommandLineFlagToken reports whether a hyphen run is the start of a long
// flag: exactly two hyphens, at the start of a word, followed by a letter. A
// triple hyphen is never a flag, and two hyphens between words are the mangled
// dash the rule exists to catch.
func isCommandLineFlagToken(lineRunes []rune, runStart int, runLength int) bool {
	if runLength != longFlagHyphenCount {
		return false
	}

	startsWord := runStart == 0 || strings.ContainsRune(flagPrecedingCharacters, lineRunes[runStart-1])

	followingIndex := runStart + runLength
	followedByLetter := followingIndex < len(lineRunes) && isASCIILetter(lineRunes[followingIndex])

	return startsWord && followedByLetter
}

func isASCIILetter(character rune) bool {
	return (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
}

// isMarkdownTableSeparatorRow reports whether a line is the row of hyphens under
// a Markdown table header, which datum 0.8 exempts.
func isMarkdownTableSeparatorRow(lineText string) bool {
	trimmedLine := strings.TrimSpace(lineText)
	if !strings.ContainsRune(trimmedLine, '|') || !strings.ContainsRune(trimmedLine, hyphenCharacter) {
		return false
	}

	return strings.Trim(trimmedLine, "|-: \t") == ""
}

// scanGoProseForHyphenRuns tokenises the Go source and checks only its comments
// and string literals. A file that does not tokenise cleanly is still scanned as
// far as the tokeniser reaches; the operating system check reports the syntax
// error itself.
func scanGoProseForHyphenRuns(relativePath string, fileContent []byte) []Violation {
	fileSet := token.NewFileSet()
	sourceFile := fileSet.AddFile(relativePath, fileSet.Base(), len(fileContent))

	var goTokeniser scanner.Scanner

	ignoreTokeniserErrors := func(_ token.Position, _ string) {}
	goTokeniser.Init(sourceFile, fileContent, ignoreTokeniserErrors, scanner.ScanComments)

	var violations []Violation

	for {
		tokenPosition, tokenKind, tokenLiteral := goTokeniser.Scan()
		if tokenKind == token.EOF {
			return violations
		}

		if tokenKind == token.COMMENT || tokenKind == token.STRING {
			startLine := sourceFile.Line(tokenPosition)
			violations = append(violations, scanTextForHyphenRuns(relativePath, tokenLiteral, startLine)...)
		}
	}
}
