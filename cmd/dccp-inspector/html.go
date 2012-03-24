package main

import (
	"bytes"
	"html/template"
	"github.com/petar/GoDCCP/dccp"
)

type HTMLRecord struct {
	Log  *dccp.LogRecord
	HTML string
}

// htmlizeEmit converts a log record into an HTMLRecord
func htmlizeEmit(t *dccp.LogRecord) *HTMLRecord {
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
	pipe.Time = ?
	return &HTMLRecord{ Log: t, HTML: htmlizePipe(pipe) }
}

type emitPipe struct {
	Time         string
	ClientState  string
	ClientDetail string
	PipeDetail   string
	ServerState  string
	ServerDetail string
}

var (
	emitTmpl = template.Must(template.New("emit").Parse(
		`{{ define "emit" }}` +
			`<div class="emit">` + 
				`<div class="time"><pre>{{ .Time }}</pre></div>` +
				`<div class="server">` + 
					`<div class="state"><pre>{{ .ClientState }}</pre></div>` + 
					`<div class="detail"><pre>{{ .ClientDetail }}</pre></div>` + 
				`</div>` +
				`<div class="pipe">` +
					`<div class="detail"><pre>{{ .PipeDetail }}</pre></div>` + 
				`</div>` +
				`<div class="client">` +
					`<div class="detail"><pre>{{ .ServerDetail }}</pre></div>` + 
					`<div class="state"><pre>{{ .ServerState }}</pre></div>` + 
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
	switch r.Module {
	case "server":
		return &emitData{
			ClientState : "",
			ClientDetail: "",
			PipeDetail  : "",
			ServerState : r.State,
			ServerDetail: htmlizePacket(r) + "<——W",
		}
	case "client":
		return &emitData{
			ClientState : r.State,
			ClientDetail: "W——>" + htmlizePacket(r),
			PipeDetail  : "",
			ServerState : "",
			ServerDetail: "",
		}
	}
	panic("unreach")
}

func pipeRead(r *dccp.LogRecord) *emitPipe {
	switch r.Module {
	case "client":
		return &emitData{
			ClientState : r.State,
			ClientDetail: "R<——" + htmlizePacket(r),
			PipeDetail  : "",
			ServerState : "",
			ServerDetail: "",
		}
	case "server":
		return &emitData{
			ClientState : "",
			ClientDetail: "",
			PipeDetail  : "",
			ServerState : r.State,
			ServerDetail: htmlizePacket(r) + "——>R",
		}
	}
	panic("unreach")
}

func pipeDrop(r *dccp.LogRecord) *emitPipe {
	switch r.Module {
	case "line":
		return &emitData{
			ClientState : r.State,
			ClientDetail: "R<——" + htmlizePacket(r),
			PipeDetail  : "",
			ServerState : "",
			ServerDetail: "",
		}
		if r.Module == "server" {
			text = fmt.Sprintf("%s|%s| D<——%s     |%s|%s",
				skipState, skip, sprintPacket(r), skip, skipState)
		} else {
			text = fmt.Sprintf("%s|%s|     %s——>D |%s|%s",
				skipState, skip, sprintPacket(r), skip, skipState)
		}
	case "client":
		e := &emitData{
			ClientState : r.State,
			PipeDetail  : "",
			ServerState : "",
			ServerDetail: "",
		}
		switch r.Comment {
		case "Slow app":
			e.ClientDetail = htmlizePacket(r) + "D<——"
		case "Slow strobe":
			e.ClientDetail = htmlizePacket(r) + "——>D"
		}
		return e
	case "server":
		e := &emitData{
			ClientState : "",
			ClientDetail: "",
			PipeDetail  : "",
			ServerState : r.State,
		}
		switch r.Comment {
		case "Slow strobe":
			e.ServerDetail = "D<——" + htmlizePacket(r) + "    "
		case "Slow app":
			e.ServerDetail = "    " + htmlizePacket(r) + "——>D"
		}
		return e
	}
	panic("unreach")
}
