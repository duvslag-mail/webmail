package icons

import (
	"embed"

	"github.com/a-h/templ"
)

//go:embed svg/*.svg
var files embed.FS

func Icon(name string) templ.Component {
	svg, err := files.ReadFile("svg/" + name + ".svg")
	if err != nil {
		return templ.Raw("<svg></svg>")
	}
	return templ.Raw(string(svg))
}
