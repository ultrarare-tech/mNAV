# mNAV Web Dashboard

A modern web interface for real-time mNAV tracking and portfolio analysis.

## 🚀 Quick Start

### 1. Build the Web Server
```bash
make mnav-web
```

### 2. Start the Web Dashboard
```bash
# From the project root directory
./bin/mnav-web
```

### 3. Open in Browser
Navigate to: **http://localhost:8080**

## 📊 Features

### **Real-Time Data Updates**
- 🔄 **Update Button**: Click to fetch latest market data
- ⚡ **Live Parsing**: Extracts key metrics from your update script
- 🎯 **Auto-Refresh**: Updates every 5 minutes automatically

### **Key Metrics Display**
- 💰 **Bitcoin Price**: Current market price
- 📈 **MSTR Stock Price**: Real-time stock price
- 💵 **FBTC Price**: Fidelity Bitcoin ETF price
- 📊 **Premium to NAV**: Current premium percentage
- 📈 **mNAV Ratio**: Net Asset Value ratio

### **Comprehensive Data View**
- 🪙 **Bitcoin Holdings**: MSTR's current BTC holdings
- 💰 **Bitcoin Value**: Total value of holdings
- 📡 **Data Sources**: Shows what data came from which source
- 📝 **Raw Output**: Full script output (collapsible)

### **Modern UI**
- 🎨 **Responsive Design**: Works on desktop and mobile
- 🌈 **Color-Coded Metrics**: Easy to read at a glance
- 📱 **Mobile Friendly**: Touch-optimized interface
- ⚡ **Fast Loading**: Lightweight and efficient

## 🛠️ Technical Details

### **Server Architecture**
- **Language**: Go
- **Framework**: Standard HTTP library
- **Port**: 8080 (configurable)
- **API**: RESTful JSON endpoints

### **API Endpoints**
- `GET /` - Serves the HTML dashboard
- `POST /api/update` - Executes update script and returns JSON

### **Data Flow**
1. User clicks "Update Data" button
2. JavaScript makes POST request to `/api/update`
3. Go server executes `./sh/update-mnav` script
4. Server parses output and returns structured JSON
5. Frontend updates display with new data

### **Script Integration**
The web interface automatically:
- ✅ Executes your existing `sh/update-mnav` script
- ✅ Parses all key metrics from the output
- ✅ Displays data in an organized format
- ✅ Shows raw output for debugging

## 🔧 Configuration

### **Running on Different Port**
```go
// Edit cmd/utilities/mnav-web/main.go
port := ":8081"  // Change from :8080
```

### **Customizing Auto-Refresh**
```javascript
// Edit the setInterval in the HTML template
setInterval(() => {
    if (currentData) {
        runUpdate();
    }
}, 10 * 60 * 1000);  // 10 minutes instead of 5
```

## 🚨 Prerequisites

1. **Project Setup**: Must run from project root directory
2. **Script Access**: `sh/update-mnav` must be executable
3. **Dependencies**: All your existing API keys and tools

## 🎯 Usage Examples

### **Manual Updates**
1. Open http://localhost:8080
2. Click "🔄 Update Data" button
3. View results in real-time

### **Background Monitoring**
- Leave browser tab open
- Auto-refreshes every 5 minutes
- Shows loading spinner during updates

### **Mobile Access**
- Access from phone/tablet using computer's IP
- Example: `http://192.168.1.100:8080`
- Responsive design adapts to screen size

## 🔍 Troubleshooting

### **"Could not find update-mnav script"**
- Ensure you're running from project root directory
- Check that `sh/update-mnav` exists and is executable

### **"Update failed"**
- Check that all API keys are properly configured
- Verify that required binaries are built (`make all`)
- View raw output section for detailed error information

### **Port Already in Use**
```bash
# Find what's using port 8080
lsof -i :8080

# Kill the process
pkill mnav-web
```

## 🎨 Screenshots

The dashboard features:
- **Header**: Gradient background with project branding
- **Update Section**: Large, prominent update button
- **Metrics Grid**: Card-based layout for key numbers
- **Data Tables**: Organized display of detailed information
- **Raw Output**: Collapsible section for debugging

## 🔄 Integration with Existing Workflow

The web interface **enhances** your existing setup:
- ✅ **No Changes Required**: Uses your existing script as-is
- ✅ **Same Data**: Shows identical results to command line
- ✅ **All Features**: Accesses portfolio, analytics, everything
- ✅ **Backwards Compatible**: Command line still works normally

## 🚀 Future Enhancements

Potential improvements:
- 📊 **Interactive Charts**: Real-time plotting
- 🔔 **Alerts**: Email/SMS notifications
- 📱 **PWA**: Install as app on phone
- 🎛️ **Settings**: Configurable refresh intervals
- 📈 **Historical**: Chart historical data
- 🔒 **Authentication**: Secure access controls

---

**Happy Tracking!** 🚀📊💰 