// utils is the package that defines the various subroutines used by Big Disk API. they are all functions.
package utils

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//Call is used to execute complex pipes to filter out the wlan0 IP address of the Pi via ifconfig, awk, and cut
// It is called by Execute once organizes all the separate components of the pipe command
func Call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {
	if stack[0].Process == nil {
		if err = stack[0].Start(); err != nil {
			return err
		}
	}
	if len(stack) > 1 {
		if err = stack[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				pipes[0].Close()
				err = Call(stack[1:], pipes[1:])
			}
		}()
	}
	return stack[0].Wait()
}

//Execute is used to execute complex pipes
func Execute(output_buffer *bytes.Buffer, stack ...*exec.Cmd) (err error) {
	var error_buffer bytes.Buffer
	pipe_stack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &error_buffer
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = output_buffer
	stack[i].Stderr = &error_buffer

	if err := Call(stack, pipe_stack); err != nil {
		log.Fatalln(string(error_buffer.Bytes()), err)
	}
	return err
}

// Pwd finds the directory of the main process (which would be ../) so that Prometheus can find ../public
// Mainly, this is necessary so that Prometheus can be started in rc.local. The directory becomes relative to the root when started as a startup process. Hence, the ./public folder will no longer be locatable through relative positioning. Pwd ensures you don't have to hardcode the path of the program directory.
func Pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir + "/"
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

func ReadFile(filename string) string {
	content, err := ioutil.ReadFile(Pwd() + filename)
	if err != nil {
		log.Println("ERROR ReadFile()")
	}
	lines := strings.Split(string(content), "\n")
	return lines[0]
}

func GetIP() string {
	var b bytes.Buffer
	var str string
	if err := Execute(&b,
		exec.Command("ifconfig", ReadFile("interface")),
		exec.Command("grep", "inet"),
		exec.Command("awk", "NR==1{print $2}"),
	); err != nil {
		log.Fatalln(err)
	}
	str = b.String()
	regex, err := regexp.Compile("\n")
	if err != nil {
		log.Println("ERROR GetIP()")
	}
	str = regex.ReplaceAllString(str, "")
	return strings.TrimSpace(str)
}
