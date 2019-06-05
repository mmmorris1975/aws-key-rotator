package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mmmorris1975/aws-config/config"
	"github.com/mmmorris1975/simple-logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	credDurationConfigKey = "aws_api_key_duration"
	credDurationDefault   = time.Duration(12) * time.Hour
	credTimeFilePrefix    = ".aws_credentials_expiration"
)

var (
	// Version of the program
	Version  string
	profile  string
	verbose  bool
	delCreds bool
	version  bool
	log      *simple_logger.Logger
	lockFile *AtomicFile
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "print verbose messages")
	flag.BoolVar(&delCreds, "delete", false, "delete credentials instead of inactivating")
	flag.BoolVar(&version, "version", false, "display program version")

	log = simple_logger.StdLogger
	log.SetLevel(simple_logger.INFO)
	log.SetFlags(0)
}

func main() {
	flag.Parse()
	if version {
		fmt.Printf("VERSION: %s\n\n", Version)
	}

	if verbose {
		log.SetLevel(simple_logger.DEBUG)
	}

	profile = config.ResolveProfile(nil)

	var err error
	lockFile, err = NewAtomicFile(fmt.Sprintf("%s.lock", expFile()))
	if err != nil {
		if os.IsExist(err) {
			log.Debug("lock file exists, another instance of this program may be updating keys")
		} else {
			log.Fatalf("unable to open lock file, can not update keys: %v", err)
		}
	}

	defer func() {
		if err := os.Remove(lockFile.Name()); err != nil {
			if !os.IsNotExist(err) {
				log.Debugf("error removing lock file: %v", err)
			}
		}
	}()

	if credExpired() {
		log.Printf("!!! IT'S TIME TO ROTATE THE AWS KEYS FOR PROFILE: %s !!!", profile)
		err := rotateAccessKeys()
		if err != nil {
			log.Fatal(err)
		}

		if err := os.Rename(lockFile.Name(), expFile()); err != nil {
			log.Warnf("error renaming lock file: %v", err)
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
	duration := credDurationDefault

	cfg, err := config.NewAwsConfigFile(nil)
	if err != nil {
		log.Warnf("error loading confg file: %v, using default credential duration (%s)", err, credDurationDefault.String())
		return duration
	}

	p, err := cfg.Profile(profile)
	if err != nil {
		log.Warnf("profile not found, returning default credential duration (%s)", credDurationDefault.String())
		return duration
	}
	log.Debugf("PROFILE: %v", p.KeysHash())

	if k, err := p.GetKey(credDurationConfigKey); err == nil {
		duration, err = time.ParseDuration(k.Value())
		if err != nil {
			log.Warnf("invalid duration, returning default credential duration (%s)", credDurationDefault.String())
			return credDurationDefault
		}
	} else {
		log.Warnf("error getting credential duration property: %v", err)
	}

	log.Debugf("DURATION: %s", duration.String())
	return duration
}

func rotateAccessKeys() error {
	defer func() {
		if err := lockFile.Close(); err != nil {
			log.Debugf("error closing lock file: %v", err)
		}
	}()

	keys, err := fetchAccessKeys()
	if err != nil {
		return err
	}
	log.Debugf("KEYS: %+v", *keys)

	creds, err := config.NewAwsCredentialsFile(nil)
	if err != nil {
		return err
	}

	if err := creds.UpdateCredentials(profile, keys); err != nil {
		return err
	}

	if err := creds.SaveTo(creds.Path); err != nil {
		return err
	}

	if _, err = lockFile.WriteString(fmt.Sprintf("%d", (*keys).CreateDate.Unix())); err != nil {
		return err
	}

	return nil
}

func fetchAccessKeys() (*iam.AccessKey, error) {
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
				if _, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: key}); err != nil {
					log.Warnf("error deleting access key: %v", err)
				}
			} else {
				if delCreds {
					log.Debugf("Deleting key %s\n", *key)
					if _, err := svc.DeleteAccessKey(&iam.DeleteAccessKeyInput{AccessKeyId: key}); err != nil {
						log.Warnf("error deleting access key: %v", err)
					}
				} else {
					log.Debugf("Inactivating key %s\n", *key)
					status := iam.StatusTypeInactive
					if _, err := svc.UpdateAccessKey(&iam.UpdateAccessKeyInput{AccessKeyId: key, Status: &status}); err != nil {
						log.Warnf("error inactivating access key: %v", err)
					}
				}
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
	return filepath.Join(confDir, fmt.Sprintf("%s_%s", credTimeFilePrefix, profile))
}
