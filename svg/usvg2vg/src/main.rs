//! Convert an SVG into a Glyph Protocol `fmt=vg` payload.
//!
//!     cargo run --release -- emoji.svg            print `vb` and the payload
//!     cargo run --release -- emoji.svg -o out.vg  write them to a file
//!
//! usvg does the hard part: CSS, transforms, `<use>`, basic shapes and
//! gradient units all come out as absolute paths and user-space
//! gradients. Each fill and stroke becomes a layer; a group's clip path
//! becomes a `clip=<n>` layer before the layers it covers. Group opacity
//! is applied to each layer, which is exact unless shapes in the group
//! overlap. Filters, masks, patterns, images, text, dashes, blend modes
//! and nested clips are refused rather than drawn differently.
//!
//! Layers are never merged: overlapping shapes drawn in opposite
//! directions cancel under nonzero winding, so two same-colour fills
//! aren't the same as one fill of both.

use std::fmt::Write as _;
use usvg::tiny_skia_path::{PathSegment, Point, Transform};

// Limits from the draft spec (raphamorim/rio#2095).
const MAX_LAYERS: usize = 128;
const MAX_SEGMENTS: usize = 2048;
const MAX_STOPS: usize = 16;
const MAX_INK_MULTIPLE: f64 = 64.0;
const MAX_PAYLOAD: usize = 64 * 1024;

struct Converter {
    /// Coordinates are rounded to this step, 1/3600 of the canvas.
    step: f32,
    layers: Vec<String>,
    /// Index into `layers` of the clip layer in force, and its count.
    clip: Option<usize>,
    segments: usize,
    /// Path length in canvas units, counted the way the limit is.
    ink: f64,
}

type Err = String;

fn num(v: f32, digits: usize) -> String {
    let s = format!("{v:.digits$}");
    let s = s.trim_end_matches('0').trim_end_matches('.');
    match s {
        "-0" | "" => "0".into(),
        _ => s.into(),
    }
}

fn colour(c: usvg::Color, alpha: f32) -> String {
    let a = (alpha.clamp(0.0, 1.0) * 255.0).round() as u8;
    if a == 255 {
        format!("#{:02x}{:02x}{:02x}", c.red, c.green, c.blue)
    } else {
        format!("#{:02x}{:02x}{:02x}{a:02x}", c.red, c.green, c.blue)
    }
}

impl Converter {
    fn round(&self, v: f32) -> f32 {
        (v / self.step).round() * self.step
    }

    /// Path data with `ts` applied, relative commands from rounded
    /// absolute points. `stroke` is the width charged per segment.
    fn path_data(&mut self, path: &usvg::tiny_skia_path::Path, ts: Transform, stroke: f64) -> String {
        let mut out = String::new();
        let map = |p: Point| {
            let mut p = p;
            ts.map_point(&mut p);
            p
        };
        let (mut cx, mut cy, mut sx, mut sy) = (0f32, 0f32, 0f32, 0f32);
        let d = |a: (f32, f32), b: (f32, f32)| (((b.0 - a.0) as f64).powi(2) + ((b.1 - a.1) as f64).powi(2)).sqrt();
        for seg in path.segments() {
            self.segments += 1;
            match seg {
                PathSegment::MoveTo(p) => {
                    let p = map(p);
                    let (x, y) = (self.round(p.x), self.round(p.y));
                    write!(out, "m{} {}", num(x - cx, 2), num(y - cy, 2)).unwrap();
                    (cx, cy, sx, sy) = (x, y, x, y);
                    self.ink += stroke;
                }
                PathSegment::LineTo(p) => {
                    let p = map(p);
                    let (x, y) = (self.round(p.x), self.round(p.y));
                    write!(out, "l{} {}", num(x - cx, 2), num(y - cy, 2)).unwrap();
                    self.ink += d((cx, cy), (x, y)) + stroke;
                    (cx, cy) = (x, y);
                }
                PathSegment::QuadTo(a, p) => {
                    let (a, p) = (map(a), map(p));
                    let (ax, ay, x, y) = (self.round(a.x), self.round(a.y), self.round(p.x), self.round(p.y));
                    write!(out, "q{} {} {} {}", num(ax - cx, 2), num(ay - cy, 2), num(x - cx, 2), num(y - cy, 2)).unwrap();
                    self.ink += d((cx, cy), (ax, ay)) + d((ax, ay), (x, y)) + stroke;
                    (cx, cy) = (x, y);
                }
                PathSegment::CubicTo(a, b, p) => {
                    let (a, b, p) = (map(a), map(b), map(p));
                    let (ax, ay) = (self.round(a.x), self.round(a.y));
                    let (bx, by) = (self.round(b.x), self.round(b.y));
                    let (x, y) = (self.round(p.x), self.round(p.y));
                    write!(
                        out,
                        "c{} {} {} {} {} {}",
                        num(ax - cx, 2), num(ay - cy, 2), num(bx - cx, 2), num(by - cy, 2), num(x - cx, 2), num(y - cy, 2)
                    )
                    .unwrap();
                    self.ink += d((cx, cy), (ax, ay)) + d((ax, ay), (bx, by)) + d((bx, by), (x, y)) + stroke;
                    (cx, cy) = (x, y);
                }
                PathSegment::Close => {
                    out.push('z');
                    self.ink += d((cx, cy), (sx, sy)) + stroke;
                    (cx, cy) = (sx, sy);
                }
            }
        }
        out.replace(" -", "-")
    }

