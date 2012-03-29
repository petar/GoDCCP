package main

const (
	headJavaScript =
	`
	jQuery(document).ready(function(){
		$('td[seqno].nonempty').click(onLeftClick);
		$('tr').mouseenter(hilightRow);
		//$('tr').mouseleave(deHilightRow);
		//$('td[ackno].nonempty').bind("contextmenu", onRightClick);
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
	`
)
