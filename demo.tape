# lazyreno demo recording
Output demo.gif

Set Shell bash
Set FontSize 14
Set Width 1400
Set Height 800
Set Padding 0
Set Framerate 30
Set CursorBlink false

Hide
Type "./target/release/lazyreno"
Enter
Sleep 20s
Show

# Browse repos in sidebar
Sleep 1s
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 400ms

# Enter PR table
Enter
Sleep 1s

# Browse PRs
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "k"
Sleep 500ms

# Move to detail panel
Type "l"
Sleep 1.5s

# Back to PRs
Type "h"
Sleep 800ms

# Toggle fork visibility
Type "f"
Sleep 1s
Type "f"
Sleep 800ms

# Show help overlay
Type "?"
Sleep 2.5s
Type "?"
Sleep 500ms

# Quit
Hide
Type "q"
Sleep 500ms
