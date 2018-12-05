package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
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

var (
	verbose bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "print verbose messages")
}

func logDebug(msg string, args ...interface{}) {
	if verbose {
		log.Printf("DEBUG "+msg, args...)
	}
}

func resolveProfile() *string {
	// Check AWS_PROFILE, AWS_DEFAULT_PROFILE env vars, use "default" as default
	profile := "default"

	if p, exists := os.LookupEnv("AWS_PROFILE"); exists {
		return &p
	}

	if p, exists := os.LookupEnv("AWS_DEFAULT_PROFILE"); exists {
		return &p
	}

	logDebug("PROFILE: %s", profile)
	return &profile
}

func resolveConfFile() *string {
	// Check AWS_CONFIG_FILE, use SDK default file as default
	conf_file := defaults.SharedConfigFilename()

	if f, exists := os.LookupEnv("AWS_CONFIG_FILE"); exists {
		conf_file = f
	}

	logDebug("CONFIG FILE: %s", conf_file)
	return &conf_file
}

func resolveCredFile() *string {
	// Check AWS_SHARED_CREDENTIALS_FILE, use SDK default file as default
	cred_file := defaults.SharedCredentialsFilename()

	if f, exists := os.LookupEnv("AWS_SHARED_CREDENTIALS_FILE"); exists {
		cred_file = f
	}

	logDebug("CREDENTIALS FILE: %s", cred_file)
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

	logDebug("DURATION: %s", duration.String())
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

func openLockFile(file string) (*os.File, error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fetchAccessKeys(profile *string) (*iam.AccessKey, error) {
	input := iam.ListAccessKeysInput{}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState:       session.SharedConfigEnable,
		Profile:                 *profile,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}))

	svc := iam.New(sess)
	truncated := true
	for truncated {
		res, err := svc.ListAccessKeys(&input)
		if err != nil {
			return nil, err
		}

		for _, k := range (*res).AccessKeyMetadata {
			key := k.AccessKeyId
			if strings.EqualFold(*k.Status, "Inactive") {
				logDebug("Deleting key %s\n", *key)
				svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: key})

			} else {
				logDebug("Inactivating key %s\n", *key)
				status := "Inactive"
				svc.UpdateAccessKey(&iam.UpdateAccessKeyInput{AccessKeyId: key, Status: &status})
			}
		}

		truncated = *res.IsTruncated
		if truncated {
			input.Marker = res.Marker
		}
	}

	res, err := svc.CreateAccessKey(&iam.CreateAccessKeyInput{})
	if err != nil {
		return nil, err
	}

	return (*res).AccessKey, nil
}

func rotateAccessKeys(profile *string) error {
	// While not the best possible solution, this should allow us to ensure only one process is attempting
	// to update the expiration file at a time.  If the "lock file" (the scratch file for writing the
	// updated expriation time) exists, then just do nothing and return.  The 'defer' function should help
	// ensure the file update happens regardless of how we leave this function after successfully opening
	// the lock file
	cf := resolveConfFile()
	expFile := filepath.Join(filepath.Dir(*cf), fmt.Sprintf("%s_%s", CRED_EXP_FILE_PREFIX, *profile))

	f, err := openLockFile(fmt.Sprintf("%s.lock", expFile))
	if err != nil {
		return nil
	}

	defer func(f *os.File, dest string) {
		f.Close()
		os.Rename(f.Name(), dest)
	}(f, expFile)

	keys, err := fetchAccessKeys(profile)
	if err != nil {
		return err
	}
	logDebug("KEYS: %+v", *keys)

	credFile := resolveCredFile()
	cfg, err := ini.Load(*credFile)
	if err != nil {
		return err
	}

	s, err := cfg.NewSection(*profile)
	_, err = s.NewKey("aws_access_key_id", *keys.AccessKeyId)
	_, err = s.NewKey("aws_secret_access_key", *keys.SecretAccessKey)
	cfg.SaveTo(*credFile)

	newExp := (*keys).CreateDate.Add(*getCredDuration(profile)).Unix()
	_, err = f.WriteString(strconv.FormatInt(newExp, 10))
	if err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()
	profile := resolveProfile()

	if getCredExpiration(profile).Before(time.Now()) {
		log.Printf("!!! IT'S TIME TO ROTATE THE AWS KEYS FOR PROFILE: %s !!!", *profile)
		err := rotateAccessKeys(profile)
		if err != nil {
			log.Fatalf("FATAL %+v\n", err)
		}
	}
}
