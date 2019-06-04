package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/mmmorris1975/aws-config/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestGetCredDuration(t *testing.T) {
	t.Run("bad-file-env", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, "bad")
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = "p"
		if getCredDuration() != credDurationDefault {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("empty-profile", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = ""
		if getCredDuration() != 1*time.Hour {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("explicit-default", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = "default"
		if getCredDuration() != 1*time.Hour {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("no-default", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile()+"-nodefault")
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = ""
		if getCredDuration() != credDurationDefault {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("other-profile", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = "other"
		if getCredDuration() != 8*time.Hour {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("missing-prop", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = "no-prop"
		if getCredDuration() != credDurationDefault {
			t.Error("credential duration mismatch")
			return
		}
	})

	t.Run("bad-prop", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)

		profile = "bad-value"
		if getCredDuration() != credDurationDefault {
			t.Error("credential duration mismatch")
			return
		}
	})
}

func TestCredExpired(t *testing.T) {
	t.Run("file-missing", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, "bad")
		defer os.Unsetenv(config.ConfigFileEnvVar)
		profile = ""

		if !credExpired() {
			t.Error("expected expired creds")
			return
		}
	})

	t.Run("invalid-value", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)
		profile = config.DefaultProfileName

		f := expFile()
		if err := ioutil.WriteFile(f, []byte("abc123"), 0600); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f)

		if !credExpired() {
			t.Error("expected expired creds")
			return
		}
	})

	t.Run("expired", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)
		profile = config.DefaultProfileName

		exp := time.Now().Add(-24 * time.Hour).Unix()
		f := expFile()
		if err := ioutil.WriteFile(f, []byte(strconv.FormatInt(exp, 10)), 0600); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f)

		if !credExpired() {
			t.Error("expected expired creds")
			return
		}
	})

	t.Run("valid", func(t *testing.T) {
		os.Setenv(config.ConfigFileEnvVar, testConfFile())
		defer os.Unsetenv(config.ConfigFileEnvVar)
		profile = config.DefaultProfileName

		exp := time.Now().Add(-30 * time.Minute).Unix()
		f := expFile()
		if err := ioutil.WriteFile(f, []byte(strconv.FormatInt(exp, 10)), 0600); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f)

		if credExpired() {
			t.Error("expected unexpired creds")
			return
		}
	})
}

func TestExpFile(t *testing.T) {
	confDir := filepath.Dir(defaults.SharedConfigFilename())

	t.Run("no-conf", func(t *testing.T) {
		profile = ""

		if expFile() != filepath.Join(confDir, credTimeFilePrefix+"_") {
			t.Error("expiration file does not match default")
		}
	})

	t.Run("path", func(t *testing.T) {
		profile = "test1"
		r := filepath.Join(confDir, fmt.Sprintf("%s_%s", credTimeFilePrefix, profile))

		if expFile() != r {
			t.Errorf("expiration file name mismatch (WANT: %s, GOT: %s)", expFile(), r)
		}
	})
}

func TestNewAtomicFileFile(t *testing.T) {
	t.Run("good", func(t *testing.T) {
		lf, err := NewAtomicFile(filepath.Join(os.TempDir(), "good"))
		if err != nil {
			t.Error(err)
			return
		}
		defer lf.Close()
		defer os.Remove(lf.Name())
	})

	t.Run("multi-open", func(t *testing.T) {
		lf, err := NewAtomicFile(filepath.Join(os.TempDir(), "multi-open"))
		if err != nil {
			t.Error(err)
			return
		}
		defer lf.Close()
		defer os.Remove(lf.Name())

		if _, err := NewAtomicFile(lf.Name()); err == nil {
			t.Error("did not receive expected error when opening existing lock file")
			return
		}
	})
}

func testConfFile() string {
	return filepath.Join("test", "aws-config")
}
