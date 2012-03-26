package main

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"strconv"
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
		return nil
	}
	pipe.SourceFile = t.SourceFile
	pipe.SourceLine = strconv.Itoa(t.SourceLine)
	return &logPipe{ Log: t, Pipe: pipe }
}

type emitPipe struct {
	Time       string
	SourceFile string
	SourceLine string
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
			`<!-- script src="js/libs/modernizr-2.5.3.min.js"></script-->` +
		`</head>` +
		`<body>`
	htmlFooter = 
		`</body>` +
		`</html>`
	emitTmpl = template.Must(template.New("emit").Parse(
		`{{ define "emit" }}` +
			`<div class="emit">` + 
				`<div class="time"><pre>{{ .Time }}</pre></div>` +
				`{{ with .Client }}` +
				`<div class="client">` + 
					`<div class="state"><pre>{{ .State }}</pre></div>` + 
					`<div class="detail">` +
						`<pre class="left">{{ .Left }}</pre>` +
						`<pre>{{ .Detail }}</pre>` +
						`<pre class="left">{{ .Right }}</pre>` +
					`</div>` + 
				`</div>` +
				`{{ end }}` +
				`{{ with .Pipe }}` +
				`<div class="pipe">` +
					`<div class="detail"><pre>{{ .Detail }}</pre></div>` + 
				`</div>` +
				`{{ end }}` +
				`{{ with .Server }}` +
				`<div class="server">` +
					`<div class="detail">` +
						`<pre class="left">{{ .Left }}</pre>` +
						`<pre>{{ .Detail }}</pre>` +
						`<pre class="left">{{ .Right }}</pre>` +
					`</div>` + 
					`<div class="state"><pre>{{ .State }}</pre></div>` + 
				`</div>` +
				`{{ end }}` +
				`<div class="source">` +
					`<pre class="file">{{ .SourceFile }}</pre>` +
					`<pre class="line">{{ .SourceLine }}</pre>` +
				`</div>` +
			`</div>` +
		`{{ end }}`,
	))
)

func htmlizePipe(e *emitPipe) string {
	var w bytes.Buffer
	if emitTmpl.Execute(&w, e) != nil {
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
				Detail: "—————————————————",
			},
		}
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: "—————————————————",
			},
		}
	}
	panic("unreach")
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
	panic("unreach")
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
	panic("unreach")
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
