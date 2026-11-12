//go:build darwin && cgo

package notify

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework AppKit -framework UserNotifications
#include <stdlib.h>
#include "native_darwin.h"
*/
import "C"

import "unsafe"

// nativePost posts a notification through UNUserNotificationCenter, so it is
// owned by ghnotify.app rather than by Script Editor (which is what
// osascript's "display notification" uses). Clicking the notification opens
// url, when set. It returns false when the process is not running from an
// .app bundle, when notifications are denied, or when posting fails,
// so the caller can fall back to another mechanism.
func nativePost(id, title, body, url string) bool {
	ci := C.CString(id)
	ct := C.CString(title)
	cb := C.CString(body)
	cu := C.CString(url)
	defer C.free(unsafe.Pointer(ci))
	defer C.free(unsafe.Pointer(ct))
	defer C.free(unsafe.Pointer(cb))
	defer C.free(unsafe.Pointer(cu))
	return C.ghn_post(ci, ct, cb, cu) != 0
}

// Init registers the click handler and asks for notification permission.
// Call it once at startup so the permission prompt appears before the first
// notification, and so a click that relaunches the app is handled.
func Init() {
	C.ghn_init()
}
