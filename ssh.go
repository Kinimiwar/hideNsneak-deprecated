package main

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os/exec"
	"fmt"
	"os"
	"bufio"
	"bytes"
	"strings"
)

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading public key file")
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		fmt.Println("Error parsing public key")
		return nil
	}
	return ssh.PublicKeys(key)
}

func setHomeDirs(allInstances map[int]*CloudInstance) map[int]*CloudInstance{
	for _,instance := range allInstances {
		sshConfig := &ssh.ClientConfig{
			User: instance.Username,
			Auth: []ssh.AuthMethod{
				PublicKeyFile(instance.PrivateKey),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		workingDir := instance.executeCmd("pwd", sshConfig)
		instance.HomeDir = strings.TrimSpace(workingDir)
	}
	return allInstances
}

func (instance CloudInstance) scpFileToHost(file string, targetDir string) {
	fmt.Println(instance.PrivateKey)
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", instance.PrivateKey, file ,instance.Username + "@" + instance.IPv4 + ":" + targetDir )
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
	}
}

func (instance CloudInstance) scpFileFromHost(file string, targetDir string) {
	fmt.Println(instance.PrivateKey)
	command := exec.Command("scp", "-o", "StrictHostKeyChecking=no", "-i", instance.PrivateKey ,instance.Username + "@" + instance.IPv4 + ":" + file, targetDir)
	if err := command.Run(); err != nil {
		fmt.Println("SCPfile failed")
		fmt.Println(err)
	}
}

func (instance CloudInstance) rsyncDirToHost(dir string, targetDir string) {
	command := exec.Command("rsync", "-azu", "-e", "'ssh", "-o", "StrictHostKeyChecking=no", "-i", instance.PrivateKey, "-l", instance.Username+"'",dir, instance.IPv4 + ":" + targetDir )
	if err := command.Start(); err != nil {
		fmt.Println("SCPDir failed")
		fmt.Println(err)
	}
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
  }

func (instance CloudInstance) rsyncDirFromHost(dir string, targetDir string) {
	rsyncCommand := "rsync -azu -e 'ssh -o StrictHostKeyChecking=no -i "+instance.PrivateKey + " -l " + instance.Username + "' " + instance.IPv4 + ":" + dir + " " + targetDir 
	command := exec.Command("bash", "-c", rsyncCommand)
	command.Stderr = os.Stderr
	printCommand(command)
	if err := command.Start(); err != nil {
		fmt.Println("Rsync Dir failed")
		fmt.Println(err)
	}
}


func (instance CloudInstance) shellSystem() {
	sshConfig := &ssh.ClientConfig{
		User: instance.Username,
		Auth: []ssh.AuthMethod{
			PublicKeyFile(instance.PrivateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := ssh.Dial("tcp", instance.IPv4+":22", sshConfig)
	if err != nil {
		fmt.Printf("dial failed:%v", err)
		return
	}
	session, err := conn.NewSession()
	if err != nil {
		fmt.Printf("session failed:%v", err)
		return
	}
	defer session.Close()

	// Set IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	in, _ := session.StdinPipe()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		fmt.Printf("request for pseudo terminal failed: %s", err)
		return
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		fmt.Printf("failed to start shell: %s", err)
		return
	}

	// Accepting commands
	for {
		fmt.Println("got here")
		reader := bufio.NewReader(os.Stdin)
		str, _ := reader.ReadString('\n')
		fmt.Println(str)
		fmt.Println(str)
		if str == "quit\n" {
			return
		}
		fmt.Fprint(in, str)
	}
}

func (instance CloudInstance) executeCmd(cmd string, config *ssh.ClientConfig) string{
    conn, _ := ssh.Dial("tcp", instance.IPv4+":22", config)
    session, _ := conn.NewSession()
    defer session.Close()

    var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Run(cmd)

    return stdoutBuf.String()
}

func (instance CloudInstance) executeBackgroundCmd(cmd string, config *ssh.ClientConfig) string{
    conn, _ := ssh.Dial("tcp", instance.IPv4+":22", config)
    session, _ := conn.NewSession()
    defer session.Close()

    var stdoutBuf bytes.Buffer
    session.Stdout = &stdoutBuf
    session.Start(cmd)

    return stdoutBuf.String()
}