package secrethub

import (
	"errors"
	"testing"

	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub/testutil"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func TestACLCheckCommand_Run(t *testing.T) {
	testutil.Unit(t)

	testError := errio.Error(errors.New("test error"))

	cases := map[string]struct {
		cmd                  ACLCheckCommand
		newClientErr         error
		getter               fakeclient.AccessRuleGetter
		getterArgPath        api.Path
		getterArgAccountName api.AccountName
		lister               fakeclient.AccessLevelLister
		listerArgPath        api.Path
		out                  string
		err                  error
	}{
		"client creation error": {
			newClientErr: testError,
			err:          testError,
		},
		"success specific account": {
			cmd: ACLCheckCommand{
				accountName: "dev1",
				path:        "namespace/repo",
			},
			getter: fakeclient.AccessRuleGetter{
				ReturnsAccessRule: &api.AccessRule{
					Permission: api.PermissionRead,
				},
			},
			getterArgPath:        "namespace/repo",
			getterArgAccountName: "dev1",
			out:                  "read\n",
		},
		"success all accounts": {
			cmd: ACLCheckCommand{
				path: "namespace/repo",
			},
			lister: fakeclient.AccessLevelLister{
				ReturnsAccessLevels: []*api.AccessLevel{
					{
						Account: &api.Account{
							Name: "dev1",
						},
						Permission: api.PermissionRead,
					},
					{
						Account: &api.Account{
							Name: "dev2",
						},
						Permission: api.PermissionWrite,
					},
				},
			},
			listerArgPath: "namespace/repo",
			out: "PERMISSIONS    ACCOUNT\n" +
				"write          dev2\n" +
				"read           dev1\n",
		},
		"get error": {
			cmd: ACLCheckCommand{
				accountName: "dev1",
				path:        "namespace/repo",
			},
			getter: fakeclient.AccessRuleGetter{
				Err: testError,
			},
			getterArgPath:        "namespace/repo",
			getterArgAccountName: "dev1",
			err:                  testError,
		},
		"list error": {
			lister: fakeclient.AccessLevelLister{
				Err: testError,
			},
			err: testError,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := ui.NewFakeIO()
			tc.cmd.io = io

			getter := &tc.getter
			lister := &tc.lister

			tc.cmd.newClient = func() (secrethub.Client, error) {
				return fakeclient.Client{
					AccessRuleService: &fakeclient.AccessRuleService{
						Getter:      getter,
						LevelLister: lister,
					},
				}, tc.newClientErr
			}

			// Act
			err := tc.cmd.Run()

			// Assert
			testutil.Compare(t, err, tc.err)
			testutil.Compare(t, io.StdOut.String(), tc.out)
			testutil.Compare(t, getter.ArgPath, tc.getterArgPath)
			testutil.Compare(t, getter.ArgAccountName, tc.getterArgAccountName)
			testutil.Compare(t, lister.ArgPath, tc.listerArgPath)
		})
	}
}
