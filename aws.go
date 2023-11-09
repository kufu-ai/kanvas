package kanvas

// AWS is an AWS-specific configuration
// This is currently used to ensure that you have the right AWS credentials
// that are required to access resources such as ECR and EKS.
type AWS struct {
	// Account is the AWS account to use / associated with
	// the AWS credentials you are using
	Account string `json:"account,omitempty"`
}
