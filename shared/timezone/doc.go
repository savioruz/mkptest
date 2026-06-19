// Package timezone provides timezone utilities for the application.
//
// Usage Examples:
//
//  1. Basic usage after initialization:
//     now := timezone.Now()                    // Get current time in app timezone
//     appTime := timezone.ToAppTime(someTime)  // Convert any time to app timezone
//
//  2. Formatting times in app timezone:
//     formatted := timezone.Format(time.Now(), "2006-01-02 15:04:05")
//
//  3. Parsing times in app timezone:
//     t, err := timezone.Parse("2006-01-02", "2024-01-01")
//
//  4. Getting the timezone location:
//     loc := timezone.GetLocation()
//
// Supported timezone formats:
// - Standard timezone names only: "UTC", "Asia/Jakarta", "America/New_York", "Europe/London"
//
// The timezone is configured via the APP_TIMEZONE environment variable
// and is automatically initialized when the package is imported.
// Use standard IANA timezone database names for reliable cross-platform compatibility.
package timezone
