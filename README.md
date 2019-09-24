# aws-key-rotator
[![Go Report Card](https://goreportcard.com/badge/github.com/mmmorris1975/aws-key-rotator)](https://goreportcard.com/report/github.com/mmmorris1975/aws-key-rotator)

A fairly opinionated tool to rotate static AWS API keys in the .aws/credentials file, or credentials provided via command
line arguments, or through standard input.  The goal of this utility is to provide a way to automate the rotation of the
credentials after a configurable interval. (Where a facility like the bash shell PROMPT_COMMAND facility will automatically
call this program to manage the credentials)

The program expects the AWS credentials to be configured in the user's `$HOME/.aws/credentials` file, by
default. It also reads some of the configuration from the user's `$HOME/.aws/config` file.

See the following for more information on AWS SDK configuration files:

- http://docs.aws.amazon.com/cli/latest/userguide/cli-config-files.html
- https://boto3.readthedocs.io/en/latest/guide/quickstart.html#configuration
- https://boto3.readthedocs.io/en/latest/guide/configuration.html#aws-config-file

## Installation

Download the release of the tool appropriate for your platform (Windows, Linux, Mac) from the [release page](https://github.com/mmmorris1975/aws-key-rotator/releases),
and install on your local system (preferably somewhere in your PATH); optionally renaming the file to something like `aws-key-rotator`.
Since the downloaded file is a statically-compiled binary, there are no other dependencies required to run the tool.

## Usage

```text
Usage of ./aws-key-rotator:
  -delete
    	delete credentials instead of inactivating
  -force
    	force credential rotation
  -verbose
    	print verbose messages
  -version
    	display program version
```

### Typical Usage
Add your static AWS credentials to the $HOME/.aws/credentials file, and see the [Configuration](#configuration) section
below to setup automatic AWS credential rotation.

Use the `-force` flag to require credential rotation regardless of the remaining credential lifetime.

### Atypical Usage
A set of AWS credentials can be passed to the program as either command line arguments, or via standard input.  This may
allow credential rotation for situations where the typical use and configuration may not be feasible.  Credentials can
be supplied to the program as long as they match the following regular expression logic: 20 or more alphanumeric characters
(which define the Access Key), followed by a single non-word character (typically whitespace, but could be a symbol
character like `:`), finally followed by at least 40 non-whitespace characters (which specifies the Secret Key).

If the credentials were successfully rotated the new Access Key and Secret Key will be returned on a single line to
standard output, separated by a single space character.

```text
$ echo AKIA32NPGC3S6ODNOKEY vWwa4Zmsoi7XSGtOI3560LiOkK1rg6HkN0S3cRe+ | aws-key-rotator
AIKANEWKEY mYneW5eCR3+KeY
```

## Configuration

### Configuring .aws/config
A configuration key called `aws_api_key_duration` can be added under each [profile] section in the .aws/config file to specify
that profile's API key lifetime as a Go [time.Duration](https://golang.org/pkg/time/#ParseDuration) string.  If this key is not found
for a particular profile, it will attempt to look up that key under the [default] section in the config file.  Failing to find this
configuration key in the .aws/config file will cause the program to default to a 12 hour credential duration.

### Environment Variables
This is the main mechanism to set the profile whose credentials will be managed by this tool.

The environment variable `AWS_PROFILE` can be used to select the profile configuration and credentials which will be managed by this tool.
If the variable is not set, then the `AWS_DEFAULT_PROFILE` environment variable is checked.  If neither is set, the tool will fallback to
using the configuration and credentials in the [default] section of the files.

The location of the config file may be over-ridden by setting the `AWS_CONFIG_FILE` environment variable, if that file exists in a location
other than the SDK default of `$HOME/.aws/config`.

The location of the credentials file may be over-ridden by setting the `AWS_SHARED_CREDENTIALS_FILE` environment variable, if that file
exists in a location other than the SDK default of `$HOME/.aws/credentials`

### Example for Shell auto-rotation of the [default] credentials on a non-default interval
  1. Edit .aws/config to set the interval
```
[default]
region = us-east-1
aws_api_key_duration = 6h
```
  2. Set AWS keys in .aws/credentials
```
[default]
aws_access_key_id = AKIA......
aws_secret_access_key = .........
```

##### Bash specific steps:
  1. Configure Bash PROMPT_COMMAND by editing $HOME/.bashrc and adding this snippet at the end of the file
```
function do_prompt_command {
  # Add any commands want executed each time before the PS1 prompt is displayed here
  aws-key-rotator
}

PROMPT_COMMAND=do_prompt_command
```
  2. Re-source $HOME/.bashrc to enable the PROMPT_COMMAND logic (`source ~/.bashrc`)

##### ZSH specific steps:
  1. Configure ZSH PROMPT_COMMAND by editing $HOME/.zprofile and adding this snippet at the end of the file
```
function do_prompt_command {
  # Add any commands want executed each time before the PS1 prompt is displayed here
  aws-key-rotator
}

precmd() { eval do_prompt_command }
```
  2. Re-source $HOME/.zprofile to enable the PROMPT_COMMAND logic (`source ~/.zprofile`)

## Building

Developed and tested using the go 1.12 tool chain, aws-sdk-go v1.24.5, and ini v1.48.0  
*NOTE* This project uses [go modules](https://github.com/golang/go/wiki/Modules) for dependency management

### Build Steps

A Makefile is included with the source code, and executing the default target via the `make` command should install all dependent
libraries and make the executable for your platform (or platform of choice if the GOOS and GOARCH env vars are set)

## Contributing

The usual github model for forking the repo and creating a pull request is the preferred way to
contribute to this tool.  Bug fixes, enhancements, doc updates, translations are always welcomed.
