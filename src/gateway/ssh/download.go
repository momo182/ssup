package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/clok/kemba"
	"github.com/pkg/sftp"
	"github.com/pterm/pterm"
)

// Download file from remote
func (c *RemoteClient) Download(remotePath, localPath string, silent bool, flat bool) error {
	l := kemba.New("gw::ssh::download").Printf
	l("will download %s to local:%s", remotePath, localPath)

	// Handle ~ for local paths
	if strings.HasPrefix(remotePath, "~/") {
		homeDir := c.GetInventory().Home
		remotePath = filepath.Join(homeDir, remotePath[2:])
	}

	// Handle ~ for remote paths
	if strings.HasPrefix(localPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		homeDir := home
		localPath = filepath.Join(homeDir, localPath[2:])
	}

	l("Will download from:", remotePath)
	l("Will download to:", localPath)

	if !silent {
		fmt.Println("+ download:")
		fmt.Println("  src:", remotePath)
		fmt.Println("  dst:", localPath)
	}

	if !flat {
		localPath = localPath + fmt.Sprintf("@%s", c.GetHost())
	}

	client, err := connectSFTP(c.GetConnection())
	if err != nil {
		return err
	}
	// defer client.Close()

	// Recursively download the directory.
	return downloadDir(client, remotePath, localPath)

}

// downloadDir recursively downloads a remote directory to the local machine.
func downloadDir(client *sftp.Client, remotePath, localPath string) error {
	walker := client.Walk(remotePath)

	for walker.Step() {
		if walker.Err() != nil {
			return walker.Err()
		}

		remoteFilePath := walker.Path()
		relPath, err := filepath.Rel(remotePath, remoteFilePath)
		if err != nil {
			return err
		}
		localFilePath := filepath.Join(localPath, relPath)

		spinner, _ := pterm.DefaultSpinner.Start("Processing: " + remoteFilePath)

		if walker.Stat().IsDir() {
			if err := os.MkdirAll(localFilePath, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create local directory %s: %v", localFilePath, err)
			}
			spinner.Success(localFilePath)
		} else {
			// Skip .DS_Store files
			if filepath.Base(remoteFilePath) == ".DS_Store" {
				spinner.Info("Skipped: " + remoteFilePath)
				continue
			}

			remoteFileSize := walker.Stat().Size()

			// Check if local file exists and compare size
			if localInfo, err := os.Stat(localFilePath); err == nil {
				if localInfo.Size() == remoteFileSize {
					spinner.Success("Already up-to-date (same size): " + localFilePath)
					continue
				}
			}

			// Open remote file
			remoteFile, err := client.Open(remoteFilePath)
			if err != nil {
				return fmt.Errorf("failed to open remote file %s: %v", remoteFilePath, err)
			}
			defer remoteFile.Close()

			// Create local file
			localFile, err := os.Create(localFilePath)
			if err != nil {
				return fmt.Errorf("failed to create local file %s: %v", localFilePath, err)
			}
			defer localFile.Close()

			// Copy remote file to local file
			if _, err := io.Copy(localFile, remoteFile); err != nil {
				return fmt.Errorf("failed to copy remote file %s: %v", remoteFilePath, err)
			}

			spinner.Success(localFilePath)
		}
	}
	return nil
}
