# gotermsql Demo — PostgreSQL
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

# Start gotermsql with PostgreSQL
Type "bin/gotermsql postgres://localhost:5432/gotermsql_test?sslmode=disable"
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

# Move to results to see the data
Tab
Sleep 1s

# Navigate result rows
Down
Sleep 200ms
Down
Sleep 200ms
Down
Sleep 200ms

# Back to editor via Tab cycle (results -> sidebar -> editor)
Tab
Sleep 500ms
Tab
Sleep 500ms

# New tab — show autocomplete for a JOIN query
Ctrl+t
Sleep 1s

# Type query, let autocomplete appear on FROM
Type "SELECT u.name, p.name AS product, p.price, o.quantity FROM ord"
Sleep 800ms

# Accept autocomplete suggestion (orders)
Tab
Sleep 300ms

# Continue typing
Type " o JOIN users u ON u.id = o.user_id JOIN products p ON p.id = o.product_id ORDER BY p.price DESC;"
Sleep 500ms

# Execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Switch to sidebar to browse schema (Tab from editor -> results -> sidebar)
Tab
Sleep 300ms
Tab
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

# Show help overlay (? works when sidebar is focused)
Type "?"
Sleep 3s

# Close help
Escape
Sleep 500ms

# Back to editor for aggregate query
Tab
Sleep 500ms
Ctrl+t
Sleep 500ms

Type "SELECT p.category, COUNT(*) AS order_count, SUM(o.quantity * p.price) AS revenue FROM orders o JOIN products p ON p.id = o.product_id GROUP BY p.category ORDER BY revenue DESC;"
Sleep 500ms

# Execute
Escape
Sleep 200ms
Ctrl+g
Sleep 2s

# Quit
Ctrl+q
Sleep 500ms
