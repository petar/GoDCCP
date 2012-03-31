
package main

import (
	"html/template"
)

const (
	htmlHeader = 
		`<!doctype html>` +
		`<!--[if lt IE 7]> <html class="no-js lt-ie9 lt-ie8 lt-ie7" lang="en"> <![endif]-->` +
		`<!--[if IE 7]>    <html class="no-js lt-ie9 lt-ie8" lang="en"> <![endif]-->` +
		`<!--[if IE 8]>    <html class="no-js lt-ie9" lang="en"> <![endif]-->` +
		`<!--[if gt IE 8]><!--> <html class="no-js" lang="en"> <!--<![endif]-->` +
		`<head>` +
			`<meta charset="utf-8">` +
			`<title>DCCP Inspector</title>` +
			`<style>` + css + `</style>` +
			`<script type="text/javascript">` + underscore_js_1_3_1 + `</script>` +
			`<script type="text/javascript">` + jQuery_1_7_2 + `</script>` +
			`<script type="text/javascript">` + headJavaScript + `</script>` +
		`</head>` +
		`<body><table cellspacing="0">`

	htmlFooter = `</table></body></html>`

	css = 
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
		`.ev_good { color: #0c0 }` + 
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
		// Bookmarking
		`td.mark-0 { border-top: 3px solid ` + optMark0Color + ` }` +
		`td.mark-0.time, td.mark-0.file, td.mark-0.line, td.mark-0.sep { background: ` + optMark0Color + `; color: white; }` +
		`td.mark-1 { border-top: 3px solid ` + optMark1Color + ` }` +
		`td.mark-1.time, td.mark-1.file, td.mark-1.line, td.mark-1.sep { background: ` + optMark1Color + `; color: white; }` +
		`td.mark-2 { border-top: 3px solid `+ optMark2Color + ` }` +
		`td.mark-2.time, td.mark-2.file, td.mark-2.line, td.mark-2.sep { background: ` + optMark2Color + `; color: white; }` +
		// Tooltips
		`div.tooltip { position: absolute; float:left; pointer-events: none; margin-top: 10px;` +
			`padding: 7px; background: black; color: white; opacity: 0.7; box-shadow: 1px 1px 3px 0px #000; border-radius: 4px; }` +
		// Folding (not working)
		`tr.folded { height: 5px !important }`
	optMark0Color = `#c33`
	optMark1Color = `#3c3`
	optMark2Color = `#33c`

	emitTmplSource =
		`{{ define "emit" }}` +
			`<tr class="emit">` + 
			`{{ $ev := .Event }}{{ $seqno := .SeqNo }}{{ $ackno := .AckNo }}{{ $trace := .Trace }}` +
				`<td class="time ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .Time }}</pre></td>` +
				`{{ with .Client }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="client state ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` + 
						`<pre>{{ .State }}</pre></td>` + 
					`<td class="client left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left  }}</pre></td>` +
					`<td class="client detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre>` + 
						`{{ if $ne }}<div class="tooltip" style="display: none"><pre>{{ $trace }}</pre></div>{{ end }}` + 
					`</td>` +
					`<td class="client right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Pipe }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="pipe left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left }}</pre></td>` +
					`<td class="pipe detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre>` + 
						`{{ if $ne }}<div class="tooltip" style="display: none"><pre>{{ $trace }}</pre></div>{{ end }}` + 
					`</td>` + 
					`<td class="pipe right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
				`{{ end }}` +
				`{{ with .Server }}` +
					`{{ $n0 := or .State .Left .Right .Detail }}` +
					`{{ $ne := and $n0 "nonempty" }}` +
					`<td class="server left ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Left }}</pre></td>` +
					`<td class="server detail ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Detail }}</pre>` +
						`{{ if $ne }}<div class="tooltip" style="display: none"><pre>{{ $trace }}</pre></div>{{ end }}` + 
					`</td>` +
					`<td class="server right ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .Right }}</pre></td>` +
					`<td class="server state ev_{{ $ev }} {{ $ne }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}">` +
						`<pre>{{ .State }}</pre></td>` + 
				`{{ end }}` +
				`<td class="file ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .SourceFile }}</pre></td>` +
				`<td class="sep ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>:</pre></td>` +
				`<td class="line ev_{{ $ev }}" seqno="{{ $seqno }}" ackno="{{ $ackno }}"><pre>{{ .SourceLine }}</pre></td>` +
			`</tr>` +
		`{{ end }}`
)

var (
	emitTmpl = template.Must(template.New("emit").Parse(emitTmplSource))
)
