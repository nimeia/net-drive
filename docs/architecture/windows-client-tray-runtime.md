# Iter 21 — Windows client tray / background / notifications

## Goal

Promote the Win32 shell from a foreground-only test window into a client that can stay alive in the background and expose the common runtime actions from the Windows notification area.

## Delivered

- notification area icon created during startup and removed during shutdown
- close/minimize-to-tray behavior instead of exiting immediately
- tray context menu with:
  - Open
  - Show Dashboard
  - Show Profiles
  - Show Diagnostics
  - Start Mount
  - Stop Mount
  - Exit
- balloon notifications for significant runtime phase changes:
  - mounted
  - stopping
  - idle
  - error
- Dashboard text updated to explain tray behavior

## Design notes

The tray layer binds to the same `winclientruntime.Snapshot` used by the Dashboard and Diagnostics pages. This keeps user-visible status changes and tray notifications consistent.

The current implementation deliberately keeps the tray logic thin:

- no separate background process
- no persistent tray-specific settings yet
- no notification throttling policy beyond phase-change dedupe

That is sufficient for a product-shaped shell iteration and keeps the next steps straightforward.

## Next steps

1. persist close-to-tray / notification preferences in the profile store
2. add richer icons for healthy / busy / error states
3. add diagnostics export and “open logs” actions to the tray menu
