# Emoji for the `fmt=vg` demos

Eight emoji (😀 😎 ❤️ 🚀 🎉 🔥 🍕 🐱) from four sets. Each `.svg` is an
unmodified copy from the set's repository; each `.vg` is that SVG
converted to a `fmt=vg` payload by [`../usvg2vg`](../usvg2vg): the
viewBox on the first line, the payload on the second.

To regenerate one:

```bash
cd svg/usvg2vg
cargo run --release -- ../emoji/noto/rocket.svg -o ../emoji/noto/rocket.vg
```

After changing any of them, rebuild the demo modules from the repository
root (see the main README).

## Licences

Each set's licence is in its directory. The `.vg` files are adaptations
of the SVGs (converted to another format, otherwise unchanged) and carry
the same licence.

- `noto/`: [Noto Emoji](https://github.com/googlefonts/noto-emoji) by
  Google. The repository states its fonts are under the SIL Open Font
  License 1.1 and its tools and most image resources under the Apache
  License 2.0; its top-level `LICENSE`, copied here, is the OFL.
- `twemoji/`: [Twemoji](https://github.com/jdecked/twemoji), copyright
  Twitter, Inc and other contributors. Graphics licensed under
  [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/)
  (`LICENSE-GRAPHICS`).
- `fluent/`: [Fluent UI Emoji](https://github.com/microsoft/fluentui-emoji),
  Flat style, copyright Microsoft Corporation, MIT (`LICENSE`).
- `openmoji/`: [OpenMoji](https://openmoji.org), the open-source emoji
  and icon project, licensed under
  [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/)
  (`LICENSE.txt`). The converted `.vg` files are shared under the same
  licence.
