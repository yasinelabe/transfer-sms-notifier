package processor

import "strings"

type TemplateVars struct {
	TransferID  string
	Sender      string
	Receiver    string
	CreatedDate string
}

func RenderTemplate(tmpl string, vars TemplateVars) string {
	out := tmpl
	repl := map[string]string{
		"{{transferId}}":  vars.TransferID,
		"{{sender}}":      vars.Sender,
		"{{receiver}}":    vars.Receiver,
		"{{createdDate}}": vars.CreatedDate,
	}
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return out
}
