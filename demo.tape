# gotermsql Demo
# VHS tape file - https://github.com/charmbracelet/vhs

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set PlaybackSpeed 0.5

# Start gotermsql with SQLite demo database
Type "./gotermsql --adapter sqlite --file demo.db"
Enter
Sleep 2s

# The app starts with editor focused, type a simple query
Sleep 500ms
Type "SELECT * FROM users;"
Sleep 500ms

# Dismiss autocomplete if visible, then execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Switch focus to results to see the data
Tab
Sleep 1s

# Back to editor
Tab
Sleep 500ms
Tab
Sleep 500ms

# Open a new tab for a JOIN query
Ctrl+t
Sleep 1s

# Type a single-line JOIN query
Type "SELECT u.name, o.product, o.amount FROM orders o JOIN users u ON u.id = o.user_id ORDER BY o.amount DESC;"
Sleep 500ms

# Dismiss autocomplete and execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Switch to sidebar to browse schema
Tab
Sleep 300ms
Tab
Sleep 300ms
Tab
Sleep 500ms

# Navigate the schema tree
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 500ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 500ms

# Go back to editor for one more query
Tab
Sleep 500ms

# New tab with aggregate query
Ctrl+t
Sleep 500ms

Type "SELECT u.name, COUNT(o.id) as orders, ROUND(SUM(o.amount),2) as total FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.name ORDER BY total DESC;"
Sleep 500ms

# Execute
Escape
Sleep 200ms
Ctrl+g
Sleep 3s

# Quit
Ctrl+q
Sleep 500ms
