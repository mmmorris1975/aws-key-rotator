# aws-key-rotator
[![Go Report Card](https://goreportcard.com/badge/github.com/mmmorris1975/aws-key-rotator)](https://goreportcard.com/report/github.com/mmmorris1975/aws-key-rotator)

A fairly opinionated tool to rotate static AWS API keys in the .aws/credentials file.  The goal of this
utility is to provide a way to automate the rotation of the credentials after a configurable interval.
(Where a facility like the bash shell PROMPT_COMMAND facility will automatically call this program to
manage the credentials)

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
Usage of aws-key-rotator:
  -delete
    	delete credentials instead of inactivating
  -verbose
    	print verbose messages
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

### Example for Bash Shell auto-rotation of the [default] credentials on a non-default interval
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
  3. Configure Bash PROMPT_COMMAND by editing $HOME/.bashrc and adding this snippet at the end of the file
```
function do_prompt_command {
  # Add any commands want executed each time before the PS1 prompt is displayed here
  aws-key-rotator
}

PROMPT_COMMAND=do_prompt_command
```
  4. Re-source $HOME/.bashrc to enable the PROMPT_COMMAND logic (`source ~/.bashrc`)

## Building

Developed and tested using the go 1.11 tool chain, aws-sdk-go v1.16.23, and ini v1.39.2  
*NOTE* This project uses the (currently) experimental `dep` dependency manager.  See https://github.com/golang/dep for details.

Assuming you have a go workspace, and GOPATH environment variable set (https://golang.org/doc/code.html#Organization):
  1. Run `go get -d github.com/mmmorris1975/aws-key-rotator`
  2. Run `dep ensure` to check/retrieve dependencies
  3. Then run `go build github.com/mmmorris1975/aws-key-rotator` to create the executable `aws-key-rotator` in the `$GOPATH/bin` directory

## Contributing

The usual github model for forking the repo and creating a pull request is the preferred way to
contribute to this tool.  Bug fixes, enhancements, doc updates, translations are always welcomed.
