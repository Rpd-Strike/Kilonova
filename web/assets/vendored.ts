import hljs from "highlight.js/lib/common";
import CopyButtonPlugin from "highlightjs-copy";
hljs.addPlugin(new CopyButtonPlugin());

import CodeMirror from "codemirror";
window.hljs = hljs;
window.CodeMirror = CodeMirror;

import "codemirror/addon/mode/overlay";
import "codemirror/mode/meta";
import "codemirror/mode/markdown/markdown";
import "codemirror/mode/gfm/gfm";
import "codemirror/mode/clike/clike";
import "codemirror/mode/go/go";
import "codemirror/mode/python/python";
import "codemirror/mode/haskell/haskell";

import katex from "katex";
import renderMathInElement from "katex/contrib/auto-render";

window.katex = katex;

document.addEventListener("DOMContentLoaded", () => {
	renderMathInElement(document.body, {
		delimiters: [
			{ left: "$$", right: "$$", display: true },
			{ left: "$", right: "$", display: false },
			{ left: "\\(", right: "\\)", display: false },
			{ left: "\\[", right: "\\]", display: true },
		],
	});
});