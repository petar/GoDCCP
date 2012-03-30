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
	subSpacer := emitSubPipe{
		State:  "",
		Detail: "",
		Left:   "    ",
		Right:  "    ",
	}
	spacer := &emitPipe{
		Client: subSpacer,
		Pipe:   subSpacer,
		Server: subSpacer,
		Event:  "spacer",
	}
	fmt.Println(htmlizePipe(spacer))
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
	pipe.Event = classify(t.Event)
	pipe.SeqNo = htmlizeAckSeqNo(t.Type, t.SeqNo)
	pipe.AckNo = htmlizeAckSeqNo(t.Type, t.AckNo)
	return &logPipe{ Log: t, Pipe: pipe }
}

func htmlizeAckSeqNo(t string, no int64) string {
	if t == "" {
		return ""
	}
	return fmt.Sprintf("%06x", no)
}

func classify(ev string) string {
	var w bytes.Buffer
	for _, b := range ev {
		switch b {
		case '-':
			w.WriteByte('_')
		default:
			w.WriteByte(byte(b))
		}
	}
	return strings.ToLower(string(w.Bytes()))
}

type emitPipe struct {
	Time       string
	SourceFile string
	SourceLine string
	Event      string
	SeqNo      string
	AckNo      string
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
			`*, body, table, tr, td, p, div, pre { cursor: default }` +
			`table, tr, td { font-family: 'Droid Sans Mono'; font-size: 12px; }` +
			`td { background: #fdfdfd; }` +
			`td { margin: 0; padding:1px; }` +
			`td { border-top: 1px dotted #ccc; }` +
			`td.time { background: #fafafa; width: 100px; text-align: right }` +
			`td.file, td.sep, td.line { background: #fafafa; }` +
			`td.state.client { text-align: right }` +
			`td.state.server { text-align: left }` +
			`td.detail { width: 200px }` +
			`td.left { text-align: right; padding-left: 3px; padding-right: 3px }` +
			`td.right { text-align: left; padding-left: 3px; padding-right: 3px }` +
			`td.file { width: 250px; text-align: right }` +
			`td.sep { width: 10px; text-align: center }` +
			`td.line { width: 30px; text-align: left }` +
			`td.client.nonempty { background: #fff0f0 }` +
			`td.server.nonempty { background: #f0f0ff }` +
			`td.pipe.nonempty { background: #f8e0f8 }` +
			`pre { padding: 0; margin: 0 }` +
			// Event coloring
			`.ev_warn { color: #c00 }` + 
			`.client.ev_idle.nonempty { background: #f8e0e0 }` + 
			`.server.ev_idle.nonempty { background: #e0e0f8 }` + 
			`.ev_event, .ev_idle, .ev_rrtt_h, .ev_rrtt, .ev_wccval, .ev_info { color: #aaa }` + 
			`.ev_write, .ev_read, .ev_drop { color: #000 }` + 
			`.ev_end { background: #0c0 !important; color: #fff !important }` + 
			`.ev_spacer { background: white !important; color: white !important }` + 
			// Highlight coloring
			`.hi-bkg { font-size: 12px; padding-top: 9px; padding-bottom: 9px; }` +
			`.orange-bkg { background: orange !important }` +
			`.yellow-bkg { background: yellow !important }` +
			`.red-bkg { background: red !important }` +
			`td.mark-bkg { border-top: 3px solid #666 }` +
			`td.mark-bkg.time, td.mark-bkg.file, td.mark-bkg.line, td.mark-bkg.sep { background: #666; color: white; }` +
			// Folding
			`tr.folded { height: 5px !important }` +
			`</style>` +
			`<script type="text/javascript">` + underscore_js_1_3_1 + `</script>` +
			`<script type="text/javascript">` + jQuery_1_7_2 + `</script>` +
			`<script type="text/javascript">` + headJavaScript + `</script>` +
		`</head>` +
		`<body><table cellspacing="0">`
	htmlFooter = 
		`</table></body></html>`
	emitTmpl = template.Must(template.New("emit").Parse(
		`{{ define "emit" }}` +
			`<tr class="emit">` + 
			`{{ $ev := .Event }}{{ $seqno := .SeqNo }}{{ $ackno := .AckNo }}` +
				`<td class="time ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .Time }}</pre></td>` +
				`{{ with .Client }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="client state ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` + 
						`<pre>{{ .State }}</pre></td>` + 
					`<td class="client left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left  }}</pre></td>` +
					`<td class="client detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre></td>` +
					`<td class="client right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Pipe }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="pipe left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left }}</pre></td>` +
					`<td class="pipe detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre></td>` + 
					`<td class="pipe right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Server }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="server left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left }}</pre></td>` +
					`<td class="server detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre></td>` +
					`<td class="server right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
					`<td class="server state ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .State }}</pre></td>` + 
				`{{ end }}` +
				`<td class="file ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .SourceFile }}</pre></td>` +
				`<td class="sep ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>:</pre></td>` +
				`<td class="line ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .SourceLine }}</pre></td>` +
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

const htmlPacketWidth = 17

func pipeWrite(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "server":
		return &emitPipe{
			Server: emitSubPipe{ 
				State:  r.State,
				Detail: sprintPacketWidth(r, htmlPacketWidth),
				Right:  "<——W",
			},
		}
	case "line":
		e := &emitPipe{
			Pipe: emitSubPipe{
				Detail: sprintPacketWidth(r, htmlPacketWidth),
			},
		}
		switch r.Module {
		case "client":
			e.Pipe.Left = "W——>"
		case "server":
			e.Pipe.Right = "<——W"
		}
		return e
	case "client":
		return &emitPipe{
			Client: emitSubPipe{
				State : r.State,
				Detail: sprintPacketWidth(r, htmlPacketWidth),
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
				Detail: sprintPacketWidth(r, htmlPacketWidth),
				Left:   "R<——",
			},
		}
	case "line":
		e := &emitPipe{
			Pipe: emitSubPipe{
				Detail: sprintPacketWidth(r, htmlPacketWidth),
			},
		}
		switch r.Module {
		case "client":
			e.Pipe.Left = "R<——"
		case "server":
			e.Pipe.Right = "——>R"
		}
		return e
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: sprintPacketWidth(r, htmlPacketWidth),
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
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:   "D<——",
				},
			}
		case "client":
			return &emitPipe{
				Pipe: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "——>D",
				},
			}
		}
	case "client":
		switch r.Comment {
		case "Slow app":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "D<——",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "——>D",
				},
			}
		}
	case "server":
		switch r.Comment {
		case "Slow app":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:  "——>D",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:  "D<——",
				},
			}
		}
	}
	return nil
}

const htmlEventWidth = 30 

func sprintPacketEventCommentHTML(r *dccp.LogRecord) string {
	if r.SeqNo == 0 {
		return fmt.Sprintf(" %s ", cut(r.Comment, htmlEventWidth-2))
	}
	return fmt.Sprintf(" %s %06x·%06x ", cut(r.Comment, htmlEventWidth-14-2), r.SeqNo, r.AckNo)
}

func pipeGeneric(r *dccp.LogRecord) *emitPipe {
	switch r.System {
	case "line":
		return &emitPipe{
			Pipe: emitSubPipe {
				Detail: sprintPacketEventCommentHTML(r),
			},
		}
	case "client":
		return &emitPipe{
			Client: emitSubPipe {
				State:  r.State,
				Detail: sprintPacketEventCommentHTML(r),
			},
		}
	case "server":
		return &emitPipe{
			Server: emitSubPipe{
				State:  r.State,
				Detail: sprintPacketEventCommentHTML(r),
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