    fn paint(&self, paint: &usvg::Paint, opacity: f32, ts: Transform) -> Result<String, Err> {
        let stops = |stops: &[usvg::Stop]| -> Result<String, Err> {
            if stops.len() > MAX_STOPS {
                return Err(format!("{} gradient stops, the limit is {MAX_STOPS}", stops.len()));
            }
            Ok(stops
                .iter()
                .map(|s| format!(" / {} {}", num(s.offset().get(), 3), colour(s.color(), s.opacity().get() * opacity)))
                .collect())
        };
        let tail = |spread: usvg::SpreadMethod, m: Transform| {
            let mut s = String::new();
            match spread {
                usvg::SpreadMethod::Pad => {}
                usvg::SpreadMethod::Reflect => s.push_str(" reflect"),
                usvg::SpreadMethod::Repeat => s.push_str(" repeat"),
            }
            if !m.is_identity() {
                write!(s, " m {} {} {} {} {} {}", num(m.sx, 4), num(m.ky, 4), num(m.kx, 4), num(m.sy, 4), num(m.tx, 2), num(m.ty, 2)).unwrap();
            }
            s
        };
        Ok(match paint {
            usvg::Paint::Color(c) => colour(*c, opacity),
            usvg::Paint::LinearGradient(g) => format!(
                "lin({} {} {} {}{}{})",
                num(g.x1(), 2), num(g.y1(), 2), num(g.x2(), 2), num(g.y2(), 2),
                tail(g.spread_method(), ts.pre_concat(g.transform())),
                stops(g.stops())?
            ),
            usvg::Paint::RadialGradient(g) => format!(
                "rad({} {} {} {} {} {}{}{})",
                num(g.fx(), 2), num(g.fy(), 2), num(g.fr().get(), 2),
                num(g.cx(), 2), num(g.cy(), 2), num(g.r().get(), 2),
                tail(g.spread_method(), ts.pre_concat(g.transform())),
                stops(g.stops())?
            ),
            usvg::Paint::Pattern(_) => return Err("patterns".into()),
        }
        .replace(" -", "-"))
    }

    fn push(&mut self, layer: String) {
        self.layers.push(layer);
        if let Some(i) = self.clip {
            // Grow the clip's count to cover this layer.
            let clip = &mut self.layers[i];
            let (head, rest) = clip.split_once(',').unwrap();
            let n: usize = head["clip=".len()..].parse().unwrap();
            *clip = format!("clip={},{}", n + 1, rest);
        }
    }

