# Mobishastra SMS Client for Go

A robust Go client library for Mobishastra SMS Gateway API. This package provides easy-to-use methods for sending single and bulk SMS, checking balance, tracking delivery status, and handling callbacks.

## Features

- ✅ Send single SMS messages
- 📱 Send bulk SMS to multiple recipients
- 💰 Check account balance
- 📊 Track delivery status (DLR)
- 🔄 Support for scheduled messages
- 🌍 Unicode support for international languages
- 📞 Callback handling for incoming SMS and DLR
- 🔐 Environment-based configuration
- 🛡️ Built-in error handling and status mapping

## Installation

```bash
go get github.com/hekimapro/mobishastra
```

## Configuration

The package uses environment variables for configuration. Create a `.env` file in your project root or set system environment variables.

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SMS_USER_ID` | Your SMS gateway username | Yes | - |
| `SMS_PASSWORD` | Your SMS gateway password | Yes | - |
| `SMS_SENDER_ID` | Sender ID (max 13 characters) | Yes | - |
| `SMS_SEND_URL` | SMS sending endpoint | No | `https://mshastra.com/sendurl.aspx` |
| `SMS_SEND_COMM_URL` | Bulk SMS endpoint | No | `https://mshastra.com/sendurlcomma.aspx` |
| `SMS_BALANCE_URL` | Balance check endpoint | No | `https://mshastra.com/balance.aspx` |
| `SMS_DELIVERY_STATUS_URL` | DLR status endpoint | No | `http://login.telkosh.com/dlrstatus_api.aspx` |
| `SMS_CALLBACK_SECRET` | Secret key for callback authentication | No | - |
| `SMS_CALLBACK_PORT` | Port for callback server | No | - |
| `SMS_COUNTRY_CODE` | Default country code | No | `ALL` |
| `SMS_PRIORITY` | Default message priority | No | `High` |

### Example .env file

```env
SMS_USER_ID=your_username
SMS_PASSWORD=your_password
SMS_SENDER_ID=YOURSMS
SMS_COUNTRY_CODE=ALL
SMS_PRIORITY=High
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/hekimapro/mobishastra"
    "github.com/hekimapro/mobishastra/config"
)

func main() {
    // Load configuration
    cfg := config.LoadConfig()

    // Create SMS client
    client := mobishastra.NewSMSClient(cfg)

    // Authenticate and check balance
    balance, err := client.Authenticate()
    if err != nil {
        log.Fatal("Authentication failed:", err)
    }
    fmt.Printf("Balance: %.2f credits\n", balance.Balance)

    // Send a single SMS
    response, err := client.SendSMS("1234567890", "Hello from Go!")
    if err != nil {
        log.Fatal("Failed to send SMS:", err)
    }

    if response.Success {
        fmt.Println("SMS sent successfully!")
        fmt.Printf("Message ID: %s\n", response.MessageID)
    } else {
        fmt.Printf("Failed to send: %s\n", response.Error)
    }
}
```

## API Reference

### Types

#### Config
Configuration structure containing API credentials and endpoints.

```go
type Config struct {
    UserID            string
    Password          string
    SenderID          string
    SendSMSURL        string
    SendSMSCommaURL   string
    CheckBalanceURL   string
    DeliveryStatusURL string
    CallbackSecret    string
    CallbackPort      string
    CountryCode       string
    Priority          string
}
```

#### SMSClient
Main client for interacting with the SMS gateway.

```go
type SMSClient struct {
    config     *config.Config
    httpClient *http.Client
}
```

#### Response Types

```go
// Single SMS response
type SingleSMSResponse struct {
    Success      bool
    MessageID    string
    ResponseCode string
    ResponseText string
    Error        string
}

// Bulk SMS response
type BulkSMSResponse struct {
    Success   bool
    Responses []SingleSMSResponseData
    TotalSent int
}

// Balance response
type BalanceResponse struct {
    Success bool
    Balance float64
    Credits int
    Error   string
}

// Delivery status response
type DeliveryStatusResponse struct {
    MessageID   string
    Status      string
    Time        time.Time
    IsDelivered bool
}
```

### Methods

#### Client Initialization

##### `NewSMSClient(cfg *config.Config) *SMSClient`
Creates a new SMS client instance with the provided configuration.

#### Authentication & Balance

##### `Authenticate() (*BalanceResponse, error)`
Authenticates the user by checking account balance. Returns balance information or error.

##### `CheckBalance() (*BalanceResponse, error)`
Returns current SMS credits balance.

#### Sending Messages

##### `SendSMS(phoneNumber, message string) (*SingleSMSResponse, error)`
Sends a single SMS message using default configuration.

##### `SendSMSWithOptions(phoneNumber, message, scheduledAt, showError string) (*SingleSMSResponse, error)`
Sends a single SMS with additional options:
- `scheduledAt`: Schedule delivery time (format depends on API)
- `showError`: Show detailed error information

##### `SendBulkSMS(phoneNumbers []string, message string) (*BulkSMSResponse, error)`
Sends the same message to multiple recipients.

##### `SendBulkSMSWithOptions(phoneNumbers []string, message, scheduledAt, showError string) (*BulkSMSResponse, error)`
Sends bulk SMS with scheduling and error options.

##### `SendUnicodeSMS(phoneNumber, message string) (*SingleSMSResponse, error)`
Sends Unicode SMS messages (supports Arabic, Chinese, etc.).

##### `SendXMLSMS(phoneNumber, message, language string) (string, error)`
Sends SMS using XML API with language specification.

#### Delivery Status

