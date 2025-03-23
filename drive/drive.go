package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Auth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type LogEntry struct {
	Email     string `json:"email"`
	Timestamp string `json:"timestamp"`
	Success   bool   `json:"success"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

type Config struct {
	APIBaseURL      string
	PollingInterval time.Duration
	CredentialsFile string
	FolderID        string
}

type ServiceClient struct {
	config       Config
	httpClient   *http.Client
	driveService *drive.Service
	jwtToken     string
}

func NewServiceClient(config Config) *ServiceClient {
	return &ServiceClient{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *ServiceClient) Initialize() error {
	b, err := os.ReadFile(s.config.CredentialsFile)
	if err != nil {
		return fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client := config.Client(context.Background())
	driveService, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("unable to create Drive service: %v", err)
	}

	s.driveService = driveService
	return nil
}

func (s *ServiceClient) Login() error {
	authData := Auth{
		Email:    os.Getenv("EMAIL"),
		Password: os.Getenv("PASSWORD"),
	}

	jsonData, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %v", err)
	}

	url := fmt.Sprintf("%s/api/login", s.config.APIBaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %v", err)
	}

	s.jwtToken = authResp.Token
	log.Println("Login successful")
	return nil
}

func (s *ServiceClient) FetchLogs() ([]LogEntry, error) {
	if s.jwtToken == "" {
		return nil, fmt.Errorf("not authenticated")
	}

	url := fmt.Sprintf("%s/api/protected/logs", s.config.APIBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", s.jwtToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching logs failed with status %d: %s", resp.StatusCode, string(body))
	}

	var logs []LogEntry
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode logs response: %v", err)
	}

	log.Printf("Retrieved %d log entries", len(logs))
	return logs, nil
}

func (s *ServiceClient) ConvertLogsToCSV(logs []LogEntry) ([]byte, error) {
	var buf bytes.Buffer
	csvWriter := csv.NewWriter(&buf)

	header := []string{"Email", "Timestamp", "Success", "IP", "UserAgent"}
	if err := csvWriter.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %v", err)
	}

	for _, log := range logs {
		success := "false"
		if log.Success {
			success = "true"
		}

		row := []string{
			log.Email,
			log.Timestamp,
			success,
			log.IP,
			log.UserAgent,
		}

		if err := csvWriter.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	csvWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		return nil, fmt.Errorf("error flushing CSV: %v", err)
	}

	return buf.Bytes(), nil
}

func (s *ServiceClient) UploadToGoogleDrive(csvData []byte) error {
	fileName := fmt.Sprintf("logs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))

	file := &drive.File{
		Name:     fileName,
		MimeType: "text/csv",
	}

	if s.config.FolderID != "" {
		file.Parents = []string{s.config.FolderID}
	}

	_, err := s.driveService.Files.Create(file).
		Media(bytes.NewReader(csvData)).
		Do()

	if err != nil {
		return fmt.Errorf("failed to upload file to Google Drive: %v", err)
	}

	log.Printf("Successfully uploaded %s to Google Drive", fileName)
	return nil
}

func (s *ServiceClient) StartPolling() {
	ticker := time.NewTicker(s.config.PollingInterval)
	defer ticker.Stop()

	log.Println("Starting log polling with interval:", s.config.PollingInterval)

	for range ticker.C {
		logs, err := s.FetchLogs()
		if err != nil {
			log.Printf("Error fetching logs: %v", err)
			if err.Error() == "not authenticated" {
				if err := s.Login(); err != nil {
					log.Printf("Error re-authenticating: %v", err)
				}
			}
			continue
		}

		if len(logs) == 0 {
			log.Println("No logs found, skipping upload")
			continue
		}

		csvData, err := s.ConvertLogsToCSV(logs)
		if err != nil {
			log.Printf("Error converting logs to CSV: %v", err)
			continue
		}

		if err := s.UploadToGoogleDrive(csvData); err != nil {
			log.Printf("Error uploading logs to Google Drive: %v", err)
		}
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		return
	}

	config := Config{
		APIBaseURL:      "http://localhost:7777",
		PollingInterval: 180 * time.Second,
		CredentialsFile: "credentials.json",
		FolderID:        os.Getenv("FOLDER_ID"),
	}

	client := NewServiceClient(config)

	if err := client.Initialize(); err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	if err := client.Login(); err != nil {
		log.Fatalf("Failed to login: %v", err)
	}

	client.StartPolling()
}
