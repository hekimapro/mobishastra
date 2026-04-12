package mobishastra

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hekimapro/mobishastra/config"
)

type SMSClient struct {
	config     *config.Config
	httpClient *http.Client
}

func NewSMSClient(cfg *config.Config) *SMSClient {
	return &SMSClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate checks if credentials are valid by checking balance
func (client *SMSClient) Authenticate() (*BalanceResponse, error) {
	return client.CheckBalance()
}

// CheckBalance returns current SMS credits balance
func (client *SMSClient) CheckBalance() (*BalanceResponse, error) {

	apiURL := fmt.Sprintf("%s?user=%s&pwd=%s", client.config.CheckBalanceURL, client.config.UserID, client.config.Password)

	response, err := client.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("balance check failed: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	balanceString := strings.TrimSpace(string(body))

	// Try to parse as float
	balance, err := strconv.ParseFloat(balanceString, 64)
	if err != nil {
		// Check for error message
		if strings.Contains(balanceString, "Invalid") {
			return &BalanceResponse{
				Success: false,
				Error:   balanceString,
			}, nil
		}
		return nil, fmt.Errorf("unexpected balance response: %s", balanceString)
	}

	return &BalanceResponse{
		Success: true,
		Balance: balance,
		Credits: int(balance),
	}, nil
}

// SendSMS sends a single SMS
func (client *SMSClient) SendSMS(phoneNumber, message string) (*SingleSMSResponse, error) {
	return client.SendSMSWithOptions(phoneNumber, message, "", "")
}

// SendSMSWithOptions sends a single SMS with additional options
func (client *SMSClient) SendSMSWithOptions(phoneNumber, message, scheduledAt, showError string) (*SingleSMSResponse, error) {

	apiURL := fmt.Sprintf("%s?user=%s&pwd=%s&senderid=%s&mobileno=%s&msgtext=%s&CountryCode=%s",
		client.config.SendSMSURL,
		client.config.UserID,
		client.config.Password,
		url.QueryEscape(client.config.SenderID),
		client.formatMobileNumber(phoneNumber),
		url.QueryEscape(message),
		client.config.CountryCode,
	)

	if client.config.Priority != "" {
		apiURL += fmt.Sprintf("&priority=%s", client.config.Priority)
	}

	if scheduledAt != "" {
		apiURL += fmt.Sprintf("&scheduledDate=%s", url.QueryEscape(scheduledAt))
	}

	if showError != "" {
		apiURL += fmt.Sprintf("&ShowError=%s", showError)
	}

	return client.executeRequest(apiURL)
}

// SendBulkSMS sends SMS to multiple numbers
func (client *SMSClient) SendBulkSMS(phoneNumbers []string, message string) (*BulkSMSResponse, error) {
	return client.SendBulkSMSWithOptions(phoneNumbers, message, "", "")
}

// SendBulkSMSWithOptions sends bulk SMS with options
func (client *SMSClient) SendBulkSMSWithOptions(phoneNumbers []string, message, scheduledAt, showError string) (*BulkSMSResponse, error) {

	if len(phoneNumbers) == 0 {
		return nil, fmt.Errorf("no mobile numbers provided")
	}

	// Format mobile numbers comma-separated
	formattedNumbers := make([]string, len(phoneNumbers))
	for i, num := range phoneNumbers {
		formattedNumbers[i] = client.formatMobileNumber(num)
	}
	mobileNoStr := strings.Join(formattedNumbers, ",")

	apiURL := fmt.Sprintf("%s?user=%s&pwd=%s&senderid=%s&mobileno=%s&msgtext=%s&CountryCode=%s",
		client.config.SendSMSCommaURL,
		client.config.UserID,
		client.config.Password,
		url.QueryEscape(client.config.SenderID),
		mobileNoStr,
		url.QueryEscape(message),
		client.config.CountryCode,
	)

	if client.config.Priority != "" {
		apiURL += fmt.Sprintf("&priority=%s", client.config.Priority)
	}

	if scheduledAt != "" {
		apiURL += fmt.Sprintf("&scheduledDate=%s", url.QueryEscape(scheduledAt))
	}

	if showError != "" {
		apiURL += fmt.Sprintf("&ShowError=%s", showError)
	}

	resp, err := client.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("bulk SMS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	responseText := strings.TrimSpace(string(body))

	bulkResp := &BulkSMSResponse{
		Success:   true,
		Responses: make([]SingleSMSResponseData, 0),
	}

	// Parse comma-separated response (if multiple)
	// Format can be: "Send Successful" or multiple comma-separated responses
	if strings.Contains(responseText, "Send Successful") {
		for _, mobileNo := range phoneNumbers {
			bulkResp.Responses = append(bulkResp.Responses, SingleSMSResponseData{
				MobileNo:     mobileNo,
				ResponseText: responseText,
				ResponseCode: "000",
				Success:      true,
			})
		}
		bulkResp.TotalSent = len(phoneNumbers)
	} else if strings.Contains(responseText, ",") {
		// Handle comma-separated responses for each number
		parts := strings.Split(responseText, ",")
		for i, part := range parts {
			if i < len(phoneNumbers) {
				success := !strings.Contains(part, "Invalid") && !strings.Contains(part, "Failed")
				code := client.extractErrorCode(part)
				bulkResp.Responses = append(bulkResp.Responses, SingleSMSResponseData{
					MobileNo:     phoneNumbers[i],
					ResponseText: strings.TrimSpace(part),
					ResponseCode: code,
					Success:      success,
				})
				if !success {
					bulkResp.Success = false
				}
			}
		}
		bulkResp.TotalSent = len(bulkResp.Responses)
	} else {
		// Single response for all numbers
		success := !strings.Contains(responseText, "Invalid") &&
			!strings.Contains(responseText, "Failed") &&
			!strings.Contains(responseText, "Blocked")

		for _, mobileNo := range phoneNumbers {
			bulkResp.Responses = append(bulkResp.Responses, SingleSMSResponseData{
				MobileNo:     mobileNo,
				ResponseText: responseText,
				ResponseCode: client.extractErrorCode(responseText),
				Success:      success,
			})
		}
		bulkResp.Success = success
		bulkResp.TotalSent = len(phoneNumbers)
	}

	return bulkResp, nil
}

// SendUnicodeSMS sends Unicode SMS (for Arabic, etc.)
func (client *SMSClient) SendUnicodeSMS(phoneNumber, message string) (*SingleSMSResponse, error) {
	apiURL := fmt.Sprintf("%s?user=%s&pwd=%s&senderid=%s&mobileno=%s&msgtext=%s&CountryCode=%s",
		client.config.SendSMSURL,
		client.config.UserID,
		client.config.Password,
		url.QueryEscape(client.config.SenderID),
		client.formatMobileNumber(phoneNumber),
		url.QueryEscape(message),
		client.config.CountryCode,
	)

	return client.executeRequest(apiURL)
}

// GetDeliveryStatus checks delivery status of a message
func (client *SMSClient) GetDeliveryStatus(messageID string) (*DeliveryStatusResponse, error) {
	apiURL := fmt.Sprintf("%s?user=%s&pwd=%s&messageid=%s",
		client.config.DeliveryStatusURL,
		client.config.UserID,
		client.config.Password,
		messageID,
	)

	resp, err := client.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("DLR status request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read DLR response: %w", err)
	}

	responseStr := strings.TrimSpace(string(body))

	// Expected format: Message_id,DLR_Status,DLR_Time
	parts := strings.Split(responseStr, ",")

	if len(parts) < 3 {
		return &DeliveryStatusResponse{
			MessageID:   messageID,
			Status:      "UNKNOWN",
			IsDelivered: false,
		}, nil
	}

	dlrTime, _ := time.Parse("2006-01-02 15:04:05", parts[2])
	if dlrTime.IsZero() {
		dlrTime = time.Now()
	}

	status := strings.ToUpper(strings.TrimSpace(parts[1]))
	isDelivered := status == "DELIVERED" || status == "DELIVRD"

	return &DeliveryStatusResponse{
		MessageID:   strings.TrimSpace(parts[0]),
		Status:      status,
		Time:        dlrTime,
		IsDelivered: isDelivered,
	}, nil
}

// GetBulkDeliveryStatus checks delivery status for multiple messages
func (client *SMSClient) GetBulkDeliveryStatus(messageIDs []string) ([]*DeliveryStatusResponse, error) {
	results := make([]*DeliveryStatusResponse, 0, len(messageIDs))

	for _, msgID := range messageIDs {
		status, err := client.GetDeliveryStatus(msgID)
		if err != nil {
			// Continue with other IDs even if one fails
			results = append(results, &DeliveryStatusResponse{
				MessageID:   msgID,
				Status:      "ERROR",
				IsDelivered: false,
			})
			continue
		}
		results = append(results, status)
	}

	return results, nil
}

// SendXMLSMS sends SMS using XML API
func (client *SMSClient) SendXMLSMS(phoneNumber, message, language string) (string, error) {
	xmlBody := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<request>
    <user>%s</user>
    <pwd>%s</pwd>
    <messages>
        <message>
            <number>%s</number>
            <msg>%s</msg>
            <sender>%s</sender>
            <language>%s</language>
        </message>
    </messages>
</request>`,
		client.config.UserID,
		client.config.Password,
		client.formatMobileNumber(phoneNumber),
		message,
		client.config.SenderID,
		language,
	)

	req, err := http.NewRequest("POST", "https://mshastra.com/sendsms_api_xml.aspx", strings.NewReader(xmlBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Helper methods

func (c *SMSClient) executeRequest(apiURL string) (*SingleSMSResponse, error) {
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("SMS request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	responseText := strings.TrimSpace(string(body))

	// Parse response
	smsResp := &SingleSMSResponse{
		ResponseText: responseText,
	}

	// Check for success
	if strings.Contains(responseText, "Send Successful") {
		smsResp.Success = true
		smsResp.ResponseCode = "000"
	} else if strings.Contains(responseText, ",") {
		// DLR format with message ID
		parts := strings.Split(responseText, ",")
		if len(parts) >= 1 {
			smsResp.MessageID = strings.TrimSpace(parts[0])
			if len(parts) >= 2 {
				smsResp.ResponseText = strings.TrimSpace(parts[1])
			}
		}
		smsResp.Success = true
	} else {
		smsResp.Success = false
		smsResp.Error = responseText
		smsResp.ResponseCode = c.extractErrorCode(responseText)
	}

	return smsResp, nil
}

func (client *SMSClient) formatMobileNumber(phoneNumber string) string {
	// Remove any spaces or special characters
	phoneNumber = strings.TrimSpace(phoneNumber)
	phoneNumber = strings.ReplaceAll(phoneNumber, " ", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, "-", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, "(", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, ")", "")
	phoneNumber = strings.ReplaceAll(phoneNumber, "+", "")

	return phoneNumber
}

func (client *SMSClient) extractErrorCode(response string) string {
	for code, msg := range ErrorCodes {
		if strings.Contains(response, msg) || strings.Contains(response, code) {
			return code
		}
	}
	return "999"
}
