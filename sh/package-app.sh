#!/bin/bash

echo "📦 Packaging mNAV Dashboard for macOS..."

# Clean and create app bundle structure
echo "🧹 Cleaning previous app bundle..."
rm -rf mNAV.app
mkdir -p mNAV.app/Contents/MacOS
mkdir -p mNAV.app/Contents/Resources

echo "🔨 Building latest mnav-web binary..."
make mnav-web

echo "📄 Creating Info.plist..."
cat > mNAV.app/Contents/Info.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDisplayName</key>
    <string>mNAV Dashboard</string>
    <key>CFBundleExecutable</key>
    <string>mnav-launcher</string>
    <key>CFBundleIdentifier</key>
    <string>com.mnav.dashboard</string>
    <key>CFBundleName</key>
    <string>mNAV</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleSignature</key>
    <string>MNAV</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <false/>
    <key>NSRequiresAquaSystemAppearance</key>
    <false/>
</dict>
</plist>
EOF

echo "🚀 Creating launcher script..."
cat > mNAV.app/Contents/MacOS/mnav-launcher << 'EOF'
#!/bin/bash

# Get the directory where the app bundle is located
APP_DIR="$(dirname "$0")/../Resources"
cd "$APP_DIR"

echo "🚀 Starting mNAV Dashboard..."

# Check if server is already running
if lsof -i :8080 >/dev/null 2>&1; then
    echo "⚠️  Server already running on port 8080"
    echo "🌐 Opening dashboard in browser..."
    open http://localhost:8080
    exit 0
fi

# Start the web server in background
echo "📊 Starting mNAV web server..."
./mnav-web &
SERVER_PID=$!

# Wait for server to start
echo "⏳ Waiting for server to initialize..."
sleep 3

# Check if server started successfully
if ! lsof -i :8080 >/dev/null 2>&1; then
    echo "❌ Failed to start server"
    exit 1
fi

echo "✅ Server started successfully"
echo "🌐 Opening dashboard in browser..."

# Open the dashboard in default browser
open http://localhost:8080

echo "📱 mNAV Dashboard is now running at http://localhost:8080"
echo "💡 Close this terminal or press Ctrl+C to stop the server"

# Wait for the server process to finish
wait $SERVER_PID
EOF

# Make launcher executable
chmod +x mNAV.app/Contents/MacOS/mnav-launcher

echo "📁 Copying application files..."
# Copy essential files
cp bin/mnav-web mNAV.app/Contents/Resources/
cp -r sh/ mNAV.app/Contents/Resources/
cp -r data/ mNAV.app/Contents/Resources/
cp -r configs/ mNAV.app/Contents/Resources/

# Copy bin directory
mkdir -p mNAV.app/Contents/Resources/bin
cp bin/* mNAV.app/Contents/Resources/bin/

echo "✅ mNAV.app bundle created successfully!"
echo ""
echo "📱 To use your app:"
echo "   • Double-click mNAV.app to launch"
echo "   • Or run: open mNAV.app"
echo ""
echo "📦 To distribute:"
echo "   • Copy mNAV.app to /Applications"
echo "   • Or zip it: zip -r mNAV-Dashboard.zip mNAV.app"
echo ""
echo "🔄 To update: Just run this script again!" 