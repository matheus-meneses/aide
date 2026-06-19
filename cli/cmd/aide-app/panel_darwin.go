//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <objc/runtime.h>

// AidePanel re-types the Wails WebviewWindow (a plain NSWindow) as a
// non-activating NSPanel. A plain NSWindow cannot be drawn over another app's
// fullscreen Space even with CanJoinAllSpaces|FullScreenAuxiliary; only a
// non-activating panel can. Wails reads `window.webView` in its own cgo, so we
// re-expose it by locating the WKWebView subview instead of relying on the
// original ivar layout (which object_setClass would otherwise invalidate).
@interface AidePanel : NSPanel
@end

@implementation AidePanel
- (BOOL)canBecomeKeyWindow { return YES; }
- (BOOL)canBecomeMainWindow { return YES; }
- (WKWebView *)webView {
	for (NSView *v in self.contentView.subviews) {
		if ([v isKindOfClass:[WKWebView class]]) {
			return (WKWebView *)v;
		}
	}
	return nil;
}
- (void)setWebView:(WKWebView *)wv { (void)wv; }
@end

static void aideApplyPanelTraits(NSWindow *w) {
	w.styleMask |= NSWindowStyleMaskNonactivatingPanel;
	w.level = NSPopUpMenuWindowLevel;
	w.collectionBehavior =
		NSWindowCollectionBehaviorCanJoinAllSpaces |
		NSWindowCollectionBehaviorFullScreenAuxiliary;
	w.hidesOnDeactivate = NO;
}

static void aidePromotePanel(void *win) {
	NSWindow *w = (__bridge NSWindow *)win;
	dispatch_async(dispatch_get_main_queue(), ^{
		if (![w isKindOfClass:[AidePanel class]]) {
			object_setClass(w, [AidePanel class]);
		}
		aideApplyPanelTraits(w);
	});
}

static void aideShowPanel(void *win) {
	NSWindow *w = (__bridge NSWindow *)win;
	dispatch_async(dispatch_get_main_queue(), ^{
		aideApplyPanelTraits(w);
		[w orderFrontRegardless];
		[w makeKeyWindow];
	});
}

*/
import "C"

import "unsafe"

func promotePanel(win unsafe.Pointer) { C.aidePromotePanel(win) }

func showPanelNative(win unsafe.Pointer) { C.aideShowPanel(win) }
