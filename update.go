package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kr/binarydist"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	upcktimePath = "cktime"
	plat         = runtime.GOOS + "-" + runtime.GOARCH
)

const devValidTime = 7 * 24 * time.Hour

// Update protocol.
//
//   GET hk.heroku.com/hk-current-linux-amd64.json
//
//   200 ok
//   {
//       "Version": "2",
//       "Sha256": "..." // base64
//   }
//
// then
//
//   GET hkpatch.s3.amazonaws.com/hk-1-linux-amd64-to-2
//
//   200 ok
//   [bsdiff data]
//
// or
//
//   GET hkdist.s3.amazonaws.com/hk-2-linux-amd64.gz
//
//   200 ok
//   [gzipped executable data]
type Updater struct {
	hkURL   string
	binURL  string
	diffURL string
	dir     string
	info    struct {
		Version   string
		Sha256 []byte
	}
}

func (u *Updater) run() {
	os.MkdirAll(u.dir, 0777)
	if u.wantUpdate() {
		l := exec.Command("logger", "-thk")
		c := exec.Command("hk", "update")
		if w, err := l.StdinPipe(); err == nil && l.Start() == nil {
			c.Stdout = w
			c.Stderr = w
		}
		c.Start()
	}
}

func (u *Updater) wantUpdate() bool {
	path := u.dir + upcktimePath
	if Version == "dev" || readTime(path).After(time.Now()) {
		return false
	}
	wait := 24*time.Hour + randDuration(24*time.Hour)
	return writeTime(path, time.Now().Add(wait))
}

func (u *Updater) update() error {
	path, err := exec.LookPath("hk")
	if err != nil {
		return err
	}
	old, err := os.Open(path)
	if err != nil {
		return err
	}
	err = u.fetchInfo()
	if err != nil {
		return err
	}
	if u.info.Version == Version {
		return nil
	}
	bin, err := u.fetchAndApplyPatch(old)
	if err != nil {
		bin, err = u.fetchBin()
		if err != nil {
			return err
		}
	}
	h := sha256.New()
	h.Write(bin)
	if !bytes.Equal(h.Sum(nil), u.info.Sha256) {
		return errors.New("new file hash mismatch after patch")
	}
	return install(old.Name(), bin)
}

func (u *Updater) fetchInfo() error {
	r, err := fetch(u.hkURL + "hk-current-" + plat + ".json")
	if err != nil {
		return err
	}
	defer r.Close()
	err = json.NewDecoder(r).Decode(&u.info)
	if err != nil {
		return err
	}
	if len(u.info.Sha256) != sha256.Size {
		return errors.New("bad cmd hash in info")
	}
	return nil
}

func (u *Updater) fetchAndApplyPatch(old io.Reader) ([]byte, error) {
	r, err := fetch(u.diffURL + slug(Version) + "-to-" + u.info.Version)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	err = binarydist.Patch(old, &buf, r)
	return buf.Bytes(), err
}

func (u *Updater) fetchBin() ([]byte, error) {
	r, err := fetch(u.binURL + slug(u.info.Version) + ".gz")
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(buf, gz); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func install(name string, p []byte) error {
	part := filepath.Join(filepath.Dir(name), "hk.part")
	err := ioutil.WriteFile(part, p, 0755)
	if err != nil {
		return err
	}
	defer os.Remove(part)
	return os.Rename(part, name)
}

// returns a random duration in [0,n).
func randDuration(n time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(n)))
}

func slug(ver string) string {
	return "hk-" + ver + "-" + plat
}

func fetch(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad http status from %s: %v", url, resp.Status)
	}
	return resp.Body, nil
}

func readTime(path string) time.Time {
	p, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return time.Time{}
	}
	if err != nil {
		return time.Now().Add(1000 * time.Hour)
	}
	t, err := time.Parse(time.RFC3339, string(p))
	if err != nil {
		return time.Now().Add(1000 * time.Hour)
	}
	return t
}

func writeTime(path string, t time.Time) bool {
	return ioutil.WriteFile(path, []byte(t.Format(time.RFC3339)), 0644) == nil
}
