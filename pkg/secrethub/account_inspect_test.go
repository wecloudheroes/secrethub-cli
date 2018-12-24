package secrethub

import (
	"testing"
	"time"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/secrethub/fakes"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestAccountInspect(t *testing.T) {
	testutil.Component(t)

	testErr := errio.Namespace("test").Code("test").Error("test error")

	date := time.Date(2018, time.July, 30, 10, 49, 18, 0, time.UTC)

	cases := map[string]struct {
		cmd AccountInspectCommand
		err error
		out string
	}{
		"success": {
			cmd: AccountInspectCommand{
				newClient: func() (secrethub.Client, error) {
					return &fakeclient.Client{
						UserService: &fakeclient.UserService{
							MeGetter: fakeclient.MeGetter{
								ReturnsUser: &api.User{
									Username:      "dev1",
									FullName:      "Developer Uno",
									Email:         "dev1@keylocker.eu",
									EmailVerified: true,
									CreatedAt:     &date,
									PublicKey:     []byte("abcde"),
								},
								Err: nil,
							},
						},
					}, nil
				},
				timeFormatter: &fakes.TimeFormatter{
					Response: "2018-07-30T10:49:18Z",
				},
			},
			err: nil,
			out: `{
    "Username": "dev1",
    "FullName": "Developer Uno",
    "Email": "dev1@keylocker.eu",
    "EmailVerified": true,
    "CreatedAt": "2018-07-30T10:49:18Z",
    "PublicAccountKey": "YWJjZGU="
}
`,
		},
		"client error": {
			cmd: AccountInspectCommand{
				newClient: func() (secrethub.Client, error) {
					return fakeclient.Client{
						UserService: &fakeclient.UserService{
							MeGetter: fakeclient.MeGetter{
								ReturnsUser: nil,
								Err:         api.ErrSignatureNotVerified,
							},
						},
					}, nil
				},
			},
			err: api.ErrSignatureNotVerified,
			out: "",
		},
		"client creation error": {
			cmd: AccountInspectCommand{
				newClient: func() (secrethub.Client, error) {
					return nil, testErr
				},
			},
			err: testErr,
			out: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			io := ui.NewFakeIO()
			tc.cmd.io = io

			// Act
			err := tc.cmd.Run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
		})
	}
}
