package kanvas

import (
	"fmt"
	"strings"
)

// Externals exposes external parameters and secrets as the component's outputs.
// Internally, externals invoke the `vals` library to fetch the values from external sources
// like AWS parameter store, AWS secret manager, etc.
type Externals struct {
	// Outputs defines the mapping from output names to the vals refs
	Outputs map[string]OutputFrom `yaml:"outputs,omitempty"`
}

func (e *Externals) NewValsTemplate() (map[string]interface{}, error) {
	m := map[string]interface{}{}
	for k, o := range e.Outputs {
		var v string

		if o.AWSParam != nil {
			v = o.AWSParam.ValsRefURL()
		} else if o.AWSSecret != nil {
			v = o.AWSSecret.ValsRefURL()
		} else if o.GoogleSheetCell != nil {
			v = o.GoogleSheetCell.ValsRefURL()
		} else {
			return nil, fmt.Errorf("invalid external output %q: it must have either AWSParam, AWSSecret, or GoogleSheetCell", k)
		}

		m[k] = v
	}

	return m, nil
}

// Var is a variable to be passed to terraform
type OutputFrom struct {
	// AWSParam is a reference to an AWS parameter store parameter
	AWSParam *AWSParam `yaml:"awsParam"`
	// AWSSecret is a reference to an AWS secret manager secret
	AWSSecret *AWSSecret `yaml:"awsSecret"`
	// GoogleSheetCell is a reference to a Google Sheet cell
	GoogleSheetCell *GoogleSheetCell `yaml:"googleSheetCell"`
	// TerraformState is a reference to a terraform state
	TerraformState *TerraformState `yaml:"terraformState"`
}

// AWSParam is a reference to an AWS parameter store parameter
type AWSParam struct {
	// Path is the path of the parameter
	Path string `yaml:"name"`
	// SubPath is the subpath of the parameter
	SubPath string `yaml:"subPath"`
	// Region is the AWS region to be used to access the parameter
	Region string `yaml:"region"`
	// RoleARN is the ARN of the role to be assumed to access the parameter
	RoleARN string `yaml:"roleARN"`
	// Profile is the AWS profile to be used to access the parameter
	Profile string `yaml:"profile"`
	// Mode is optional and maps to the `mode` parameter of the `vals` library.
	// See https://github.com/helmfile/vals#aws-ssm-parameter-store for more information.
	Mode string `yaml:"mode"`
	// Recursive is optional and maps to the `recursive` parameter of the `vals` library.
	// This is effective only when Mode=singleparam.
	// See https://github.com/helmfile/vals#aws-ssm-parameter-store for more information.
	Recursive bool `yaml:"recursive"`
}

func (p AWSParam) ValsRefURL() string {
	base := fmt.Sprintf("ref+awsssm://%s?region=%s&role_arn=%s&profile=%s", p.Path, p.Region, p.RoleARN, p.Profile)

	if p.Mode != "" {
		base = fmt.Sprintf("%s&mode=%s", base, p.Mode)
	}

	if p.SubPath != "" {
		base = fmt.Sprintf("%s#/%s", base, p.SubPath)
	}

	return base
}

func (p AWSParam) Validate() error {
	if p.Path == "" {
		return fmt.Errorf("path must be set")
	}

	if p.Recursive && p.Mode != "singleparam" {
		return fmt.Errorf("recursive is effective only when mode=singleparam")
	}

	return nil
}

// AWSSecret is a reference to an AWS secret manager secret
// See https://github.com/helmfile/vals#aws-secrets-manager
// for more details.
type AWSSecret struct {
	// ID is the ID of the secret
	// Examples:
	// - myteam/mydoc
	ID string `yaml:"arn"`
	// ARN is the ARN of the secret
	// Examples:
	// - arn:aws:secretsmanager:<REGION>:<ACCOUNT_ID>:secret:/myteam/mydoc
	ARN string `yaml:"arn"`
	// Path is the key path to the value within the json-decoded secret.
	// If empty, the whole content of the secret is returned as-is.
	// If non-empty, the content of the secret denoted by ID or ARN is decoded as json,
	// and the value at the path is returned.
	// We don't currently call this "json path" because it is not well-known JSONPath.
	Path string `yaml:"path"`
	// VersionID is the version ID of the secret
	VersionID string `yaml:"versionID"`
	// Region is the AWS region to be used to access the secret
	Region string `yaml:"region"`
	// RoleARN is the ARN of the role to be assumed to access the secret
	RoleARN string `yaml:"roleARN"`
	// Profile is the AWS profile to be used to access the secret
	Profile string `yaml:"profile"`
}

