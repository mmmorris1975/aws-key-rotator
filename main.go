package aws_key_rotator

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mmmorris1975/aws-config"
	"github.com/mmmorris1975/aws-config/config"
	"github.com/mmmorris1975/aws-config/credentials"
	"github.com/mmmorris1975/simple-logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	CredDurationConfigKey = "aws_api_key_duration"
	CredDurationDefault   = time.Duration(12) * time.Hour
	CredTimeFilePrefix    = ".aws_credentials_expiration"
)

var (
	profile  string
	verbose  bool
	delCreds bool
	log      *simple_logger.Logger
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "print verbose messages")
	flag.BoolVar(&delCreds, "delete", false, "delete credentials instead of inactivating")

	log = simple_logger.StdLogger
	log.SetLevel(simple_logger.INFO)
	log.SetFlags(0)
}

func main() {
	flag.Parse()
	if verbose {
		log.SetLevel(simple_logger.DEBUG)
	}

	profile = aws_config.ResolveProfile(nil)

	if credExpired() {
		log.Printf("!!! IT'S TIME TO ROTATE THE AWS KEYS FOR PROFILE: %s !!!", profile)
		err := rotateAccessKeys()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func credExpired() bool {
	data, err := ioutil.ReadFile(expFile())
	if err != nil {
		// file doesn't exist, or is unreadable, return expired
		log.Debug("could not read expiration file, returning expired")
		return true
	}

	num, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		log.Warnf("unable to parse value '%s' as an integer, returning expired", data)
		return true
	}

	return time.Since(time.Unix(int64(num), 0)) > getCredDuration()
}

func getCredDuration() time.Duration {
	duration := CredDurationDefault

	cfg, err := config.Load(nil)
	if err != nil {
		log.Warnf("error loading confg file: %v, using default credential duration (%s)", err, CredDurationDefault.String())
		return duration
	}

	p, err := cfg.Profile(profile)
	if err != nil {
		log.Warnf("profile not found, returning default credential duration (%s)", CredDurationDefault.String())
		return duration
	}

	if k, err := p.GetKey(CredDurationConfigKey); err == nil {
		if duration, err = time.ParseDuration(k.Value()); err != nil {
			log.Warnf("invalid duration, returning default credential duration (%s)", CredDurationDefault.String())
			return CredDurationDefault
		}
	}

	log.Debugf("DURATION: %s", duration.String())
	return duration
}

func rotateAccessKeys() error {
	// While not the best possible solution, this should allow us to ensure only one process is attempting to update
	// the expiration file at a time.  If the "lock file" (the scratch file for writing the updated expiration time)
	// exists, then just do nothing and return.  The 'defer' function should help ensure the file update happens
	// regardless of how we leave this function after successfully opening the lock file
	f, err := openLockFile(fmt.Sprintf("%s.lock", expFile()))
	if err != nil {
		log.Warn("unable to open lock file, can not update keys")
		return nil
	}

	defer func(f *os.File, dest string) {
		f.Close()
		os.Rename(f.Name(), dest)
	}(f, expFile())

	keys, err := fetchAccessKeys(profile)
	if err != nil {
		return err
	}
	log.Debugf("KEYS: %+v", *keys)

	creds, err := credentials.Load(nil)
	if err != nil {
		return err
	}

	if err := creds.UpdateCredentials(profile, keys); err != nil {
		return err
	}

	if err := creds.SaveTo(creds.Path); err != nil {
		return err
	}

	if _, err = f.WriteString(fmt.Sprintf("%d", (*keys).CreateDate.Unix())); err != nil {
		return err
	}

	return nil
}

func openLockFile(file string) (*os.File, error) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fetchAccessKeys(profile string) (*iam.AccessKey, error) {
	input := iam.ListAccessKeysInput{}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState:       session.SharedConfigEnable,
		Profile:                 profile,
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
				log.Debugf("Deleting key %s\n", *key)
				svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: key})
			} else {
				log.Debugf("Inactivating key %s\n", *key)
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

func expFile() string {
	confDir := filepath.Dir(defaults.SharedConfigFilename())
	return filepath.Join(confDir, fmt.Sprintf("%s_%s", CredTimeFilePrefix, profile))
}
