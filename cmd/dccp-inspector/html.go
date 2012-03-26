package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strconv"
	"strings"
	"github.com/petar/GoDCCP/dccp"
)

type logPipe struct {
	Log  *dccp.LogRecord
	Pipe *emitPipe
}

// htmlize orders a slice of logPipes by time and prints them to standard output with the
// surrounding HTML boilerplate
func htmlize(records []*logPipe, srt bool) {
	if srt {
		sort.Sort(logPipeTimeSort(records))
	}
	fmt.Println(htmlHeader)
	var last int64
	var sec  int64
	var sflag rune = ' '
	for _, r := range records {
		r.Pipe.Time = fmt.Sprintf("%15s %c", dccp.Nstoa(r.Log.Time - last), sflag)
		sflag = ' '
		last = r.Log.Time
		if last / 1e9 > sec {
			sflag = '*'
			sec = last / 1e9
		}
		fmt.Println(htmlizePipe(r.Pipe))
	}
	fmt.Println(htmlFooter)
}

// pipeEmit converts a log record into an HTMLRecord.
// The Time field of the 
func pipeEmit(t *dccp.LogRecord) *logPipe {
	var pipe *emitPipe
	switch t.Event {
	case "Write":
		pipe = pipeWrite(t)
	case "Read":
		pipe = pipeRead(t)
	case "Drop":
		pipe = pipeDrop(t)
	case "Idle":
		pipe = pipeIdle(t)
	default:
		pipe = pipeGeneric(t)
	}
	if pipe == nil {
		fmt.Fprintf(os.Stderr, "Dropping: %v\n", t)
		return nil
	}
	pipe.SourceFile = t.SourceFile
	pipe.SourceLine = strconv.Itoa(t.SourceLine)
	pipe.Event = strings.ToLower(t.Event)
	return &logPipe{ Log: t, Pipe: pipe }
}

type emitPipe struct {
	Time       string
	SourceFile string
	SourceLine string
	Event      string
	Client     emitSubPipe
	Pipe       emitSubPipe
	Server     emitSubPipe
}

type emitSubPipe struct {
	State       string
	Detail      string
	Left, Right string
}

var (
	htmlHeader = 
		`<!doctype html>` +
		`<!--[if lt IE 7]> <html class="no-js lt-ie9 lt-ie8 lt-ie7" lang="en"> <![endif]-->` +
		`<!--[if IE 7]>    <html class="no-js lt-ie9 lt-ie8" lang="en"> <![endif]-->` +
		`<!--[if IE 8]>    <html class="no-js lt-ie9" lang="en"> <![endif]-->` +
		`<!--[if gt IE 8]><!--> <html class="no-js" lang="en"> <!--<![endif]-->` +
		`<head>` +
			`<meta charset="utf-8">` +
			`<title>DCCP Inspector</title>` +
			`<style>` +
			`table, tr, td { font-family: monospace; }` +
			`td { background: #fcfcfc; }` +
			`td { margin: 0; padding:0px; }` +
			`td.time { width: 100px; text-align: right }` +
			`td.state.client { text-align: right }` +
			`td.state.server { text-align: left }` +
			`td.detail { width: 200px }` +
			`td.left { text-align: right; width: 60px !important; }` +
			`td.right { text-align: left; width: 60px !important; }` +
			`td.file { width: 250px; text-align: right }` +
			`td.sep { width: 10px; text-align: center }` +
			`td.line { width: 30px; text-align: left }` +
			`td.nonempty { background: #f0f0f0 }` +
			`pre { padding: 0; margin: 0 }` +
			// Event coloring
			`.ev_warn { color: #c00 }` + 
			`.ev_idle.nonempty { background: #cec }` + 
			`</style>` +
			`<!-- script src="js/libs/modernizr-2.5.3.min.js"></script-->` +
		`</head>` +
		`<body><table cell-spacing="2px">`
	htmlFooter = 
		`</table></body></html>`
	emitTmpl = template.Must(template.New("emit").Parse(
		`{{ define "emit" }}` +
			`<tr class="emit">` + 
				`{{ $ev := .Event }}` +
				`<td class="time ev_{{ $ev }} "><pre>{{ .Time }}</pre></td>` +
				`{{ with .Client }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="client state ev_{{ $ev }} {{ $ne }}"><pre>{{ .State }}</pre></td>` + 
					`<td class="client left ev_{{ $ev }} {{ $ne }}"><pre>{{ .Left  }}</pre></td>` +
					`<td class="client detail ev_{{ $ev }} {{ $ne }}"><pre>{{ .Detail }}</pre></td>` +
					`<td class="client right ev_{{ $ev }} {{ $ne }}"><pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Pipe }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="pipe left ev_{{ $ev }} {{ $ne }}"><pre>{{ .Left }}</pre></td>` +
					`<td class="pipe detail ev_{{ $ev }} {{ $ne }}"><pre>{{ .Detail }}</pre></td>` + 
					`<td class="pipe right ev_{{ $ev }} {{ $ne }}"><pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Server }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="server left ev_{{ $ev }} {{ $ne }}"><pre>{{ .Left }}</pre></td>` +
					`<td class="server detail ev_{{ $ev }} {{ $ne }}"><pre>{{ .Detail }}</pre></td>` +
					`<td class="server right ev_{{ $ev }} {{ $ne }}"><pre>{{ .Right }}</pre></td>` +
					`<td class="server state ev_{{ $ev }} {{ $ne }}"><pre>{{ .State }}</pre></td>` + 
				`{{ end }}` +
				`<td class="file ev_{{ $ev }} "><pre>{{ .SourceFile }}</pre></td>` +
				`<td class="sep ev_{{ $ev }} "><pre>:</pre></td>` +
				`<td class="line ev_{{ $ev }} "><pre>{{ .SourceLine }}</pre></td>` +
			`</tr>` +
		`{{ end }}`,
	))
)

