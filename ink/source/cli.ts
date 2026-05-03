#!/usr/bin/env node
// Glyph Protocol demo — sample output exercising the protocol's three
// payload formats:
//   colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//   colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//   glyf   : five Nerd Font outlines (branch / folder / home / heart /
//            Rust cog) registered at consecutive PUA-B codepoints.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

import {nablaLetters, nablaText} from './nabla_rio.js';
import {zupraLetters, zupraText, zupraUpm} from './zupra_using.js';
import {twemojiLetters, twemojiText, twemojiUpm} from './twemoji.js';
import {notoFruitLetters, notoFruitText} from './noto_fruit.js';

// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.
const slotCps: ReadonlyArray<number> = [
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
];

// `width=2` slots — the same five outlines re-registered at a fresh
// PUA-B range so the wide row prints alongside the narrow row for a
// direct visual comparison. Without `width=2` the terminal would call
// PUA-B narrow per UAX-11 / wcwidth and pack the icons into one cell.
const wideSlotCps: ReadonlyArray<number> = [
	0x100005, 0x100006, 0x100007, 0x100008, 0x100009,
];

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
		`\x1b_25a1;r;cp=${cp.toString(16)};reply=0;upm=1000;${payload}\x1b\\`,
	);
}

function sendRegisterGlyfWide(cp: number, payload: string): void {
	// Same as sendRegisterGlyf but with `width=2` so the terminal
	// reserves two cells for the codepoint at print time.
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};reply=0;upm=1000;width=2;${payload}\x1b\\`,
	);
}

function sendRegisterColrv1(cp: number, payload: string): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv1;reply=2;upm=1000;${payload}\x1b\\`,
	);
}

function sendRegisterColrv0(cp: number, upm: number, payload: string): void {
	process.stdout.write(
		`\x1b_25a1;r;cp=${cp.toString(16)};fmt=colrv0;reply=2;upm=${upm};${payload}\x1b\\`,
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
}

function registerNabla(): void {
	for (const e of nablaLetters) {
		sendRegisterColrv1(e.cp, e.payload);
	}
}

function registerZupra(): void {
	for (const e of zupraLetters) {
		sendRegisterColrv0(e.cp, zupraUpm, e.payload);
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
): string {
	let out = '';
	for (const ch of text) {
		const mapped = entries.find(e => e.letter === ch);
		out += mapped ? String.fromCodePoint(mapped.cp) : ch;
	}
	return out;
}

function glyfIcons(): string {
	return slotCps.map(cp => String.fromCodePoint(cp)).join('  ');
}

function glyfWideIcons(): string {
	return wideSlotCps.map(cp => String.fromCodePoint(cp)).join('  ');
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

if (hasGlyf) registerGlyfSlots();
if (hasColrv0) {
	registerZupra();
	registerTwemoji();
}
if (hasColrv1) {
	registerNabla();
	registerNotoFruit();
}

const colrv1Line = hasColrv1
	? remapText(nablaText, nablaLetters)
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
process.stdout.write(`colrv1 example: ${colrv1Line}\n`);
process.stdout.write(`colrv1 example: ${colrv1Emoji}\n`);
process.stdout.write(`colrv0 example: ${colrv0Line}\n`);
process.stdout.write(`colrv0 example: ${colrv0Emoji}\n`);
process.stdout.write(`glyf examples:  ${glyfLine}\n`);
process.stdout.write(`glyf wide ex.:  ${glyfWideLine}\n`);