    fn group(&mut self, g: &usvg::Group, opacity: f32) -> Result<(), Err> {
        if !g.filters().is_empty() {
            return Err("filters".into());
        }
        if g.mask().is_some() {
            return Err("masks".into());
        }
        if g.blend_mode() != usvg::BlendMode::Normal {
            return Err("blend modes".into());
        }
        let opacity = opacity * g.opacity().get();
        let mut entered = false;
        if let Some(clip) = g.clip_path() {
            if self.clip.is_some() || clip.clip_path().is_some() {
                return Err("nested clip paths".into());
            }
            let mut data = String::new();
            let mut rule = None;
            self.clip_paths(clip.root(), g.abs_transform().pre_concat(clip.transform()), &mut data, &mut rule)?;
            let rule = rule.unwrap_or("nonzero");
            // Starts at 0 and grows as clipped layers are pushed.
            self.layers.push(format!("clip=0,{rule}:{data}"));
            self.clip = Some(self.layers.len() - 1);
            entered = true;
        }
        for node in g.children() {
            match node {
                usvg::Node::Group(c) => self.group(c, opacity)?,
                usvg::Node::Path(p) => self.path(p, opacity)?,
                usvg::Node::Image(_) => return Err("images".into()),
                usvg::Node::Text(_) => return Err("text".into()),
            }
        }
        if entered {
            let i = self.clip.take().unwrap();
            if self.layers[i].starts_with("clip=0,") {
                // Nothing drawn through it.
                self.layers.remove(i);
            }
        }
        Ok(())
    }

    fn clip_paths(&mut self, g: &usvg::Group, ts: Transform, data: &mut String, rule: &mut Option<&'static str>) -> Result<(), Err> {
        for node in g.children() {
            match node {
                usvg::Node::Path(p) if p.is_visible() => {
                    let this = match p.fill().map(|f| f.rule()) {
                        Some(usvg::FillRule::EvenOdd) => "evenodd",
                        _ => "nonzero",
                    };
                    if rule.is_some_and(|r| r != this) {
                        return Err("a clip path mixing fill rules".into());
                    }
                    *rule = Some(this);
                    let d = self.path_data(p.data(), ts, 0.0);
                    // Each part starts with a relative moveto from the
                    // origin; keep it absolute when joining.
                    data.push_str(&absolute_start(&d));
                }
                usvg::Node::Group(c) => {
                    if c.clip_path().is_some() {
                        return Err("nested clip paths".into());
                    }
                    self.clip_paths(c, ts.pre_concat(c.transform()), data, rule)?
                }
                _ => {}
            }
        }
        Ok(())
    }

    fn path(&mut self, p: &usvg::Path, opacity: f32) -> Result<(), Err> {
        if !p.is_visible() {
            return Ok(());
        }
        let ts = p.abs_transform();
        let mut layers = Vec::new();
        if let Some(fill) = p.fill() {
            let rule = match fill.rule() {
                usvg::FillRule::NonZero => "nonzero",
                usvg::FillRule::EvenOdd => "evenodd",
            };
            let paint = self.paint(fill.paint(), opacity * fill.opacity().get(), ts)?;
            let d = self.path_data(p.data(), ts, 0.0);
            layers.push(format!("fill={rule},{paint}:{d}"));
        }
        if let Some(stroke) = p.stroke() {
            if stroke.dasharray().is_some() {
                return Err("dashes".into());
            }
            // Width scales with the transform; emoji only scale uniformly.
            let width = stroke.width().get() * (ts.sx * ts.sx + ts.ky * ts.ky).sqrt();
            let cap = match stroke.linecap() {
                usvg::LineCap::Butt => "butt",
                usvg::LineCap::Round => "round",
                usvg::LineCap::Square => "square",
            };
            let join = match stroke.linejoin() {
                usvg::LineJoin::Round => "round".to_string(),
                usvg::LineJoin::Bevel => "bevel".to_string(),
                usvg::LineJoin::Miter | usvg::LineJoin::MiterClip => {
                    let limit = stroke.miterlimit().get();
                    if limit == 4.0 { "miter".into() } else { format!("miter{}", num(limit, 2)) }
                }
            };
            let style = if cap == "round" && join == "round" { String::new() } else { format!("/{cap}/{join}") };
            let paint = self.paint(stroke.paint(), opacity * stroke.opacity().get(), ts)?;
            let d = self.path_data(p.data(), ts, width as f64);
            let stroke = format!("stroke={}{style},{paint}:{d}", num(width, 2));
            if p.paint_order() == usvg::PaintOrder::StrokeAndFill {
                layers.insert(0, stroke);
            } else {
                layers.push(stroke);
            }
        }
        for l in layers {
            self.push(l);
        }
        Ok(())
    }
}

