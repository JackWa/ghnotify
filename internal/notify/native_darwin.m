//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

#include "native_darwin.h"

@interface GHNDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation GHNDelegate

// Show banners even if ghnotify happens to be the active app.
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completionHandler {
	if (@available(macOS 11.0, *)) {
		completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList);
	} else {
		completionHandler(UNNotificationPresentationOptionAlert);
	}
}

// Open the PR in the default browser when the notification is clicked.
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
didReceiveNotificationResponse:(UNNotificationResponse *)response
         withCompletionHandler:(void (^)(void))completionHandler {
	if ([response.actionIdentifier isEqualToString:UNNotificationDefaultActionIdentifier]) {
		NSString *u = response.notification.request.content.userInfo[@"url"];
		NSURL *url = u.length ? [NSURL URLWithString:u] : nil;
		NSString *scheme = url.scheme.lowercaseString;
		if ([scheme isEqualToString:@"https"] || [scheme isEqualToString:@"http"]) {
			[[NSWorkspace sharedWorkspace] openURL:url];
		}
	}
	completionHandler();
}

@end

static GHNDelegate *ghnDelegate;
static BOOL ghnReady;

int ghn_init(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		// UNUserNotificationCenter throws when the process has no bundle
		// identifier, e.g. when bin/ghnotify is run directly.
		if ([[NSBundle mainBundle] bundleIdentifier] == nil) {
			return;
		}
		UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
		ghnDelegate = [GHNDelegate new];
		center.delegate = ghnDelegate;
		[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
		                      completionHandler:^(BOOL granted, NSError *error) {
			if (error != nil) {
				NSLog(@"ghnotify: notification authorization: %@", error);
			} else if (!granted) {
				NSLog(@"ghnotify: notifications are disabled in System Settings");
			}
		}];
		ghnReady = YES;
	});
	return ghnReady ? 1 : 0;
}

static NSString *str(const char *s) {
	NSString *v = s ? [NSString stringWithUTF8String:s] : nil;
	return v ?: @"";
}

static const int64_t ghnTimeout = 5 * NSEC_PER_SEC;

static BOOL ghnDenied(UNUserNotificationCenter *center) {
	__block UNAuthorizationStatus status = UNAuthorizationStatusNotDetermined;
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
		status = settings.authorizationStatus;
		dispatch_semaphore_signal(done);
	}];
	if (dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, ghnTimeout)) != 0) {
		return NO;
	}
	return status == UNAuthorizationStatusDenied;
}

int ghn_post(const char *ident, const char *title, const char *body, const char *url) {
	if (!ghn_init()) {
		return 0;
	}
	@autoreleasepool {
		UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
		if (ghnDenied(center)) {
			return 0;
		}
		UNMutableNotificationContent *content = [UNMutableNotificationContent new];
		content.title = str(title);
		content.body = str(body);
		NSString *u = str(url);
		if (u.length) {
			content.userInfo = @{@"url" : u};
		}
		NSString *identifier = str(ident);
		if (!identifier.length) {
			identifier = [[NSUUID UUID] UUIDString];
		}
		UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:identifier
		                                                                  content:content
		                                                                  trigger:nil];
		__block BOOL ok = NO;
		dispatch_semaphore_t done = dispatch_semaphore_create(0);
		[center addNotificationRequest:req
		         withCompletionHandler:^(NSError *error) {
			if (error != nil) {
				NSLog(@"ghnotify: notification: %@", error);
			} else {
				ok = YES;
			}
			dispatch_semaphore_signal(done);
		}];
		if (dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, ghnTimeout)) != 0) {
			return 1;
		}
		return ok ? 1 : 0;
	}
}