func (s AWSSecret) ValsRefURL() string {
	base := "ref+awssecret://"

	if s.ARN != "" {
		base = fmt.Sprintf("%s%s", base, s.ARN)
	} else {
		base = fmt.Sprintf("%s%s", base, s.ID)
	}

	if s.RoleARN != "" {
		base = fmt.Sprintf("%s&role_arn=%s", base, s.RoleARN)
	}

	if s.VersionID != "" {
		base = fmt.Sprintf("%s&version_id=%s", base, s.VersionID)
	}

	if s.Profile != "" {
		base = fmt.Sprintf("%s&profile=%s", base, s.Profile)
	}

	if s.Region != "" {
		base = fmt.Sprintf("%s&region=%s", base, s.Region)
	}

	if s.Path != "" {
		base = fmt.Sprintf("%s#/%s", base, s.Path)
	}

	return base
}

func (s AWSSecret) Validate() error {
	if s.ARN == "" && s.ID == "" {
		return fmt.Errorf("either arn or id must be set")
	}

	return nil
}

// GoogleSheetCell is a reference to a Google Sheet cell
type GoogleSheetCell struct {
	// SheetID is the SheetID of the Google Sheet
	SheetID string `yaml:"sheetID"`
	// CredentialsFile is the path to the credentials.json file
	CredentialsFile string `yaml:"credentialsFile"`
	// Key is the key of the value to be fetched from the Google Sheet
	// We assume the first column is the key, and the second column is the value
	Key string `yaml:"key"`
}

func (c GoogleSheetCell) Validate() error {
	if c.SheetID == "" {
		return fmt.Errorf("sheetID must be set")
	}

	if c.CredentialsFile == "" {
		return fmt.Errorf("credentialsFile must be set")
	}

	if c.Key == "" {
		return fmt.Errorf("key must be set")
	}

	return nil
}

func (c GoogleSheetCell) ValsRefURL() string {
	return fmt.Sprintf("ref+googlesheets://%s?credentials_file=%s/%s", c.SheetID, c.CredentialsFile, c.Key)
}

type TerraformState struct {
	// Path is the path to the terraform state file
	// This is relative to the base dir, which is where kanvas.yaml is located.
	// Example: terraform/terraform.tfstate
	Path string `yaml:"path"`
	// URL is the URL to the terraform state file
	// Either s3:// or gs:// is supported.
	// Example: s3://mybucket/terraform.tfstate
	URL string `yaml:"url"`
	// Expr is the expression to be evaluated against the terraform state to fetch the value
	// Only dot-notation is supported.
	// Example: output.OUTPUT_NAME
	Expr string `yaml:"expr"`
}

func (s TerraformState) Validate() error {
	if s.Path == "" && s.URL == "" {
		return fmt.Errorf("either path or url must be set")
	}

	if s.Path != "" && s.URL != "" {
		return fmt.Errorf("path and url are mutually exclusive")
	}

	if s.Path != "" && !strings.HasSuffix(s.Path, ".tfstate") {
		return fmt.Errorf("path must end with .tfstate")
	}

	if s.URL != "" && !strings.HasPrefix(s.URL, "s3://") && !strings.HasPrefix(s.URL, "gs://") {
		return fmt.Errorf("url must start with s3:// or gs://")
	}

	if s.Expr == "" {
		return fmt.Errorf("expr must be set")
	}

	return nil
}

func (s TerraformState) ValsRefURL() string {
	base := "ref+tfstate"
	if s.URL != "" {
		base = fmt.Sprintf("%s%s", base, s.URL)
	} else {
		base = fmt.Sprintf("%s://%s", base, s.Path)
	}
	return fmt.Sprintf("%s/%s", base, s.Expr)
}
