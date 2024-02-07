package send

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpServer struct {
	server   string
	port     uint32
	username string
	password string
}

func NewSftp(server string, port uint32, username string, password string) *SftpServer {
	return &SftpServer{
		server:   server,
		port:     port,
		username: username,
		password: password,
	}
}

func (sftp_process *SftpServer) Start() {

	// Remote directory on the SFTP server
	remoteDir := "/path/on/remote/server"

	// Connect to the SSH server
	config := &ssh.ClientConfig{
		User: sftp_process.username,
		Auth: []ssh.AuthMethod{
			ssh.Password(sftp_process.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sftp_process.server, sftp_process.port), config)
	if err != nil {
		panic(fmt.Errorf("Failed to dial SSH server: %v", err))
	}
	defer sshClient.Close()

	// Create an SFTP session
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		panic(fmt.Errorf("Failed to create SFTP client: %v", err))
	}
	defer sftpClient.Close()

	// List current directory contents
	localDir := "."
	files, err := os.ReadDir(localDir)
	if err != nil {
		panic(fmt.Errorf("Failed to list directory contents: %v", err))
	}

	fmt.Println("Local Files:")
	for _, file := range files {
		fmt.Println(file.Name())
	}

	// Prompt user to select a file
	fmt.Println("Enter the name of the file you want to upload:")
	var filename string
	fmt.Scanln(&filename)

	// Open the local file
	localFilePath := filepath.Join(localDir, filename)
	localFile, err := os.Open(localFilePath)
	if err != nil {
		panic(fmt.Errorf("Failed to open local file: %v", err))
	}
	defer localFile.Close()

	// Create the remote file
	remoteFile, err := sftpClient.Create(remoteDir + "/" + filename)
	if err != nil {
		panic(fmt.Errorf("Failed to create remote file: %v", err))
	}
	defer remoteFile.Close()

	// Copy the contents of the local file to the remote file
	bytes, err := io.Copy(remoteFile, localFile)
	if err != nil {
		panic(fmt.Errorf("Failed to copy file contents: %v", err))
	}
	fmt.Printf("File uploaded successfully. %d bytes transferred.\n", bytes)
}
