package LogConsolidateLib

import (
	"os"
	"os/exec"
	"testing"

	"github.com/pschlump/Go-FTL/server/sizlib"
)

func Test_HostnameFileSubtitution(t *testing.T) {

	// this tests the substitution for hostnames and the sizlib.Qt function

	cfg := ReadConfig("../testdata/%{hostname%}.test2.json", "test2")

	if cfg.Default.Key != "test2-a" {
		t.Errorf("Error: Failed to resolve file name corectly")
	}

}

func Test_SizlibExistsWorksWithNamedPipes(t *testing.T) {

	// this tests the substitution for hostnames and the sizlib.Qt function
	var mkfifo = "/usr/bin/mkfifo"

	pipe := "../testdata/pipe4"
	os.Remove(pipe)

	if sizlib.Exists(pipe) {
		t.Errorf("Error: Failed to remove pipe")
	}

	out, err := exec.Command(mkfifo, pipe).Output()
	if err != nil {
		t.Errorf("Error: Error from runing mkfifo, %s\n", err)
	}
	if string(out) != "" {
		t.Errorf("Error: Error from runing mkfifo, output=%s\n", out)
	}

	if !sizlib.Exists(pipe) {
		t.Errorf("Error: Failed to create pipe")
	}

	os.Remove(pipe)
}
