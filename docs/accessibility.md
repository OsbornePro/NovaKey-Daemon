# Accessibility

NovaKey is designed to work well with iOS accessibility features. This page documents the supported features and a few tips for best results.

## Supported features

### VoiceOver
NovaKey supports VoiceOver throughout the app:
- Buttons and controls have meaningful labels (not just icons)
- Important events announce status (for example: pairing results, scan success, limit reached)
- Alerts and dialogs are readable and actionable

**Tip:** If VoiceOver is enabled, you can navigate the main list of secrets and open the action menu (Copy/Send/etc.) using standard VoiceOver gestures.

### Voice Control
NovaKey is Voice Control friendly:
- Primary actions use labeled controls where possible (e.g., “Add Secret”, “Listeners”)
- Menu items are named clearly so they can be activated by voice

**Tip:** Use commands like “Tap Add Secret” or “Tap Listeners”.

### Larger Text (Dynamic Type)
NovaKey supports Dynamic Type:
- Text scales when you increase system font size
- Layouts are intended to remain usable at larger sizes

**Tip:** Set text size via:
Settings → Accessibility → Display & Text Size → Larger Text

### Dark Mode
NovaKey supports system appearance:
- Works in Light and Dark Mode
- Uses system materials and colors to remain readable

**Tip:** Toggle via:
Settings → Display & Brightness

### Differentiate Without Color Alone
NovaKey avoids relying only on color to communicate meaning:
- “FREE/PRO” is shown as text, not color-only
- Paired / Not Paired uses both icon + label text

### Sufficient contrast
NovaKey primarily uses system colors/materials (which follow iOS contrast rules). If you enable higher contrast in iOS, the UI should remain legible.

**Tip:** Increase contrast via:
Settings → Accessibility → Display & Text Size → Increase Contrast

### Reduce Motion
NovaKey respects Reduce Motion:
- Animations are reduced/disabled when the system setting is enabled
- Toast transitions and other UI motion are minimized accordingly

**Tip:** Enable via:
Settings → Accessibility → Motion → Reduce Motion

## Captions and Audio Descriptions
NovaKey does not include video or audio playback content in the app itself, so captions/audio descriptions generally do not apply.

If you view tutorial videos hosted elsewhere, use iOS captioning/audio description settings as appropriate:
- Settings → Accessibility → Subtitles & Captioning

## Accessibility and security
NovaKey intentionally combines accessibility with security:
- Sensitive actions (copy/send/export) require authentication
- Secrets are not displayed after saving
- Clear feedback is provided both visually and (when enabled) via VoiceOver announcements

## Feedback
If you encounter an accessibility issue (label missing, hard-to-tap control, confusing navigation), please report it with:
- iOS version
- device model
- which screen/action caused the problem
- whether VoiceOver/Voice Control/Larger Text/Reduce Motion was enabled