##### `GetDeliveryStatus(messageID string) (*DeliveryStatusResponse, error)`
Checks delivery status for a single message.

##### `GetBulkDeliveryStatus(messageIDs []string) ([]*DeliveryStatusResponse, error)`
Checks delivery status for multiple messages.

## Usage Examples

### Sending SMS with Options

```go
// Send scheduled SMS
scheduledTime := "2024-12-25 10:00:00"
response, err := client.SendSMSWithOptions(
    "1234567890",
    "Merry Christmas!",
    scheduledTime,
    "1",
)

// Send with custom priority
client.config.Priority = "High"
response, err := client.SendSMS("1234567890", "Important message")
```

### Bulk SMS with Detailed Response

```go
phoneNumbers := []string{
    "1234567890",
    "0987654321",
    "5551234567",
}

response, err := client.SendBulkSMS(phoneNumbers, "Welcome to our service!")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Total sent: %d/%d\n", response.TotalSent, len(phoneNumbers))
for _, resp := range response.Responses {
    if resp.Success {
        fmt.Printf("✓ %s: Sent (ID: %s)\n", resp.MobileNo, resp.MessageID)
    } else {
        fmt.Printf("✗ %s: Failed - %s\n", resp.MobileNo, resp.ResponseText)
    }
}
```

### Tracking Delivery Status

```go
// After sending SMS, track delivery
messageID := "1234567890"
time.Sleep(5 * time.Second) // Wait for processing

status, err := client.GetDeliveryStatus(messageID)
if err != nil {
    log.Fatal(err)
}

if status.IsDelivered {
    fmt.Printf("Message %s delivered at %s\n", status.MessageID, status.Time)
} else {
    fmt.Printf("Message %s status: %s\n", status.MessageID, status.Status)
}
```

### Checking Balance

```go
balance, err := client.CheckBalance()
if err != nil {
    log.Fatal("Balance check failed:", err)
}

if balance.Success {
    fmt.Printf("Available credits: %d\n", balance.Credits)
    fmt.Printf("Balance: %.2f\n", balance.Balance)
} else {
    fmt.Printf("Error: %s\n", balance.Error)
}
```

### Handling Callbacks

```go
package main

import (
    "encoding/json"
    "net/http"

    "github.com/hekimapro/mobishastra"
)

func handleInboundSMS(w http.ResponseWriter, r *http.Request) {
    var callback mobishastra.InboundSMSCallback

    if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Process incoming message
    fmt.Printf("Received from %s: %s\n", callback.PhoneNumber, callback.Message)

    // Auto-reply or process as needed
    w.WriteHeader(http.StatusOK)
}

func handleDeliveryReport(w http.ResponseWriter, r *http.Request) {
    var dlr mobishastra.DeliveryStatusCallback

    if err := json.NewDecoder(r.Body).Decode(&dlr); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    fmt.Printf("Delivery update - ID: %s, Status: %s\n", dlr.MessageID, dlr.Status)
    w.WriteHeader(http.StatusOK)
}

func main() {
    cfg := config.LoadConfig()

    http.HandleFunc("/sms/callback", handleInboundSMS)
    http.HandleFunc("/dlr/callback", handleDeliveryReport)

    port := cfg.CallbackPort
    if port == "" {
        port = "8080"
    }

    log.Printf("Starting callback server on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Error Handling

The package includes predefined error codes mapping:

```go
var ErrorCodes = map[string]string{
    "000": "Send Successful",
    "001": "Invalid Receiver",
    "003": "Invalid Message",
    "005": "Authorization failed",
    "006": "DND Number",
    "007": "Cannot Extract Country Code",
    "008": "Empty Receiver",
    "009": "Profile Blocked",
    "010": "Invalid Profile ID",
    "011": "Profile ID expired",
    "012": "Sender Id more than 13 Chars",
    "013": "Server Error",
}
```

### Error Handling Example

```go
response, err := client.SendSMS(phoneNumber, message)
if err != nil {
    // Network or connection error
    log.Printf("Connection error: %v", err)
    return
}

if !response.Success {
    // API error
    errorMsg, exists := mobishastra.ErrorCodes[response.ResponseCode]
    if exists {
        log.Printf("SMS failed: %s (Code: %s)", errorMsg, response.ResponseCode)
    } else {
        log.Printf("SMS failed: %s", response.Error)
    }
}
```

## Best Practices

1. **Environment Variables**: Always use environment variables for sensitive credentials. Never hardcode them.

2. **Error Handling**: Always check both the error return value and the `Success` field in responses.

3. **Rate Limiting**: Implement rate limiting to avoid overwhelming the API.

4. **Connection Timeout**: The client uses a 30-second timeout by default. Adjust if needed:

```go
client := &SMSClient{
    config: cfg,
    httpClient: &http.Client{
        Timeout: 60 * time.Second,
    },
}
```

5. **Phone Number Formatting**: The client automatically formats phone numbers, but ensure numbers include country codes when needed.

## Testing

```go
func TestSendSMS(t *testing.T) {
    cfg := &config.Config{
        UserID:   "test_user",
        Password: "test_pass",
        SenderID: "TEST",
    }

    client := NewSMSClient(cfg)

    // Mock HTTP client for testing
    // ... test implementation
}
```

## Contributing

Contributions are welcome! Please submit pull requests or create issues for bugs and feature requests.

## License

This project is licensed under the MIT License.

## Support

For API-specific issues, contact Mobishastra support. For package issues, open an issue on GitHub.

---

**Note**: This package is community-maintained and not officially affiliated with Mobishastra. Always test thoroughly before using in production.