func htmlizePipe(e *emitPipe) string {
	var w bytes.Buffer
	if err := emitTmpl.Execute(&w, e); err != nil {
		fmt.Fprintf(os.Stderr, "template error (%s)\n", err)
		panic("error htmlizing emit")
	}
	return string(w.Bytes())
}

func pipeWrite(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "server":
		return &emitPipe{
			Server: emitSubPipe{ 
				State:  r.State,
				Detail: sprintPacket(r),
				Right:  "<——W",
			},
		}
	case "client":
		return &emitPipe{
			Client: emitSubPipe{
				State : r.State,
				Detail: sprintPacket(r),
				Left:   "W——>",
			},
		}
	}
	return nil
}

func pipeRead(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "client":
		return &emitPipe{
			Client: emitSubPipe {
				State:  r.State,
				Detail: sprintPacket(r),
				Left:   "R<——",
			},
		}
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: sprintPacket(r),
				Right:  "——>R",
			},
		}
	}
	return nil
}

func pipeIdle(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "client":
		return &emitPipe{
			Client: emitSubPipe {
				State:  r.State,
				Detail: "",
			},
		}
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: "",
			},
		}
	}
	return nil
}

func pipeDrop(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "line":
		switch r.Module {
		case "server":
			return &emitPipe{
				Pipe: emitSubPipe{
					Detail: sprintPacket(r),
					Left:   "D<——",
				},
			}
		case "client":
			return &emitPipe{
				Pipe: emitSubPipe{
					Detail: sprintPacket(r),
					Right:  "——>D",
				},
			}
		}
	case "client":
		switch r.Comment {
		case "Slow app":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacket(r),
					Right:  "D<——",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacket(r),
					Right:  "——>D",
				},
			}
		}
	case "server":
		switch r.Comment {
		case "Slow app":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacket(r),
					Left:  "——>D",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacket(r),
					Left:  "D<——",
				},
			}
		}
	}
	return nil
}

func pipeGeneric(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "line":
		return &emitPipe{
			Pipe: emitSubPipe {
				Detail: sprintPacketEventComment(r),
			},
		}
	case "client":
		return &emitPipe{
			Client: emitSubPipe {
				State:  r.State,
				Detail: sprintPacketEventComment(r),
			},
		}
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: sprintPacketEventComment(r),
			},
		}
	}
	return nil
}

// logPipeTimeSort sorts logPipe records by timestamp
type logPipeTimeSort []*logPipe

func (t logPipeTimeSort) Len() int {
	return len(t)
}

func (t logPipeTimeSort) Less(i, j int) bool {
	return t[i].Log.Time < t[j].Log.Time
}

func (t logPipeTimeSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
