# gtk4

Drive **GTK4** from pure Go with **`CGO_ENABLED=0`** — windows, the common
controls, GObject signals and the GLib main loop — over
[`ebitengine/purego`](https://github.com/ebitengine/purego).

It is the **Linux sibling of [`go-macos/objc`](https://github.com/go-macos/objc)**:
the shared native-toolkit bridge a [`go-widgets`](https://github.com/go-widgets)
host backend embeds real controls through, so a pixel-drawn app can inlay a real
`GtkEntry`, `GtkButton` or secure field over its own rendering.

```go
ok, err := gtk4.Init()
// win := gtk4.WindowNew(); win.SetTitle("…"); win.SetDefaultSize(360, 200)
// fixed := gtk4.FixedNew()          // the overlay container
// e := gtk4.EntryNew(); e.SetVisibility(false) // a password field
// b := gtk4.ButtonNewWithLabel("OK")
// b.Connect("clicked", func() { … }) // a GObject signal, as a Go closure
// fixed.Put(e, 10, 10); fixed.Put(b, 10, 44); win.SetChild(fixed); win.Present()
```

## No cgo

`libgtk-4`, `libgobject-2.0` and `libglib-2.0` are `dlopen`'d **by soname** and
every entry point is resolved with `purego.RegisterLibFunc`. A GObject signal
reaches Go through a `purego.NewCallback`, retained for the process life (GTK
holds the C pointer). The package links with no cgo and cross-compiles like any
other Go code.

## What it binds

`Init` · `WindowNew` (+ `SetTitle`/`SetDefaultSize`/`SetChild`/`Present`) ·
`FixedNew` (+ `Put`/`Move` — absolute overlay positioning) · `ButtonNewWithLabel`
(+ `SetLabel`) · `EntryNew` (+ `SetVisibility` for a secure field) ·
`CheckButtonNewWithLabel` (+ `Active`/`SetActive`; grouped check buttons are GTK
radios) · `LabelNew` · `Text`/`SetText` (GtkEditable) · `SetSizeRequest` ·
`SetVisible` · `Unparent` · `Connect` (signals) · `MainLoop` (`Run`/`Quit`) ·
`IdleAdd` (marshal onto the GTK thread).

Pixels: `PictureNew` with `SetRGBA`/`UpdateRGBA` (the frame is copied) or
`NewSharedFrame` with `SetFrame`/`UpdateFrame` (GTK 4.16+ for the update; **no
copy**: the texture is built over the frame's own GLib memory, which
`GdkMemoryTextureBuilder` needs whole anyway,
`gdk/gdkmemorytexturebuilder.c:726`). `OverlayNew` (+ `SetOverlayChild`/
`AddOverlay`) and `AllocatedSize` let a picture fill a resizable window and
report its size.

Input method: `IMContextNew` (the user's session input method, falling back to
GTK's built-in compose/dead-key handling) and `IMContextSimpleNew` (that built-in
handling only), with `SetClientWidget`, `FocusIn`/`FocusOut`, `Reset`,
`SetCursorLocation` (anchors the candidate window), `Preedit` (the text being
composed and the caret, in characters), `OnPreeditChanged` and `OnCommit`.
`Widget.OnKeyIM` is `OnKey` with the context in front: GTK offers each key to the
input method first, and the callback sees only what it did not consume. A host
that paints its own text fields uses these to take dead keys, compose sequences
and CJK input. Pango's underline attributes of the preedit are not read.

Context menus and window close: `ContextMenuNew(widget)` attaches a GtkPopoverMenu
over a GMenu that a secondary click opens (capture phase, so it wins over a child's
own menu); `SetItems([]MenuItem)` swaps its entries in place (an empty `Label` is a
separator, a nil `OnPick` an inert greyed entry), and `Popup`, `Popdown`, `IsOpen`,
`Activate` and `Destroy` drive it. `MenuSections` is the pure layout of a flat list
into GMenu sections. `Widget.OnCloseRequest` stops or lets through the window's
close control, `Widget.Close` and `Widget.Mapped` complete it. `SetButtonChild`,
`SetTooltip`, `Picture.SetCanShrink` and `Picture.SetAlternativeText` put a picture
in a button with a caption a screen reader still has.

Dialogs (GTK 4.10+): `ShowFileDialog(parent, FileDialogRequest, done)` drives
`GtkFileDialog` (open, open several, choose a folder, save; suffix filters, the
first the default) and `ShowAlert(parent, AlertRequest, done)` drives
`GtkAlertDialog` (message, detail, up to any number of buttons; the first is the
default). Both may be called from any goroutine, show the dialog on the main loop
thread and call `done` there exactly once, with the paths as `g_file_get_path`
gives them or the index of the button pressed; a dismissal is `Cancelled`, not an
error. On a GTK older than 4.10, and off Linux, they return `ErrDialogUnsupported`
and never call `done`; `DialogsSupported` asks first.

Collections: `ColumnViewNew(columns)` is a table (a GtkColumnView in a scroller over
a GtkStringList): `SetRows([][]string)` replaces every row with one splice, so
100 000 rows are 100 000 small GObjects and only the rows on screen get widgets;
`Selected`/`SetSelected`, `OnSelect` (called for the person's change only, never
for the host's own; `Select`, `SelectPath` and `Switch` change the selection as the
person would, handler included), `Len`, `Rebuilds`. `TreeViewNew()` is a tree (a GtkListView
over a GtkTreeListModel with expander rows): `SetNodes([]TreeNode)` (children are
made lazily as a row expands; `Expanded` nodes start open), `SetSelectedPath`
(opens the ancestors), `SelectedPath`, `OnSelect`. `NotebookNew()` is a tab strip
(a GtkNotebook with empty pages, the host lays the content out): `SetTabs`,
`Current`/`SetCurrent`, `Remove`, `Len`, `OnSwitch`. Each has `Widget()` to place
and `Destroy()`; a cell may not hold NUL or the ASCII unit separator (dropped).

## Platform

**Linux only** — this is the GTK path; a `go-widgets` host uses AppKit/Win32 on
macOS/Windows. Off Linux the package still **compiles** (a consumer cross-compiles
without a build tag) and every constructor's `Init` returns
[`ErrUnsupported`](https://pkg.go.dev/github.com/go-gtk/gtk4#ErrUnsupported), so
nothing is a build-time surprise. It needs only the **runtime** `libgtk-4`
(`libgtk-4-1` on Debian/Ubuntu) — no `-dev` package.

The Linux binding is built for **amd64 and arm64 only**. `purego.Dlopen` links a
Go binary dynamically (it needs the loader, `ld-linux`), and on the other Linux
architectures a test binary run under qemu user emulation (loong64, ppc64le,
riscv64, s390x) fails with "Could not open ld-linux". Those architectures get
the same stub as every other platform: it compiles, `Init` reports
`ErrUnsupported`, and the collection constructors return nil.

## Testing

CI builds it on every platform and **runs a live test against real GTK4 under a
headless X server** (Xvfb) — creating a window, entry, secure field, button and
check button, round-tripping their values, and proving a `changed` signal reaches
Go through the main loop.

## License

BSD-3-Clause. See [LICENSE](LICENSE).
