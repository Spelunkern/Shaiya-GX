package text

import "strings"

// Format expands native SysMsgTxt placeholders in a single pass. Substituted
// names and chat bodies are opaque: text containing <p> cannot inject a token.
func (m SysMsg) Format(id int, values map[string]string) string {
	template := m.Get(id)
	var out strings.Builder
	for len(template) > 0 {
		start := strings.IndexByte(template, '<')
		if start < 0 {
			out.WriteString(template)
			break
		}
		out.WriteString(template[:start])
		template = template[start:]
		end := strings.IndexByte(template, '>')
		if end < 0 {
			out.WriteString(template)
			break
		}
		token := template[1:end]
		if value, ok := values[token]; ok {
			out.WriteString(value)
		} else {
			out.WriteString(template[:end+1])
		}
		template = template[end+1:]
	}
	return out.String()
}
