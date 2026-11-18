#!/usr/bin/env node
// Glyph Protocol demo — sample output exercising the protocol's payload
// formats:
//   colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//   colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//   glyf   : five Nerd Font outlines (branch / folder / home / heart /
//            Rust cog) registered at consecutive PUA-B codepoints.
//   vg     : SVG path data and paint as text (proposed, raphamorim/rio#2095):
//            icons, gradients, clips, and Noto and Twemoji emoji.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

import {
	nablaLetters,
	nablaLineHeight,
	nablaText,
	nablaYMin,
} from './nabla_rio.js';
import {
	zupraLetters,
	zupraLineHeight,
	zupraText,
	zupraUpm,
	zupraYMin,
} from './zupra_using.js';
import {twemojiLetters, twemojiText, twemojiUpm} from './twemoji.js';
import {notoFruitLetters, notoFruitText} from './noto_fruit.js';
import {
	VG_EMOJI_NOTO,
	VG_EMOJI_TWEMOJI,
	VG_ICONS,
	VG_PAINTS,
	type VgGlyph,
} from './vg.js';

// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.
const slotCps: ReadonlyArray<number> = [
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
];

// `width=2` slots — the same five outlines re-registered at a fresh
// PUA-B range so the wide row prints alongside the narrow row for a
// direct visual comparison. `width=2` only widens the pixels a glyph
// is drawn into: the codepoint still takes one column (its wcwidth),
// so the printed text leaves a blank cell after each one.
const wideSlotCps: ReadonlyArray<number> = [
	0x100005, 0x100006, 0x100007, 0x100008, 0x100009,
];

// The Nerd Font icons are drawn for the font's own cell: a 600 unit
// advance, ascender 1020 and descender -300 at upm 1000. Declaring
// that extent (§8.5.1) places every icon the way the patched font
// would, and keeps the five in proportion to each other. Without it
// the terminal would size each outline from its own bounds.
const nerdExtent = 'aw=600;lh=1320;ymin=-300';

// The placement rows declare a square em around the heart instead
// (the heart spans x 0..923, y -35..761). A terminal cell is taller
// than it is wide, so a square extent is where the size modes differ.
const placementExtent = 'aw=1000;lh=1000;ymin=-120';

// Placement demo slots — the Nerd Font heart outline registered
// repeatedly at a fresh PUA-B range, each registration carrying
// different §8.5 sizing/placement parameters (`size=`, `align=`,
// `pad=`, `width=`). Printed over a grey background block so the
// cell box — and where the glyph lands inside it — is visible.
const placementSlots: ReadonlyArray<{cp: number; params: string}> = [
	// Size modes (§8.5.3): height scales the extent to the cell's
	// height (may overflow sideways and get clipped), contain is the
	// default and fits it inside, cover fills the cell (may overflow),
	// stretch distorts to fill both axes exactly.
	{cp: 0x100070, params: 'size=height'},
	{cp: 0x100071, params: 'size=contain'},
	// In a cell taller than wide, a square extent makes cover pick the
	// height factor, so this heart renders the same as the height one,
	// while contain fits by width. That's correct: the modes only
	// differ by which axis wins.
	{cp: 0x100072, params: 'size=cover'},
	{cp: 0x100073, params: 'size=stretch'},
	// Alignment (§8.5.4) — where the scaled outline sits in the span.
	{cp: 0x100074, params: 'size=contain;align=start,start'},
	{cp: 0x100075, params: 'size=contain;align=end,end'},
	// `baseline` puts the outline's y=0 on the text baseline, so a
	// character-like glyph sits on the same line as adjacent text.
	{cp: 0x100076, params: 'size=contain;align=center,baseline'},
	// Pad (§8.5.2) insets the render span before scaling.
	{cp: 0x100077, params: 'size=stretch;pad=0.2,0.2,0.2,0.2'},
	// width=2 with contain: one glyph drawn across two cells.
	{cp: 0x100078, params: 'width=2;size=contain'},
];

// Each letter of the colour demos is registered with the font's line
// height and descender, plus its own advance. size=height scales by
// line height alone, so every letter gets the same scale and they all
// sit on one baseline; the advance centres each letter in its cell.
function letterExtent(
	advance: number,
	lineHeight: number,
	ymin: number,
): string {
	return `aw=${advance};lh=${lineHeight};ymin=${ymin};size=height`;
}

