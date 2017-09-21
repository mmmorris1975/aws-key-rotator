package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	CONFIG_CRED_DURATION_KEY = "aws_api_key_duration"
	DEFAULT_CRED_DURATION    = time.Duration(12) * time.Hour
	CRED_EXP_FILE_PREFIX     = ".aws_credentials_expiration"
)

func resolveProfile() *string {
	// Check AWS_PROFILE, AWS_DEFAULT_PROFILE env vars, use "default" as default
	profile := "default"

	if p, exists := os.LookupEnv("AWS_PROFILE"); exists {
		return &p
	}

	if p, exists := os.LookupEnv("AWS_DEFAULT_PROFILE"); exists {
		return &p
	}

	return &profile
}

func resolveConfFile() *string {
	// Check AWS_CONFIG_FILE, use SDK default file as default
	conf_file := defaults.SharedConfigFilename()

	if f, exists := os.LookupEnv("AWS_CONFIG_FILE"); exists {
		conf_file = f
	}

	return &conf_file
}

func resolveCredFile() *string {
	// Check AWS_SHARED_CREDENTIALS_FILE, use SDK default file as default
	cred_file := defaults.SharedCredentialsFilename()

	if f, exists := os.LookupEnv("AWS_SHARED_CREDENTIALS_FILE"); exists {
		cred_file = f
	}

	return &cred_file
}

func getCredDuration(profile *string) *time.Duration {
	// lookup aws_api_key_duration in resolved conf file for profile (or "default" profile if not found)
	duration := DEFAULT_CRED_DURATION
	confFile := resolveConfFile()

	cfg, err := ini.Load(*confFile)
	cfg.BlockMode = false

	if err != nil {
		log.Printf("WARNING Error loading confg file %s (%v), using default credential duration\n", *confFile, err)
	} else {
		s, err := cfg.GetSection(*profile)
		if err != nil {
			// Section does not exist, try default
			s, err = cfg.GetSection("default")
			if err != nil {
				return &duration
			}
		}

		if s.HasKey(CONFIG_CRED_DURATION_KEY) {
			k, err := s.GetKey(CONFIG_CRED_DURATION_KEY)
			if err == nil {
				duration, _ = time.ParseDuration(k.Value())
			}
		}
	}

	return &duration
}

func getCredExpiration(profile *string) time.Time {
	cf := resolveConfFile()
	expFile := filepath.Join(filepath.Dir(*cf), fmt.Sprintf("%s_%s", CRED_EXP_FILE_PREFIX, *profile))

	data, err := ioutil.ReadFile(expFile)
	if err != nil {
		// TODO handle error (especially if file does not exist (possible first run))
	}

	num, err := strconv.ParseInt(strings.TrimSpace(string(data)), 0, 64)
	if err != nil {
		// TODO handle error
	}

	return time.Unix(num, 0)
}

func setCredExpiration(profile *string) {
	// While not the best possible solution, this should allow us to ensure only one process is attempting
	// to update the expiration file at a time.  If the "lock file" (the scratch file for writing the
	// updated expriation time) exists, then just do nothing and return.  The 'defer' function should help
	// ensure the file update happens regardless of how we leave this function after successfully opening
	// the lock file
	cf := resolveConfFile()
	expFile := filepath.Join(filepath.Dir(*cf), fmt.Sprintf("%s_%s", CRED_EXP_FILE_PREFIX, *profile))
	lockFile := fmt.Sprintf("%s.lock", expFile)

	f, err := os.OpenFile(lockFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		// just leave, someone else is probably updating the expiration time
		// what if they aren't, and it's a dead lock file?
		return
	}

	defer func(f *os.File, dest string) {
		f.Close()
		os.Rename(f.Name(), dest)
	}(f, expFile)

	newExp := time.Now().Add(*getCredDuration(profile)).Unix()
	newExpStr := strconv.FormatInt(newExp, 10)

	_, err = f.WriteString(newExpStr)
	if err != nil {
		log.Printf("ERROR writing new expiration %+v", err)
	}
}

func main() {
	profile := resolveProfile()
	credDuration := getCredDuration(profile)

	log.Printf("PROFILE: %s", *profile)
	log.Printf("CRED_DUR: %s", (*credDuration).String())
	log.Printf("EXP: %+v", getCredExpiration(profile))

	setCredExpiration(profile)
}
