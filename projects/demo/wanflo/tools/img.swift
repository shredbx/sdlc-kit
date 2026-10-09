// img.swift — one-off asset preparation for the Wanflo site.
//
// Uses only CoreGraphics/ImageIO, which ship with macOS. Nothing is installed to run this,
// and nothing here runs at build time: it is a manual step whose OUTPUT is committed under
// static/. Kept in the repo so the same crop can be reproduced instead of re-guessed.
//
//   swift tools/img.swift info   <in>
//   swift tools/img.swift crop   <in> <out> <x> <y> <w> <h> [maxw]
//   swift tools/img.swift scale  <in> <out> <maxw>
//   swift tools/img.swift trim   <in> <out> [maxw] [alphaMin]   # drop transparent border
//   swift tools/img.swift pad    <in> <out> <size> <inset> <#RRGGBB|none>  # square canvas
//   swift tools/img.swift grid   <in> <out> <step>        # coordinate grid, for locating crops
//   swift tools/img.swift pick   <in> <x> <y>             # sample one pixel

import Foundation
import CoreGraphics
import ImageIO
import UniformTypeIdentifiers

func die(_ m: String) -> Never { FileHandle.standardError.write((m + "\n").data(using: .utf8)!); exit(1) }

func load(_ path: String) -> CGImage {
    guard let src = CGImageSourceCreateWithURL(URL(fileURLWithPath: path) as CFURL, nil),
          let img = CGImageSourceCreateImageAtIndex(src, 0, nil) else { die("cannot read \(path)") }
    return img
}

func write(_ img: CGImage, _ path: String) {
    let url = URL(fileURLWithPath: path) as CFURL
    let ext = (path as NSString).pathExtension.lowercased()
    let type: CFString
    switch ext {
    case "png":          type = UTType.png.identifier as CFString
    case "jpg", "jpeg":  type = UTType.jpeg.identifier as CFString
    case "webp":         type = "org.webmproject.webp" as CFString
    default: die("unsupported output extension: .\(ext)")
    }
    guard let dst = CGImageDestinationCreateWithURL(url, type, 1, nil) else {
        die("cannot encode .\(ext) on this system — try .png")
    }
    // 0.86 is visually lossless for screenshots at these sizes and keeps the page light.
    CGImageDestinationAddImage(dst, img, [kCGImageDestinationLossyCompressionQuality: 0.86] as CFDictionary)
    guard CGImageDestinationFinalize(dst) else { die("write failed: \(path)") }
}

/// Redraw at `w`x`h`. Always into a fresh premultiplied-first ARGB context so that
/// screenshots in odd source colour spaces come out identical everywhere.
func resize(_ img: CGImage, _ w: Int, _ h: Int) -> CGImage {
    guard let ctx = CGContext(data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: 0,
                              space: CGColorSpace(name: CGColorSpace.sRGB)!,
                              bitmapInfo: CGImageAlphaInfo.premultipliedFirst.rawValue) else { die("context") }
    ctx.interpolationQuality = .high
    ctx.draw(img, in: CGRect(x: 0, y: 0, width: w, height: h))
    guard let out = ctx.makeImage() else { die("makeImage") }
    return out
}

func capped(_ img: CGImage, _ maxw: Int?) -> CGImage {
    guard let maxw, img.width > maxw else { return img }
    let h = Int((Double(img.height) * Double(maxw) / Double(img.width)).rounded())
    return resize(img, maxw, h)
}

/// Row-major RGBA bytes, so alpha and colour can be inspected without guessing the layout.
func rgba(_ img: CGImage) -> ([UInt8], Int, Int) {
    let w = img.width, h = img.height
    var buf = [UInt8](repeating: 0, count: w * h * 4)
    buf.withUnsafeMutableBytes { raw in
        let ctx = CGContext(data: raw.baseAddress, width: w, height: h, bitsPerComponent: 8,
                            bytesPerRow: w * 4, space: CGColorSpace(name: CGColorSpace.sRGB)!,
                            bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)!
        ctx.draw(img, in: CGRect(x: 0, y: 0, width: w, height: h))
    }
    return (buf, w, h)
}

let a = CommandLine.arguments
guard a.count >= 3 else { die("usage: img.swift <info|crop|scale|trim|grid|pick> ...") }
let cmd = a[1], inPath = a[2]
let img = load(inPath)