// `glyf` simple-glyph bytes, base64-encoded, extracted from
// JetBrainsMono Nerd Font via fontTools. units per em = 1000.
// Source codepoints in the Nerd Font (for reference):
//   branch  U+E0A0
//   folder  U+F07B
//   home    U+F015
//   heart   U+F004
//   rust    U+E7A8
const nerdGlyfsB64: ReadonlyArray<string> = [
	'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',
];

function sendRegisterGlyf(cp: number, payload: string): void {
	// `fmt` is implicit (`glyf`). `upm` defaults to 1000 but we send
	// it explicitly to match the source font's authoring EM. The
	// `reply=0` level tells the terminal to register silently — no
	// `status=0` ACK — which keeps the shell prompt clean on exit
	// (otherwise the queued ACK bytes spill to the next prompt).
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};reply=0;upm=1000;${nerdExtent};${payload}\x1b\\`,
	);
}

function sendRegisterGlyfWide(cp: number, payload: string): void {
	// Same as sendRegisterGlyf but drawn across two cells. The printed
	// text keeps the cell after it blank for the overflow.
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};reply=0;upm=1000;${nerdExtent};width=2;${payload}\x1b\\`,
	);
}

function sendRegisterGlyfPlaced(
	cp: number,
	params: string,
	payload: string,
): void {
	// Same as sendRegisterGlyf plus the registration's §8.5
	// sizing/placement parameters, verbatim from placementSlots.
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};reply=0;upm=1000;${placementExtent};${params};${payload}\x1b\\`,
	);
}

function sendRegisterColrv1(cp: number, payload: string): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv1;reply=2;upm=1000;${payload}\x1b\\`,
	);
}

// Offset for the width=2 duplicates of the Nabla letters: the same
// colrv1 payloads re-registered at cp+nablaWideOffset (0x100020 to
// 0x10002C, free between the Nabla and Zupra ranges) so the second
// colrv1 line renders each letter across two cells.
const nablaWideOffset = 0x10;

function sendRegisterColrv1Letter(
	cp: number,
	extent: string,
	payload: string,
): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv1;reply=2;upm=1000;${extent};${payload}\x1b\\`,
	);
}

function sendRegisterColrv1Wide(
	cp: number,
	extent: string,
	payload: string,
): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv1;reply=2;upm=1000;${extent};width=2;${payload}\x1b\\`,
	);
}

function sendRegisterColrv0(cp: number, upm: number, payload: string): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv0;reply=2;upm=${upm};${payload}\x1b\\`,
	);
}

function sendRegisterColrv0Letter(
	cp: number,
	upm: number,
	extent: string,
	payload: string,
): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv0;reply=2;upm=${upm};${extent};${payload}\x1b\\`,
	);
}

function registerGlyfSlots(): void {
	for (let i = 0; i < slotCps.length; i++) {
		sendRegisterGlyf(slotCps[i]!, nerdGlyfsB64[i]!);
	}
	// Wide row: the same five outlines, re-registered at a fresh PUA-B
	// range with `width=2` so each occupies two terminal cells.
	for (let i = 0; i < wideSlotCps.length; i++) {
		sendRegisterGlyfWide(wideSlotCps[i]!, nerdGlyfsB64[i]!);
	}
	// Placement rows: the heart outline under each §8.5 parameter set.
	for (const slot of placementSlots) {
		sendRegisterGlyfPlaced(slot.cp, slot.params, nerdGlyfsB64[3]!);
	}
}

function registerNabla(): void {
	for (const e of nablaLetters) {
		const extent = letterExtent(e.advance, nablaLineHeight, nablaYMin);
		sendRegisterColrv1Letter(e.cp, extent, e.payload);
		// Wide duplicate for the second colrv1 line: same payload at
		// a fresh codepoint, drawn across two cells with width=2.
		sendRegisterColrv1Wide(e.cp + nablaWideOffset, extent, e.payload);
	}
}

function registerZupra(): void {
	for (const e of zupraLetters) {
		const extent = letterExtent(e.advance, zupraLineHeight, zupraYMin);
		sendRegisterColrv0Letter(e.cp, zupraUpm, extent, e.payload);
	}
}

