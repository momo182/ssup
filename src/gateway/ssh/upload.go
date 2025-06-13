package ssh

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clok/kemba"
	"github.com/momo182/ssup/src/entity"
	sc "github.com/momo182/ssup/src/gateway/shellcheck"
	"github.com/pkg/sftp"
	"github.com/pterm/pterm"
	"golang.org/x/crypto/ssh"
)

// Upload local file to remote server
func (c *RemoteClient) Upload(localPath, remotePath string) error {
	l := kemba.New("gw::sshclient::Upload").Printf
	l("uploading files with UI feedback")

	// Replace ~ with home directory.
	l("inspect if PATH starts with ~/")
	if strings.HasPrefix(remotePath, "~/") {
		l("yes, it does, removing the ~/")
		remotePath = strings.Replace(remotePath, "~/", "", 1)
	}

	fmt.Println("will upload from:", localPath)
	fmt.Println("will upload to:", remotePath)

	l("getting sftp client")
	client, err := connectSFTP(c.GetConnection())
	if err != nil {
		return err
	}
	defer client.Close()

	// Recursively upload the directory.
	return doUpload(client, localPath, remotePath, false)
}

// GenerateOnRemote basically cats file content to "~/" + entity.TASK_TAIL on remote
func (c *RemoteClient) GenerateOnRemote(data []byte, remotePath string) error {
	l := kemba.New("gw::sshclient::GenerateOnRemote").Printf
	var shellcheck entity.ShellCheckFacade
	shellcheck = &sc.ShellCheckProvider{}
	l("processing:\ndump: FC693B9D-DA60-4DA9-B783-647270E27BBC\n%s", string(shellcheck.AddNumbers(data)))
	l("uploading files without any UI feedback")
	localPath, err := os.CreateTemp("", "ssup_temp_*_data")

	defer func() {
		err := localPath.Close()
		if err != nil {
			fmt.Println("error closing localPath:", err)
		}
		err = os.Remove(localPath.Name())
		if err != nil {
			fmt.Println("error deleting localPath:", err)
		}
	}()

	if err != nil {
		return err
	}

	_, err = localPath.Write(data)
	if err != nil {
		return err
	}

	// // Replace ~ with home directory.
	// l("inspect if PATH starts with ~/")
	// if strings.HasPrefix(remotePath, "~/") {
	// 	l("yes, it does, removing the ~/")
	// 	remotePath = strings.Replace(remotePath, "~/", "", 1)
	// }

	l("will upload from:", localPath.Name())
	l("will upload to:", remotePath)

	l("getting sftp client")
	client, err := connectSFTP(c.GetConnection())
	if err != nil {
		return err
	}
	defer client.Close()

	// Recursively upload the directory.
	return doUpload(client, localPath.Name(), remotePath, true)
}

// connectSFTP creates an SSH connection and returns an SFTP client.
func connectSFTP(sshClient *ssh.Client) (*sftp.Client, error) {

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create sftp client: %v", err)
	}

	return sftpClient, nil
}

// doUpload recursively uploads a local directory to the remote SFTP server.
func doUpload(client *sftp.Client, localPath, remotePath string, silent bool) error {
	l := kemba.New("gw::sshclient::uploadDir").Printf
	return filepath.Walk(localPath, func(localFilePath string, info os.FileInfo, err error) error {
		// spinner init
		multi := pterm.DefaultMultiPrinter

		// Compute the relative path from the source directory.
		relPath, err := filepath.Rel(localPath, localFilePath)
		if err != nil {
			return err
		}
		// Use path.Join to construct the remote path (ensuring forward slashes).
		remoteFilePath := path.Join(remotePath, filepath.ToSlash(relPath))
		var spinner1 *pterm.SpinnerPrinter

		if !silent {
			spinner1, err = pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("uploading: " + remoteFilePath)
		}

		if err != nil {
			return err
		}
		multi.Start()
		defer multi.Stop()

		if info.IsDir() {
			l("inspecting if localPath is a directory")
			if err != nil {
				return err
			}

			if err := client.MkdirAll(remoteFilePath); err != nil {
				return fmt.Errorf("failed to create remote directory %s: %v", remoteFilePath, err)
			}

			if !silent {
				spinner1.Success(fmt.Sprintf("%s", remoteFilePath))
			}

		} else {
			l("inspecting if localPath is not a directory")
			// if basename == ".DS_Store" skip
			basename := filepath.Base(localFilePath)

			if !silent {
				if basename == ".DS_Store" {
					spinner1.Info(fmt.Sprintf("skipped: %s", remoteFilePath))
					return nil
				}
			}

			// Check if the remote directory exists
			baseDir := filepath.Dir(remoteFilePath)
			_, statErr := client.Stat(baseDir)
			if statErr != nil {
				l("remote directory does not exist, creating it")
				client.MkdirAll(baseDir)
			}

			// Open local file.
			localFile, err := os.Open(localFilePath)
			if err != nil {
				return fmt.Errorf("failed to open local file %s: %v", localFilePath, err)
			}
			defer localFile.Close()

			// Create remote file.
			remoteFile, err := client.Create(remoteFilePath)
			if err != nil {
				return fmt.Errorf("failed to create remote file %s: %v", remoteFilePath, err)
			}
			defer remoteFile.Close()

			// Copy local file to remote file using io.Copy.
			if _, err := io.Copy(remoteFile, localFile); err != nil {
				return fmt.Errorf("failed to copy to remote file %s: %v", remoteFilePath, err)
			}
			if !silent {
				spinner1.Success(fmt.Sprintf("%s", remoteFilePath))
			}
		}
		return nil
	})
}