switch cmd {
case "info":
    print("\(img.width)x\(img.height) alpha=\(img.alphaInfo.rawValue) bpc=\(img.bitsPerComponent)")

case "pick":
    let (b, w, _) = rgba(img)
    let x = Int(a[3])!, y = Int(a[4])!
    let i = (y * w + x) * 4
    print(String(format: "#%02X%02X%02X a=%d", b[i], b[i+1], b[i+2], b[i+3]))

case "crop":
    guard a.count >= 8 else { die("crop <in> <out> <x> <y> <w> <h> [maxw]") }
    let r = CGRect(x: Int(a[4])!, y: Int(a[5])!, width: Int(a[6])!, height: Int(a[7])!)
    guard let c = img.cropping(to: r) else { die("crop rect outside image") }
    let out = capped(c, a.count > 8 ? Int(a[8]) : nil)
    write(out, a[3]); print("\(a[3]) \(out.width)x\(out.height)")

case "scale":
    let out = capped(img, Int(a[4])!)
    write(out, a[3]); print("\(a[3]) \(out.width)x\(out.height)")

case "trim":
    let (b, w, h) = rgba(img)
    // A soft drop shadow islow-alpha; raising the floor snaps the box to the artwork itself
    // instead of to the halo around it. Default 8 = "anything not fully transparent".
    let floorA = UInt8(a.count > 5 ? Int(a[5])! : 8)
    var x0 = w, y0 = h, x1 = -1, y1 = -1
    for y in 0..<h { for x in 0..<w where b[(y * w + x) * 4 + 3] > floorA {
        if x < x0 { x0 = x }; if x > x1 { x1 = x }
        if y < y0 { y0 = y }; if y > y1 { y1 = y }
    } }
    guard x1 >= x0 else { die("image is fully transparent") }
    guard let c = img.cropping(to: CGRect(x: x0, y: y0, width: x1 - x0 + 1, height: y1 - y0 + 1)) else { die("trim") }
    let out = capped(c, (a.count > 4 && a[4] != "-") ? Int(a[4]) : nil)
    write(out, a[3]); print("\(a[3]) \(out.width)x\(out.height)  (trimmed from \(w)x\(h) at \(x0),\(y0))")

case "grid":
    // Draws labelled rules over the image so a crop rect can be READ off it rather than guessed.
    let step = Int(a[4])!
    let w = img.width, h = img.height
    let ctx = CGContext(data: nil, width: w, height: h, bitsPerComponent: 8, bytesPerRow: 0,
                        space: CGColorSpace(name: CGColorSpace.sRGB)!,
                        bitmapInfo: CGImageAlphaInfo.premultipliedFirst.rawValue)!
    ctx.draw(img, in: CGRect(x: 0, y: 0, width: w, height: h))
    ctx.setLineWidth(1)
    for x in stride(from: 0, to: w, by: step) {
        ctx.setStrokeColor(CGColor(red: 1, green: 0, blue: 0, alpha: x % (step * 5) == 0 ? 0.9 : 0.35))
        ctx.move(to: CGPoint(x: CGFloat(x), y: 0)); ctx.addLine(to: CGPoint(x: CGFloat(x), y: CGFloat(h))); ctx.strokePath()
    }
    for y in stride(from: 0, to: h, by: step) {
        // y is measured from the TOP in the output, matching how crop rects are written here.
        let cy = CGFloat(h - y)
        ctx.setStrokeColor(CGColor(red: 0, green: 0.5, blue: 1, alpha: y % (step * 5) == 0 ? 0.9 : 0.35))
        ctx.move(to: CGPoint(x: 0, y: cy)); ctx.addLine(to: CGPoint(x: CGFloat(w), y: cy)); ctx.strokePath()
    }
    write(ctx.makeImage()!, a[3]); print("\(a[3]) grid step=\(step)")

case "pad":
    guard a.count >= 7 else { die("pad <in> <out> <size> <inset> <#RRGGBB|none>") }
    let size = Int(a[4])!, inset = Int(a[5])!, fill = a[6]
    let ctx = CGContext(data: nil, width: size, height: size, bitsPerComponent: 8, bytesPerRow: 0,
                        space: CGColorSpace(name: CGColorSpace.sRGB)!,
                        bitmapInfo: CGImageAlphaInfo.premultipliedFirst.rawValue)!
    if fill != "none" {
        var hex = fill; hex.removeFirst()
        let v = UInt32(hex, radix: 16)!
        ctx.setFillColor(CGColor(red: CGFloat((v >> 16) & 255) / 255, green: CGFloat((v >> 8) & 255) / 255,
                                 blue: CGFloat(v & 255) / 255, alpha: 1))
        ctx.fill(CGRect(x: 0, y: 0, width: size, height: size))
    }
    // Fit inside the inset box, preserving aspect — never stretch a logo.
    let box = size - inset * 2
    let s = min(Double(box) / Double(img.width), Double(box) / Double(img.height))
    let dw = Int((Double(img.width) * s).rounded()), dh = Int((Double(img.height) * s).rounded())
    ctx.interpolationQuality = .high
    ctx.draw(img, in: CGRect(x: (size - dw) / 2, y: (size - dh) / 2, width: dw, height: dh))
    write(ctx.makeImage()!, a[3]); print("\(a[3]) \(size)x\(size) fill=\(fill)")

default: die("unknown command \(cmd)")
}
