package sftpclient

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/fusor/cpma/env"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	kh "golang.org/x/crypto/ssh/knownhosts"
)

// Client Wrapper around sftp.Client
type Client struct {
	*sftp.Client
}

var globalClient Client

// GlobalClient returns global client
func GlobalClient() Client {
	return globalClient
}

// NewClient creates a new SFTP client
func NewClient() {
	source := env.Config().GetString("Source")
	sshCreds := env.Config().GetStringMapString("SSHCreds")

	key, err := ioutil.ReadFile(sshCreds["privatekey"])
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	hostKeyCallback, err := kh.New(filepath.Join(env.Config().GetString("home"), ".ssh", "known_hosts"))
	if err != nil {
		log.Fatal(err)
	}

	sshConfig := &ssh.ClientConfig{
		User: sshCreds["login"],
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,

		Timeout: 10 * time.Second,
	}

	port := sshCreds["port"]

	log.Debugln("Using port: ", port)

	if port == "" {
		port = "22"
	}

	addr := fmt.Sprintf("%s:%s", source, port)

	connection, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatal(err)
	}

	// create new SFTP client
	client, err := sftp.NewClient(connection)
	if err != nil {
		log.Fatal(err)
	}

	globalClient = Client{client}
}

// GetFile copies source file to destination file
func (c *Client) GetFile(srcFilePath string, dstFilePath string) {
	srcFile, err := c.Open(srcFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer srcFile.Close()

	os.MkdirAll(path.Dir(dstFilePath), 0755)
	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("File %s: %d bytes copied\n", srcFilePath, bytes)
}
