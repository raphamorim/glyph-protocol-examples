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

// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.
const slotCps: ReadonlyArray<number> = [
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
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

// Protocol bits, mirroring the spec's fmt bitfield.
const FMT_GLYF_BIT = 0b0000_0001;
const FMT_COLRV0_BIT = 0b0000_0010;
const FMT_COLRV1_BIT = 0b0000_0100;

// probeSupport sends `s` and waits up to `timeoutMs` for the terminal's
// reply. Resolves to the `fmt` bitfield on success, or `undefined` on
// timeout / malformed reply.
async function probeSupport(timeoutMs: number): Promise<number | undefined> {
	if (!process.stdin.isTTY) return undefined;

	// Raw mode so the kernel hands us the APC reply as it arrives
	// instead of buffering until a newline.
	process.stdin.setRawMode(true);
	process.stdin.resume();

	process.stdout.write(`\x1b_25a1;s\x1b\\`);

	return new Promise<number | undefined>(resolve => {
		let collected = Buffer.alloc(0);
		let settled = false;

		const cleanup = () => {
			clearTimeout(timer);
			process.stdin.removeListener('data', onData);
			process.stdin.pause();
			process.stdin.setRawMode(false);
		};
		const settle = (value: number | undefined) => {
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

// parseFmtField locates `fmt=` in an APC reply and parses the decimal
// u8 that follows.
function parseFmtField(data: Buffer): number | undefined {
	const i = data.indexOf('fmt=');
	if (i < 0) return undefined;
	const start = i + 4;
	let end = start;
	while (end < data.length && data[end]! >= 0x30 && data[end]! <= 0x39) {
		end++;
	}
	if (end === start) return undefined;
	const v = Number.parseInt(data.slice(start, end).toString('ascii'), 10);
	return Number.isFinite(v) && v >= 0 && v <= 255 ? v : undefined;
}

const fmtBits = await probeSupport(200);
if (fmtBits === undefined) {
	process.stderr.write(
		'This terminal does not implement Glyph Protocol. Run in Rio or ' +
			'another conformant terminal to see the demo.\n',
	);
	process.exit(0);
}

const hasGlyf = (fmtBits & FMT_GLYF_BIT) !== 0;
const hasColrv0 = (fmtBits & FMT_COLRV0_BIT) !== 0;
const hasColrv1 = (fmtBits & FMT_COLRV1_BIT) !== 0;

if (hasGlyf) registerGlyfSlots();
if (hasColrv0) registerZupra();
if (hasColrv1) registerNabla();

const colrv1Line = hasColrv1
	? remapText(nablaText, nablaLetters)
	: '(colrv1 not advertised by terminal)';
const colrv0Line = hasColrv0
	? remapText(zupraText, zupraLetters)
	: '(colrv0 not advertised by terminal)';
const glyfLine = hasGlyf ? glyfIcons() : '(glyf not advertised by terminal)';

process.stdout.write('\n');
process.stdout.write(`colrv1 example: \x1b[48;2;21;49;162m${colrv1Line}\x1b[0m\n`);
process.stdout.write(`colrv1 example: ${colrv1Line}\n`);
process.stdout.write(`colrv0 example: ${colrv0Line}\n`);
process.stdout.write(`glyf examples:  ${glyfLine}\n`);
