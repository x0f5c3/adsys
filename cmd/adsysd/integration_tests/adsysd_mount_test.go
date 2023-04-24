package adsys_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/adsys/cmd/adsysd/daemon"
)

func TestAdsysdMount(t *testing.T) {
	fixtureDir := filepath.Join("testdata", t.Name())

	tests := map[string]struct {
		mountsFile    string
		sessionAnswer string
		noKrbTicket   bool

		addArgs []string

		wantErr bool
	}{
		// Single entries
		"Mount successfully nfs share":                     {mountsFile: "mounts_with_nfs_entry"},
		"Mount successfully smb share":                     {mountsFile: "mounts_with_smb_entry"},
		"Mount successfully ftp share":                     {mountsFile: "mounts_with_ftp_entry"},
		"Mount successfully entry without kerberos ticket": {mountsFile: "mounts_with_nfs_entry", noKrbTicket: true},

		// Kerberos authentication entries
		"Mount successfully krb auth entry": {mountsFile: "mounts_with_krb_auth_entry"},

		// Many entries
		"Mount successfully many entries with same protocol":       {mountsFile: "mounts_with_many_nfs_entries"},
		"Mount successfully many entries with different protocols": {mountsFile: "mounts_with_many_entries"},
		"Mount successfully many kerberos auth entries":            {mountsFile: "mounts_with_many_krb_auth_entries"},

		// File cases
		"Exit code 0 when file is empty": {mountsFile: "mounts_with_no_entries"},

		// File errors
		"Error when file has badly formated entries": {mountsFile: "mounts_with_bad_entries", wantErr: true},
		"Error when file doesn't exist":              {mountsFile: "do_not_exist", wantErr: true},

		// Authentication errors
		"Error when auth is needed but no kerberos ticket is available": {mountsFile: "mounts_with_krb_auth_entry", noKrbTicket: true, wantErr: true},
		"Error when anonymous auth is not supported by the server":      {mountsFile: "mounts_with_nfs_entry", sessionAnswer: "gvfs_anonymous_error", noKrbTicket: true, wantErr: true},

		// Bus errors
		"Error when VFS bus is not available": {mountsFile: "mounts_with_nfs_entry", sessionAnswer: "gvfs_no_vfs_bus", wantErr: true},
		"Error during ListMountableInfo step": {mountsFile: "mounts_with_nfs_entry", sessionAnswer: "gvfs_list_info_fail", wantErr: true},
		"Error during MountLocation step":     {mountsFile: "mounts_with_nfs_entry", sessionAnswer: "gvfs_mount_loc_fail", wantErr: true},

		// Generic errors
		"Error when trying to mount unsupported protocol": {mountsFile: "mounts_with_unsupported_protocol", wantErr: true},
		"Error during mount process":                      {mountsFile: "mounts_with_error", wantErr: true},

		// Binary usage cases
		"Correctly prints the help message": {addArgs: []string{"--help"}},

		// Binary usage errors
		"Errors out and prints usage message when executed with less than 2 arguments": {wantErr: true},
		"Errors out and prints usage message when executed with more than 2 arguments": {addArgs: []string{"more", "than", "two"}, wantErr: true},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.sessionAnswer == "" {
				tc.sessionAnswer = "polkit_yes"
			}
			dbusAnswer(t, tc.sessionAnswer)

			d := daemon.New()

			args := []string{"mount"}
			if tc.mountsFile != "" {
				args = append(args, filepath.Join(fixtureDir, tc.mountsFile))
			}
			if tc.addArgs != nil {
				args = append(args, tc.addArgs...)
			}
			changeAppArgs(t, d, "", args...)

			if !tc.noKrbTicket {
				t.Setenv("KRB5CCNAME", "kerberos_ticket")
			}

			err := d.Run()
			if tc.wantErr {
				require.Error(t, err, "Client should exit with an error")
				return
			}
			require.NoError(t, err, "client should exit with no error")
		})
	}
}