function registerTwemoji(): void {
	for (const e of twemojiLetters) {
		sendRegisterColrv0(e.cp, twemojiUpm, e.payload);
	}
}

function registerNotoFruit(): void {
	for (const e of notoFruitLetters) {
		sendRegisterColrv1(e.cp, e.payload);
	}
}

function remapText(
	text: string,
	entries: ReadonlyArray<{letter: string; cp: number}>,
	// A width=2 glyph overflows into the following cell in *pixels*; the
	// terminal does not reserve that cell, so the codepoint still counts
	// as one logical column (== system wcwidth → no cursor/copy desync).
	// Pass true to author a trailing space after each mapped glyph for
	// the overflow to land on (used by the width=2 row).
	spaceAfterMapped = false,
): string {
	let out = '';
	for (const ch of text) {
		const mapped = entries.find(e => e.letter === ch);
		if (mapped) {
			out += String.fromCodePoint(mapped.cp);
			if (spaceAfterMapped) out += ' ';
		} else {
			out += ch;
		}
	}
	return out;
}

function glyfIcons(): string {
	return slotCps.map(cp => String.fromCodePoint(cp)).join('  ');
}

// registerVg sends fmt=vg glyphs. The payload is text, not base64.
function registerVg(glyphs: readonly VgGlyph[]): void {
	for (const g of glyphs) {
		const extra = g.extra === '' ? '' : `;${g.extra}`;
		process.stdout.write(
			`\x1b_25a1;r;cp=${g.cp.toString(16)};fmt=vg;reply=0;vb=${g.vb}${extra};${g.payload}\x1b\\`,
		);
	}
}

// vgLine joins glyphs with two spaces; a width=2 glyph takes the first.
function vgLine(glyphs: readonly VgGlyph[]): string {
	return glyphs.map(g => String.fromCodePoint(g.cp)).join('  ');
}

function glyfWideIcons(): string {
	return wideSlotCps.map(cp => String.fromCodePoint(cp)).join('  ');
}

// boxed wraps one placement glyph in a grey background block, so the
// cell box (and where the §8.5 transform put the outline inside it)
// is visible.
function boxed(cp: number): string {
	return `\x1b[48;2;64;64;64m${String.fromCodePoint(cp)}\x1b[0m`;
}

// boxedWide does the same for a width=2 glyph. The codepoint takes one
// column and the glyph overflows into the blank cell after it, so that
// cell is printed as a space inside the same grey block.
function boxedWide(cp: number): string {
	return `\x1b[48;2;64;64;64m${String.fromCodePoint(cp)} \x1b[0m`;
}

// probeSupport sends `s` and waits up to `timeoutMs` for the terminal's
// reply. Resolves to the list of `fmt` names on success, or `undefined`
// on timeout / malformed reply.
async function probeSupport(timeoutMs: number): Promise<string[] | undefined> {
	if (!process.stdin.isTTY) return undefined;

	// Raw mode so the kernel hands us the APC reply as it arrives
	// instead of buffering until a newline.
	process.stdin.setRawMode(true);
	process.stdin.resume();

	process.stdout.write(`\x1b_25a1;s\x1b\\`);

	return new Promise<string[] | undefined>(resolve => {
		let collected = Buffer.alloc(0);
		let settled = false;

		const cleanup = () => {
			clearTimeout(timer);
			process.stdin.removeListener('data', onData);
			process.stdin.pause();
			process.stdin.setRawMode(false);
		};
		const settle = (value: string[] | undefined) => {
			if (settled) return;
			settled = true;
			cleanup();
			resolve(value);
		};

		const onData = (chunk: Buffer) => {
			collected = Buffer.concat([collected, chunk]);
			if (collected.includes('\x1b\\') || collected.length > 2048) {
				settle(parseFmtField(collected));
			}
		};
		const timer = setTimeout(() => settle(undefined), timeoutMs);
		process.stdin.on('data', onData);
	});
}

// parseFmtField locates `fmt=` in an APC reply and splits its
// comma-separated list of format names. Returns [] when the field is
// empty (terminal advertises no formats); undefined when no `fmt=`
// field is present.
function parseFmtField(data: Buffer): string[] | undefined {
	const i = data.indexOf('fmt=');
	if (i < 0) return undefined;
	const start = i + 4;
	let end = start;
	while (end < data.length && data[end] !== 0x3b /* ; */ && data[end] !== 0x1b /* ESC */) {
		end++;
	}
	const value = data.slice(start, end).toString('ascii');
	return value === '' ? [] : value.split(',');
}

