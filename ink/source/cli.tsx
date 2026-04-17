#!/usr/bin/env node
// Glyph Protocol demo — registers five real Nerd Font outlines at
// empty Supplementary PUA-B codepoints (U+100000..U+100004). The
// `glyf` bytes come straight out of JetBrainsMono Nerd Font via
// fontTools, so the icons look exactly like Nerd Fonts but the
// codepoints they're mounted at are empty — you don't need Nerd
// Fonts installed to see them.
//
// Keybindings:
//   ↑ re-register all
//   ↓ clear all registrations
//   q / esc / ctrl+c  quit

import React, {useEffect, useState} from 'react';
import {Box, render, Text, useApp, useInput} from 'ink';

// -------------------------------------------------------------------------
// Slots + glyf payloads
// -------------------------------------------------------------------------

const slotCps: ReadonlyArray<number> = [
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
];

const slotLabels = ['branch', 'folder', 'home', 'heart', 'rust'] as const;

// Base64 `glyf` bytes extracted from JetBrainsMono Nerd Font via
// fontTools. units per em = 1000. Source Nerd Font codepoints:
//   branch  U+E0A0
//   folder  U+F07B
//   home    U+F015
//   heart   U+F004
//   rust    U+E7A8
const nerdGlyfsB64: ReadonlyArray<string> = [
	'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',
];

// -------------------------------------------------------------------------
// Wire helpers
// -------------------------------------------------------------------------

function sendClearAll(): void {
	process.stdout.write(`\x1b_1cc6D;c\x1b\\`);
}

function sendRegister(cp: number, payload: string): void {
	// `fmt` is implicit (`glyf`); `upm` defaults to 1000 but we send
	// it explicitly to match the font's authoring EM.
	process.stdout.write(
		`\x1b_1cc6D;r;cp=${cp.toString(16)};upm=1000;${payload}\x1b\\`,
	);
}

function registerAll(): void {
	for (let i = 0; i < slotCps.length; i++) {
		sendRegister(slotCps[i]!, nerdGlyfsB64[i]!);
	}
}

// -------------------------------------------------------------------------
// App
// -------------------------------------------------------------------------

function App() {
	const {exit} = useApp();
	const [cleared, setCleared] = useState(false);

	useEffect(() => {
		registerAll();
	}, []);

	useInput((input, key) => {
		if (input === 'q' || key.escape || (key.ctrl && input === 'c')) {
			sendClearAll();
			exit();
			return;
		}
		if (key.upArrow) {
			registerAll();
			setCleared(false);
		} else if (key.downArrow) {
			sendClearAll();
			setCleared(true);
		}
	});

	const promptLine = ` ${String.fromCodePoint(slotCps[0]!)} main  ${String.fromCodePoint(
		slotCps[1]!,
	)} src  ${String.fromCodePoint(slotCps[2]!)} ~  ${String.fromCodePoint(
		slotCps[3]!,
	)} favs  ${String.fromCodePoint(slotCps[4]!)} rio `;

	return (
		<Box flexDirection="column" alignItems="center" paddingY={1}>
			<Box borderStyle="round" paddingX={2} marginBottom={1}>
				<Text>
					Glyph Protocol — real Nerd Font outlines at empty PUA-B codepoints
				</Text>
			</Box>

			<Box
				borderStyle="single"
				paddingX={2}
				paddingY={1}
				marginBottom={1}
			>
				<Text bold>{promptLine}</Text>
			</Box>

			<Box
				borderStyle="single"
				paddingX={2}
				flexDirection="column"
				marginBottom={1}
			>
				{slotCps.map((cp, i) => (
					<Text key={cp}>
						{`U+${cp.toString(16).toUpperCase().padStart(6, '0')}  ${slotLabels[i]!.padEnd(7)}  ${String.fromCodePoint(cp)}`}
					</Text>
				))}
			</Box>

			<Box
				borderStyle="single"
				paddingX={2}
				flexDirection="column"
			>
				<Text>{'\u2191  re-register all'}</Text>
				<Text>{'\u2193  clear all registrations'}</Text>
				<Text>q  quit</Text>
				<Text> </Text>
				<Text>
					{cleared
						? 'cleared (press \u2191 to re-register)'
						: '5 Nerd Font outlines registered'}
				</Text>
			</Box>
		</Box>
	);
}

// -------------------------------------------------------------------------
// Boot
// -------------------------------------------------------------------------

const app = render(<App />);
app.waitUntilExit().finally(() => {
	sendClearAll();
});
