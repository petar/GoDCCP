package main

/*
  UI behavior:
	o Clicking on a row whose log entry pertains to a packet:
		(i)   Highlights in red the cell that was originally clicked
		(ii)  Highlights in yellow all other rows pertaining to the same packet (same SeqNo)
		(iii) Highlights in orange all other rows pertaining to packets acknowledging this
		packet (AckNo is same as SeqNo of original packet)
		(iv)  Removes highlighting on rows that were previously highlighted using this
		procedure
	o Clicking the (left-most) time cell of a row toggles a dark frame around it
	o Hovering over any row zooms on the row
 */

const (
	headJavaScript =
	`
	jQuery(document).ready(function(){
		$('td[seqno].nonempty').click(onLeftClick);
		$('tr').mouseenter(hilightRow);
		$('td.time').click(toggleMarkRow);
	})
	function onLeftClick(e) {
		var seqno = $(this).attr("seqno");
		if (_.isUndefined(seqno) || seqno == "") {
			return;
		}
		clearEmphasis();
		_.each($('[seqno='+seqno+'].nonempty'), function(t) { emphasize(t, "yellow-bkg") });
		_.each($('[ackno='+seqno+'].nonempty'), function(t) { emphasize(t, "orange-bkg") });
		emphasize($(this), "red-bkg");
	}
	function emphasize(t, bkg) {
		t = $(t);
		var saved_bkg = t.attr("emph");
		if (!_.isUndefined(saved_bkg)) {
			t.removeClass(saved_bkg);
		}
		t.addClass(bkg);
		t.attr("emph", bkg);
	}
	function clearEmphasis() {
		_.each($('[emph]'), function(t) {
			t = $(t);
			var saved_bkg = t.attr("emph");
			t.removeAttr("emph");
			t.removeClass(saved_bkg);
		});
	}
	function hilightRow() {
		_.each($('[hi]'), deHilightRow);
		$(this).attr("hi", 1);
		$('td', this).addClass("hi-bkg");
	}
	function deHilightRow(t) {
		t = $(t);
		$('td', t).removeClass("hi-bkg");
		t.removeAttr("hi");
	}
	function toggleMarkRow() {
		var trow = $(this).parents()[0];
		_.each($('td', trow), function(t) {
			t = $(t);
			if (t.hasClass("mark-bkg")) {
				t.removeClass("mark-bkg");
			} else {
				t.addClass("mark-bkg");
			}
		});
	}
	`
)
