package internal

import (
	"bufio"
	"fmt"
	"github.com/ilius/crock32"
	"github.com/jpillora/backoff"
	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multihash"

	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BackoffDuration is
func BackoffDuration() func(error) {
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	return func(rc error) {
		secs := b.Duration()

		fmt.Printf("rc: %v sleeping %v\n", rc, secs)
		time.Sleep(secs)
		if secs.Nanoseconds() >= b.Max.Nanoseconds() {
			b.Reset()
		}
	}
}

// FreePort is
func FreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// CurrentTime is
func CurrentTime() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// IsPeer checks if the string s ends in a valid b32-encoded peer address or b58-encoded ID
func IsPeer(s string) bool {
	sa := strings.Split(s, ".")
	le := len(sa) - 1
	id := ToPeerID(sa[le])
	return id != ""
}

// ToPeerID returns b58-encoded ID. it converts to b58 if b32-encoded.
func ToPeerID(s string) string {
	m, err := multihash.FromB58String(s)
	if err == nil {
		return m.B58String()
	}

	c, err := crock32.Decode(s)
	if err != nil {
		return ""
	}

	m, err = multihash.Cast(c)
	if err == nil {
		return m.B58String()
	}

	return ""
}

// ToPeerAddr returns b32-encoded ID. it converts to b32 if B58-encoded.
func ToPeerAddr(s string) string {
	m, err := multihash.FromB58String(s)
	if err == nil {
		return strings.ToLower(crock32.Encode(m))
	}

	//normalize/validate
	d, err := crock32.Decode(s)
	if err == nil {
		m, err = multihash.Cast(d)
		if err != nil {
			return ""
		}
		return strings.ToLower(crock32.Encode(m))
	}

	return ""
}

// ParseInt parses s into int
func ParseInt(s string, v int) int {
	if s == "" {
		return v
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		i = v
	}
	return i
}

// TLD returns last part of a domain name
func TLD(name string) string {
	sa := strings.Split(name, ".")
	s := sa[len(sa)-1]

	return s
}

// Alias returns the second last part of a domain name ending in .a
// or error if not an alias
func Alias(name string) (string, error) {
	if name != "a" && !strings.HasSuffix(name, ".a") {
		return "", fmt.Errorf("Not an alias: %v", name)
	}
	sa := strings.Split(name, ".")
	if len(sa) == 1 {
		return "", nil
	}
	s := sa[0 : len(sa)-1]

	return strings.Join(s, "."), nil
}

var localHostIpv4RE = regexp.MustCompile(`127\.0\.0\.\d+`)

// IsLocalHost checks whether host is explicitly local host
// taken from goproxy
func IsLocalHost(host string) bool {
	return host == "::1" ||
		host == "0:0:0:0:0:0:0:1" ||
		localHostIpv4RE.MatchString(host) ||
		host == "localhost"
}

var localHomeRE = regexp.MustCompile(`.*\.?home`)

// IsHome checks whether host is explicitly local home node
func IsHome(host string) bool {
	return host == "home" ||
		localHomeRE.MatchString(host)
}

// // ConvertTLD changes tld to home
// func ConvertTLD(host string) (string, string) {
// 	sa := strings.Split(host, ".")
// 	le := len(sa)
// 	tld := sa[le-1]
// 	sa[le-1] = "home"
// 	return strings.Join(sa, "."), tld
// }

// GetDefaultBase returns $DHNT_BASE or $HOME/dhnt if not found
func GetDefaultBase() string {
	return getBase()
}

func getBase() string {
	base := os.Getenv("DHNT_BASE")
	if base != "" {
		return base
	}
	home, err := homedir.Dir()
	if err != nil {
		base = fmt.Sprintf("%v/dhnt", home)
	}

	return getBaseFromExe()
}

func getBaseFromExe() string {
	// dhnt/go/bin/m3d
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	for {
		dir, file := filepath.Split(dir)
		if file == "dhnt" {
			return filepath.Join(dir, file)
		}
		if dir == "" {
			break
		}
	}
	return ""
}

// GetDefaultPort returns $M3_PORT or 18080 if not found
func GetDefaultPort() int {
	if p := os.Getenv("M3_PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			return port
		}
	}
	return 18080
}

// GetDaemonPort returns $M3_PORT or 18080 if not found
func GetDaemonPort() int {
	if p := os.Getenv("M3D_PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			return port
		}
	}
	return 18082
}

// GetIntEnv returns int env or default
func GetIntEnv(env string, i int) int {
	if p := os.Getenv(env); p != "" {
		if i, err := strconv.Atoi(p); err == nil {
			return i
		}
	}
	return i
}

func ToTimestamp(d time.Time) int64 {
	return d.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// // SetDefaultPath sets required env
// func SetDefaultEnv() {
// 	base := getBase()
// 	gopath := fmt.Sprintf("%v/go", base)
// 	gogsworkdir := fmt.Sprintf("%v/var/gogs", base)

// 	os.Setenv("DHNT_BASE", base)
// 	os.Setenv("GOPATH", gopath)
// 	os.Setenv("GOGS_WORK_DIR", gogsworkdir)

// 	AddPath([]string{
// 		fmt.Sprintf("%v/bin", gopath),
// 		fmt.Sprintf("%v/bin", base),
// 	})
// }

// Execute sets up env and runs file
func Execute(base, file string) error {
	// binary, err := exec.LookPath(file)
	// if err != nil {
	// 	return err
	// }
	binary := filepath.Join(base, file)
	cmd := exec.Command(binary)

	//TODO template?
	el := []string{
		fmt.Sprintf("DHNT_BASE=%v/go", base),
		fmt.Sprintf("GOPATH=%v/go", base),
		fmt.Sprintf("GOGS_WORK_DIR=%v/var/gogs", base),
		fmt.Sprintf("PATH=%v", AddPath(os.Getenv("PATH"), []string{
			fmt.Sprintf("%v/go/bin", base),
			fmt.Sprintf("%v/bin", base),
		})),
	}
	env := os.Environ()
	for _, e := range el {
		env = append(env, e)
	}
	cmd.Env = env

	//
	// cmdOut, err := cmd.StdoutPipe()
	// cmdErr, _ := cmd.StderrPipe()
	// read stdout and stderr
	// stdOutput, _ := ioutil.ReadAll(cmdOut)
	// errOutput, _ := ioutil.ReadAll(cmdErr)
	// fmt.Printf("STDOUT: %s\n", stdOutput)
	// fmt.Printf("ERROUT: %s\n", errOutput)

	cmdOut, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error creating stdout", err)
		return err
	}

	scanner := bufio.NewScanner(cmdOut)
	go func() {
		for scanner.Scan() {
			fmt.Printf("> %s\n", scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error starting cmd", err)
		return err
	}

	return cmd.Wait()
}
