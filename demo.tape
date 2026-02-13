# gotermsql Demo — SQLite
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
Type "bin/gotermsql --adapter sqlite --file demo.db"
Enter
Sleep 2s

# Editor is focused — type a simple query
Sleep 500ms
Type "SELECT * FROM users;"
Sleep 500ms

# Dismiss autocomplete, execute with Ctrl+G
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Switch to results pane (Ctrl+J cycles focus)
Ctrl+j
Sleep 1s

# Navigate result rows — see cursor highlight + alternating colors
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Show help overlay from results pane
Type "?"
Sleep 3s

# Close help
Escape
Sleep 500ms

# Back to editor (Ctrl+J: results -> editor)
Ctrl+j
Sleep 500ms

# New tab — show autocomplete for a JOIN query
Ctrl+t
Sleep 1s

# Type query, let autocomplete appear on FROM
Type "SELECT u.name, p.name AS product, p.price, o.amount FROM ord"
Sleep 800ms

# Accept autocomplete suggestion (orders)
Tab
Sleep 300ms

# Continue typing
Type " o JOIN users u ON u.id = o.user_id JOIN products p ON p.name = o.product ORDER BY p.price DESC;"
Sleep 500ms

# Execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Switch to sidebar to browse schema (Ctrl+J twice: editor -> results -> sidebar)
Ctrl+j
Sleep 300ms
Ctrl+j
Sleep 500ms

# Navigate schema tree
Down
Sleep 300ms
Enter
Sleep 500ms
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

# New tab for aggregate query
Ctrl+t
Sleep 500ms

Type "SELECT p.category, COUNT(*) AS order_count, SUM(o.amount) AS revenue FROM orders o JOIN products p ON p.name = o.product GROUP BY p.category ORDER BY revenue DESC;"
Sleep 500ms

# Execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Show results with zebra striping
Ctrl+j
Sleep 1s
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

Sleep 1s

# Quit
Ctrl+q
Sleep 500ms
