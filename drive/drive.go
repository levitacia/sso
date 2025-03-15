package drive

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"sso/internal/repository"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type DriveService struct {
	service         *drive.Service
	credentialsPath string
	logFolderId     string
	logRepo         repository.LogRepository
}

func NewDriveService(credentialsPath, logFolderId string, logRepo repository.LogRepository) (*DriveService, error) {
	ctx := context.Background()

	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	client := config.Client(ctx)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}

	if logFolderId == "" {
		folder, err := createLogFolder(srv)
		if err != nil {
			return nil, fmt.Errorf("unable to create log folder: %v", err)
		}
		logFolderId = folder.Id
		log.Printf("Created new log folder with ID: %s", logFolderId)
	}

	return &DriveService{
		service:         srv,
		credentialsPath: credentialsPath,
		logFolderId:     logFolderId,
		logRepo:         logRepo,
	}, nil
}

func createLogFolder(srv *drive.Service) (*drive.File, error) {
	folderName := fmt.Sprintf("SSO_Logs_%s", time.Now().Format("2006-01-02"))

	folderMetadata := &drive.File{
		Name:     folderName,
		MimeType: "application/vnd.google-apps.folder",
	}

	folder, err := srv.Files.Create(folderMetadata).Do()
	if err != nil {
		return nil, err
	}

	return folder, nil
}

// func (d *DriveService) ExportLogs() error {
//     logs, err := d.logRepo.GetAllLogs()
//     if err != nil {
//         return fmt.Errorf("failed to retrieve logs: %v", err)
//     }

//     if len(logs) == 0 {
//         log.Println("No logs to export")
//         return nil
//     }
