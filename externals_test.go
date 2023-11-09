package kanvas

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAWSParamURL(t *testing.T) {
	cases := []struct {
		name string
		p    AWSParam
		want string
	}{
		{
			name: "simple",
			p: AWSParam{
				Path: "foo",
			},
			want: "ref+awsssm://foo",
		},
		{
			name: "with subpath",
			p: AWSParam{
				Path:    "foo",
				SubPath: "bar",
			},
			want: "ref+awsssm://foo#/bar",
		},
		{
			name: "with region",
			p: AWSParam{
				Path:   "foo",
				Region: "us-east-1",
			},
			want: "ref+awsssm://foo?region=us-east-1",
		},
		{
			name: "with profile",
			p: AWSParam{
				Path:    "foo",
				Profile: "default",
			},
			want: "ref+awsssm://foo?profile=default",
		},
		{
			name: "with role arn",
			p: AWSParam{
				Path:    "foo",
				RoleARN: "arn:aws:iam::123456789012:role/MyRole",
			},
			want: "ref+awsssm://foo?role_arn=arn:aws:iam::123456789012:role/MyRole",
		},
		{
			name: "everything except subpath",
			p: AWSParam{
				Path:    "foo",
				Region:  "us-east-1",
				Profile: "default",
				RoleARN: "arn:aws:iam::123456789012:role/MyRole",
			},
			want: "ref+awsssm://foo?region=us-east-1&profile=default&role_arn=arn:aws:iam::123456789012:role/MyRole",
		},
		{
			name: "everything",
			p: AWSParam{
				Path:    "foo",
				SubPath: "bar",
				Region:  "us-east-1",
				Profile: "default",
				RoleARN: "arn:aws:iam::123456789012:role/MyRole",
			},
			want: "ref+awsssm://foo?region=us-east-1&profile=default&role_arn=arn:aws:iam::123456789012:role/MyRole#/bar",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.p.ValsRefURL()
			require.Equal(t, tc.want, got)
		})
	}
}