/// `m4 17 6-6` -> `M4 17l6-6`: a leading relative moveto is absolute
/// on its own, but not once it follows other path data.
fn absolute_start(d: &str) -> String {
    let Some(rest) = d.strip_prefix('m') else { return d.to_string() };
    // The first two numbers are the point; anything after them that is
    // a number continues as relative linetos.
    let mut nums = 0;
    let mut i = 0;
    let b = rest.as_bytes();
    while i < b.len() && nums < 2 {
        let start = i;
        if b[i] == b'-' || b[i] == b'+' {
            i += 1;
        }
        while i < b.len() && (b[i].is_ascii_digit() || b[i] == b'.') {
            i += 1;
        }
        if i > start {
            nums += 1;
        }
        if nums < 2 && i < b.len() && b[i] == b' ' {
            i += 1;
        }
    }
    let (point, tail) = rest.split_at(i);
    let implicit = tail.starts_with(|c: char| c.is_ascii_digit() || c == '-' || c == '.' || c == ' ');
    format!("M{point}{}{}", if implicit { "l" } else { "" }, tail.trim_start())
}

fn convert(data: &[u8]) -> Result<(String, String), Err> {
    let tree = usvg::Tree::from_data(data, &usvg::Options::default()).map_err(|e| e.to_string())?;
    let size = tree.size();
    let side = size.width().max(size.height());
    let mut c = Converter { step: side / 3600.0, layers: Vec::new(), clip: None, segments: 0, ink: 0.0 };
    c.group(tree.root(), 1.0)?;
    if c.layers.is_empty() {
        return Err("nothing to draw".into());
    }
    if c.layers.len() > MAX_LAYERS {
        return Err(format!("{} layers, the limit is {MAX_LAYERS}", c.layers.len()));
    }
    if c.segments > MAX_SEGMENTS {
        return Err(format!("{} segments, the limit is {MAX_SEGMENTS}", c.segments));
    }
    if c.ink > MAX_INK_MULTIPLE * side as f64 {
        return Err(format!("{:.0} viewBox sizes of path, the limit is {MAX_INK_MULTIPLE}", c.ink / side as f64));
    }
    let payload = c.layers.join("|");
    if payload.len() > MAX_PAYLOAD {
        return Err(format!("{} bytes, the limit is {MAX_PAYLOAD}", payload.len()));
    }
    Ok((format!("0,0,{},{}", num(size.width(), 2), num(size.height(), 2)), payload))
}

fn main() {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let Some(input) = args.first() else {
        eprintln!("usage: usvg2vg <file.svg> [-o out.vg]");
        std::process::exit(2);
    };
    let data = std::fs::read(input).unwrap_or_else(|e| {
        eprintln!("{input}: {e}");
        std::process::exit(1);
    });
    match convert(&data) {
        Ok((vb, payload)) => {
            let text = format!("{vb}\n{payload}\n");
            match args.iter().position(|a| a == "-o").and_then(|i| args.get(i + 1)) {
                Some(out) => std::fs::write(out, text).unwrap(),
                None => print!("{text}"),
            }
        }
        Err(e) => {
            eprintln!("{input}: {e}");
            std::process::exit(1);
        }
    }
}
