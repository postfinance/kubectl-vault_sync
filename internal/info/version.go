// Package info creates version information
package info

import (
	"bytes"
	"runtime"
	"strings"
	"text/template"
)

const versionInfoTmpl = `
{{.Program}}, version {{.Version}} (revision: {{.Revision}})
  build date:       {{.BuildDate}}
  go version:       {{.GoVersion}}
`

// Version contains all version information.
type Version struct {
	Program   string
	Version   string
	Revision  string
	BuildDate string
	GoVersion string
}

func (v Version) String() string {
	t := template.Must(template.New("version").Parse(versionInfoTmpl))

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "version", v); err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())
}

// New create a new Version.
func New(program, version, date, commit string) Version {
	revision := "12345678"
	if len(commit) >= 8 {
		revision = commit[:8]
	}

	return Version{
		Program:   program,
		Version:   version,
		Revision:  revision,
		BuildDate: date,
		GoVersion: runtime.Version(),
	}
}