const fmts = await probeSupport(200);
if (fmts === undefined) {
	process.stderr.write(
		'This terminal does not implement Glyph Protocol. Run in Rio or ' +
			'another conformant terminal to see the demo.\n',
	);
	process.exit(0);
}

const hasGlyf = fmts.includes('glyf');
const hasColrv0 = fmts.includes('colrv0');
const hasColrv1 = fmts.includes('colrv1');
const hasVg = fmts.includes('vg');

if (hasGlyf) registerGlyfSlots();
if (hasColrv0) {
	registerZupra();
	registerTwemoji();
}
if (hasColrv1) {
	registerNabla();
	registerNotoFruit();
}
if (hasVg) {
	for (const group of [VG_ICONS, VG_PAINTS, VG_EMOJI_NOTO, VG_EMOJI_TWEMOJI]) {
		registerVg(group);
	}
}

const colrv1Line = hasColrv1
	? remapText(nablaText, nablaLetters)
	: '(colrv1 not advertised by terminal)';
const colrv1WideLine = hasColrv1
	? remapText(
			nablaText,
			nablaLetters.map(e => ({...e, cp: e.cp + nablaWideOffset})),
			true,
		)
	: '(colrv1 not advertised by terminal)';
const colrv1Emoji = hasColrv1
	? remapText(notoFruitText, notoFruitLetters)
	: '(colrv1 not advertised by terminal)';
const colrv0Line = hasColrv0
	? remapText(zupraText, zupraLetters)
	: '(colrv0 not advertised by terminal)';
const colrv0Emoji = hasColrv0
	? remapText(twemojiText, twemojiLetters)
	: '(colrv0 not advertised by terminal)';
const glyfLine = hasGlyf ? glyfIcons() : '(glyf not advertised by terminal)';
const glyfWideLine = hasGlyf ? glyfWideIcons() : '(glyf not advertised by terminal)';

process.stdout.write('\n');
process.stdout.write(`colrv1 example: \x1b[48;2;21;49;162m${colrv1Line}\x1b[0m\n`);
// Second line: the same letters via the width=2 duplicates.
process.stdout.write(`colrv1 example: ${colrv1WideLine}\n`);
process.stdout.write(`colrv1 example: ${colrv1Emoji}\n`);
process.stdout.write(`colrv0 example: ${colrv0Line}\n`);
process.stdout.write(`colrv0 example: ${colrv0Emoji}\n`);
process.stdout.write(`glyf examples:  ${glyfLine}\n`);
process.stdout.write(`glyf wide ex.:  ${glyfWideLine}\n`);
if (hasVg) {
	process.stdout.write(`vg icons:       ${vgLine(VG_ICONS)}\n`);
	process.stdout.write(`vg paints:      ${vgLine(VG_PAINTS)}\n`);
	process.stdout.write(`vg emoji:       ${vgLine(VG_EMOJI_NOTO)}\n`);
	process.stdout.write(`vg emoji:       ${vgLine(VG_EMOJI_TWEMOJI)}\n`);
} else {
	process.stdout.write('vg examples:    (vg not advertised by terminal)\n');
}

// §8.5 sizing/placement — the same heart outline under different
// size= / align= / pad= / width= parameters. The grey block is the
// cell box; where the heart lands inside it is the demo.
if (hasGlyf) {
	process.stdout.write(
		`placement size: ${boxed(0x100070)}  ${boxed(0x100071)}  ${boxed(
			0x100072,
		)}  ${boxed(0x100073)}   (height / contain / cover / stretch)\n`,
	);
	process.stdout.write(
		`placement align: ${boxed(0x100074)} ${boxed(0x100075)}  xy${boxed(
			0x100076,
		)}z   (start,start / end,end / baseline in text)\n`,
	);
	process.stdout.write(
		`placement more: ${boxed(0x100077)} ${boxedWide(
			0x100078,
		)}   (stretch + pad=0.2 / width=2 contain)\n`,
	);
} else {
	process.stdout.write('placement:      (glyf not advertised by terminal)\n');
}
