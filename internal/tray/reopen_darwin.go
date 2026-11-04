//go:build darwin

package tray

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// reopenMenu pops the status item menu open again. macOS closes an NSMenu on
// every item click, and systray does not expose its NSStatusItem, so read the
// delegate's statusItem ivar through KVC. Delay so AppKit finishes closing the
// menu before it reopens.
static void reopenMenu(void) {
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 100 * NSEC_PER_MSEC), dispatch_get_main_queue(), ^{
		@try {
			NSStatusItem *item = [(NSObject *)[NSApp delegate] valueForKey:@"statusItem"];
			[item.button performClick:nil];
		} @catch (NSException *e) {
			NSLog(@"ghnotify: cannot reopen menu: %@", e);
		}
	});
}
*/
import "C"

// reopenMenu shows the tray menu again after a click closed it.
func reopenMenu() { C.reopenMenu() }
