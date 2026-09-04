// Package vagrant parses Vagrant box declarations without evaluating Ruby.
package vagrant

import (
	"bytes"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

const boxLocation = "config.vm.box"

func init() {
	core.Register("vagrant", core.Manifest, &vagrantfileParser{}, core.ExactMatch("Vagrantfile"))
}

type vagrantfileParser struct{}

type boxFields struct {
	name         string
	version      string
	url          string
	checksum     string
	checksumType string
}

func (p *vagrantfileParser) Parse(_ string, content []byte) (*core.Result, error) {
	var fields boxFields
	var ignored rubyIgnoredRegion
	for len(content) > 0 {
		line := content
		if newline := bytes.IndexByte(content, '\n'); newline >= 0 {
			line = content[:newline]
			content = content[newline+1:]
		} else {
			content = nil
		}
		if ignored.skipLine(line) {
			continue
		}
		field, value, ok := parseBoxAssignment(line)
		if ok {
			fields.set(field, value)
		}
		ignored.startHeredoc(line)
	}
	return fields.result(), nil
}

type rubyIgnoredRegion struct {
	blockComment bool
	heredoc      string
}

func (region *rubyIgnoredRegion) skipLine(line []byte) bool {
	trimmed := bytes.TrimSpace(line)
	if region.blockComment {
		if hasRubyBlockCommentMarker(trimmed, "=end") {
			region.blockComment = false
		}
		return true
	}
	if region.heredoc != "" {
		if string(trimmed) == region.heredoc {
			region.heredoc = ""
		}
		return true
	}
	if hasRubyBlockCommentMarker(trimmed, "=begin") {
		region.blockComment = true
		return true
	}
	return false
}

func (region *rubyIgnoredRegion) startHeredoc(line []byte) {
	region.heredoc = rubyHeredocDelimiter(line)
}

func hasRubyBlockCommentMarker(line []byte, marker string) bool {
	if !bytes.HasPrefix(line, []byte(marker)) {
		return false
	}
	return len(line) == len(marker) || isRubySpace(line[len(marker)])
}

func rubyHeredocDelimiter(line []byte) string {
	var quote byte
	escaped := false
	for position := 0; position < len(line); position++ {
		character := line[position]
		if quote != 0 {
			switch {
			case escaped:
				escaped = false
			case character == '\\':
				escaped = true
			case character == quote:
				quote = 0
			}
			continue
		}
		switch character {
		case '\'', '"':
			quote = character
		case '#':
			return ""
		case '<':
			if delimiter, ok := parseRubyHeredocOpener(line, position); ok {
				return delimiter
			}
		}
	}
	return ""
}

func parseRubyHeredocOpener(line []byte, position int) (string, bool) {
	if position+2 >= len(line) || line[position+1] != '<' {
		return "", false
	}
	position += 2
	if line[position] == '-' || line[position] == '~' {
		position++
	}
	if position >= len(line) {
		return "", false
	}
	if line[position] == '\'' || line[position] == '"' {
		quote := line[position]
		position++
		start := position
		for position < len(line) && line[position] != quote {
			position++
		}
		return string(line[start:position]), position > start && position < len(line)
	}
	start := position
	for position < len(line) && isRubyIdentifierByte(line[position]) {
		position++
	}
	return string(line[start:position]), position > start
}

func (fields *boxFields) set(field, value string) {
	switch field {
	case "box":
		fields.name = value
	case "box_version":
		fields.version = value
	case "box_url":
		fields.url = value
	case "box_download_checksum":
		fields.checksum = value
	case "box_download_checksum_type":
		fields.checksumType = value
	}
}

func (fields boxFields) result() *core.Result {
	if fields.name == "" {
		return &core.Result{}
	}
	source := core.Source{}
	if fields.url != "" {
		source = core.Source{Kind: core.SourceURL, Value: fields.url}
	}
	integrity := ""
	if fields.checksumType != "" && fields.checksum != "" {
		integrity = fields.checksumType + "-" + fields.checksum
	}
	dependency := core.Dependency{
		Name:      fields.name,
		Version:   fields.version,
		Scope:     core.Runtime,
		Integrity: integrity,
		Direct:    true,
		Source:    source,
	}
	declaration := core.Declaration{
		Name:     fields.name,
		Version:  fields.version,
		Scope:    core.Runtime,
		Direct:   true,
		Location: boxLocation,
		Source:   source,
	}
	return &core.Result{
		Dependencies: []core.Dependency{dependency},
		Declarations: []core.Declaration{declaration},
	}
}

func parseBoxAssignment(line []byte) (string, string, bool) {
	position := 0
	skipRubySpace(line, &position)
	if !consumeToken(line, &position, "config") || !consumeDot(line, &position) ||
		!consumeToken(line, &position, "vm") || !consumeDot(line, &position) {
		return "", "", false
	}
	fieldStart := position
	for position < len(line) && isRubyIdentifierByte(line[position]) {
		position++
	}
	field := string(line[fieldStart:position])
	if !isBoxField(field) {
		return "", "", false
	}
	skipRubySpace(line, &position)
	if position >= len(line) || line[position] != '=' {
		return "", "", false
	}
	position++
	skipRubySpace(line, &position)
	value, ok := parseRubyString(line, &position)
	if !ok {
		return "", "", false
	}
	skipRubySpace(line, &position)
	if position < len(line) && line[position] != '#' {
		return "", "", false
	}
	return field, value, true
}

func consumeToken(line []byte, position *int, token string) bool {
	skipRubySpace(line, position)
	if !bytes.HasPrefix(line[*position:], []byte(token)) {
		return false
	}
	*position += len(token)
	return true
}

func consumeDot(line []byte, position *int) bool {
	skipRubySpace(line, position)
	if *position >= len(line) || line[*position] != '.' {
		return false
	}
	*position++
	skipRubySpace(line, position)
	return true
}

func isBoxField(field string) bool {
	switch field {
	case "box", "box_version", "box_url", "box_download_checksum", "box_download_checksum_type":
		return true
	default:
		return false
	}
}

func parseRubyString(line []byte, position *int) (string, bool) {
	if *position >= len(line) || line[*position] != '\'' && line[*position] != '"' {
		return "", false
	}
	quote := line[*position]
	*position++
	var value strings.Builder
	for *position < len(line) {
		character := line[*position]
		*position++
		switch character {
		case quote:
			return value.String(), true
		case '\\':
			if !appendRubyEscape(line, position, quote, &value) {
				return "", false
			}
		case '#':
			if quote == '"' && startsRubyInterpolation(line[*position:]) {
				return "", false
			}
			value.WriteByte(character)
		default:
			value.WriteByte(character)
		}
	}
	return "", false
}

func appendRubyEscape(line []byte, position *int, quote byte, value *strings.Builder) bool {
	if *position >= len(line) {
		return false
	}
	next := line[*position]
	*position++
	if quote == '\'' && next != '\'' && next != '\\' {
		value.WriteByte('\\')
		value.WriteByte(next)
		return true
	}
	switch next {
	case 'n':
		value.WriteByte('\n')
	case 'r':
		value.WriteByte('\r')
	case 't':
		value.WriteByte('\t')
	default:
		value.WriteByte(next)
	}
	return true
}

func startsRubyInterpolation(value []byte) bool {
	return len(value) > 0 && (value[0] == '{' || value[0] == '$' || value[0] == '@')
}

func skipRubySpace(line []byte, position *int) {
	for *position < len(line) && isRubySpace(line[*position]) {
		*position++
	}
}

func isRubySpace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\r'
}

func isRubyIdentifierByte(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' || character == '_'
}
