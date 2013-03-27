package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"github.com/petar/GoDCCP/dccp"
)

type emitPipe struct {
	Time       string
	TimeAbs    string
	SourceFile string
	SourceLine string
	Trace      string
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

type logPipe struct {
	Log  *dccp.Trace
	Pipe *emitPipe
}

// htmlize orders a slice of logPipes by time and prints them to standard output with the
// surrounding HTML boilerplate
func htmlize(records []*logPipe, srt bool, includeEmits bool) {
	if srt {
		sort.Sort(logPipeTimeSort(records))
	}
	fmt.Println(htmlHeader)

	var last int64
	var sec  int64
	var sflag rune = ' '
	var series SeriesSweeper
	series.Init()
	for _, r := range records {
		series.Add(r.Log)
		r.Pipe.Time = fmt.Sprintf("%15s %c", dccp.Nstoa(r.Log.Time - last), sflag)
		r.Pipe.TimeAbs = fmt.Sprintf("%c %-15s", sflag, dccp.Nstoa(r.Log.Time))
		sflag = ' '
		last = r.Log.Time
		if last / 1e9 > sec {
			sflag = '*'
			sec = last / 1e9
		}
		if includeEmits {
			fmt.Println(htmlizePipe(r.Pipe))
		}
	}

	if includeEmits {
		// TODO: The spacer row is a hack to prevent TDs from collapsing their width
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
	}

	fmt.Println(htmlFooterPreSeries)
	printGraphJavaScript(os.Stdout, &series)
	fmt.Println(htmlFooterPostSeries)
}

// pipeEmit converts a log record into an HTMLRecord.
// The Time field of the 
func pipeEmit(t *dccp.Trace) *logPipe {
	var pipe *emitPipe
	switch t.Event {
	case dccp.EventWrite:
		pipe = pipeWrite(t)
	case dccp.EventRead:
		pipe = pipeRead(t)
	case dccp.EventDrop:
		pipe = pipeDrop(t)
	case dccp.EventIdle:
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
	if t.Type == "" {
		pipe.Trace = t.Trace
	} else {
		pipe.Trace = sprintPacketWide(t) + "\n" + t.Trace
	}
	return &logPipe{ Log: t, Pipe: pipe }
}

func htmlizeAckSeqNo(t string, no int64) string {
	if t == "" {
		return ""
	}
	return fmt.Sprintf("%06x", no)
}

func classify(ev dccp.Event) string {
	var w bytes.Buffer
	for _, b := range ev.String() {
		switch b {
		case '-':
			w.WriteByte('_')
		default:
			w.WriteByte(byte(b))
		}
	}
	return strings.ToLower(string(w.Bytes()))
}

func htmlizePipe(e *emitPipe) string {
	var w bytes.Buffer
	if err := emitTmpl.Execute(&w, e); err != nil {
		fmt.Fprintf(os.Stderr, "template error (%s)\n", err)
		panic("error htmlizing emit")
	}
	return string(w.Bytes())
}

const htmlPacketWidth = 21 

func pipeWrite(r *dccp.Trace) *emitPipe {
	switch r.Labels[0] {
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
		switch r.Labels[1] {
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

func pipeRead(r *dccp.Trace) *emitPipe {
	switch r.Labels[0] {
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
		switch r.Labels[1] {
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

func pipeIdle(r *dccp.Trace) *emitPipe {
	switch r.Labels[0] {
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

func pipeDrop(r *dccp.Trace) *emitPipe {
	switch r.Labels[0] {
	case "line":
		switch r.Labels[1] {
		case "server":
			return &emitPipe{
				Pipe: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:   "X<——",
				},
			}
		case "client":
			return &emitPipe{
				Pipe: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "——>X",
				},
			}
		default:
			panic("line drop without side")
		}
	case "client":
		switch r.Comment {
		case "Slow app", "Bad header":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "X<——",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "——>X",
				},
			}
		default:
			return &emitPipe{
				Client: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Right:  "—XX—",
				},
			}
		}
	case "server":
		switch r.Comment {
		case "Slow app", "Bad header":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:  "——>X",
				},
			}
		case "Slow strobe":
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:  "X<——",
				},
			}
		default:
			return &emitPipe{
				Server: emitSubPipe{
					Detail: sprintPacketWidth(r, htmlPacketWidth),
					Left:  "—XX—",
				},
			}
		}
	}
	return nil
}

const htmlEventWidth = 41

func sprintPacketEventCommentHTML(r *dccp.Trace) string {
	if r.Type == "" {
		return fmt.Sprintf("   %s ", cut(r.Comment, htmlEventWidth-4))
	}
	return fmt.Sprintf(" ¶ %s ", cut(r.Comment, htmlEventWidth-4))
}

func pipeGeneric(r *dccp.Trace) *emitPipe {
	switch r.Labels[0] {
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
