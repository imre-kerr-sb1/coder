package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	agpl "github.com/coder/coder/cli"
	"github.com/coder/coder/cli/cliui"
	"github.com/coder/coder/codersdk"
	"github.com/coder/coder/enterprise/coderd/license"
)

var jwtRegexp = regexp.MustCompile(`^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`)

func licenses() *cobra.Command {
	cmd := &cobra.Command{
		Short:   "Add, delete, and list licenses",
		Use:     "licenses",
		Aliases: []string{"license"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		licenseAdd(),
		licensesList(),
		licenseDelete(),
	)
	return cmd
}

func licenseAdd() *cobra.Command {
	var (
		filename string
		license  string
		debug    bool
	)
	cmd := &cobra.Command{
		Use:   "add [-f file | -l license]",
		Short: "Add license to Coder deployment",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := agpl.CreateClient(cmd)
			if err != nil {
				return err
			}

			switch {
			case filename != "" && license != "":
				return xerrors.New("only one of (--file, --license) may be specified")

			case filename == "" && license == "":
				license, err = cliui.Prompt(cmd, cliui.PromptOptions{
					Text:     "Paste license:",
					Secret:   true,
					Validate: validJWT,
				})
				if err != nil {
					return err
				}

			case filename != "" && license == "":
				var r io.Reader
				if filename == "-" {
					r = cmd.InOrStdin()
				} else {
					f, err := os.Open(filename)
					if err != nil {
						return err
					}
					defer f.Close()
					r = f
				}
				lb, err := io.ReadAll(r)
				if err != nil {
					return err
				}
				license = string(lb)
			}
			license = strings.Trim(license, " \n")
			err = validJWT(license)
			if err != nil {
				return err
			}

			licResp, err := client.AddLicense(
				cmd.Context(),
				codersdk.AddLicenseRequest{License: license},
			)
			if err != nil {
				return err
			}
			if debug {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(licResp)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "License with ID %d added\n", licResp.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&filename, "file", "f", "", "Load license from file")
	cmd.Flags().StringVarP(&license, "license", "l", "", "License string")
	cmd.Flags().BoolVar(&debug, "debug", false, "Output license claims for debugging")
	return cmd
}

func validJWT(s string) error {
	if jwtRegexp.MatchString(s) {
		return nil
	}
	return xerrors.New("Invalid license")
}

func parseClaims(c map[string]interface{}) (*license.Claims, error) {
	byt, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var claims license.Claims
	err = json.Unmarshal(byt, &claims)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

func prettyPrintLicenses(w io.Writer, ls []codersdk.License) error {
	for _, l := range ls {
		fmt.Fprintf(w, "--- License #%v\n", l.ID)
		t := cliui.Table()
		t.AppendHeader(table.Row{"Feature", "Status"})
		claims, err := parseClaims(l.Claims)
		if err != nil {
			return err
		}
		if claims.AllFeatures {
			for _, name := range codersdk.FeatureNames {
				t.AppendRow(table.Row{name, "Available"})
			}
		} else {
			for _, name := claims.Features
		}
		// features, ok := l.Claims["features"].(map[string])
		_, _ = io.WriteString(w, t.Render())
		fmt.Fprintf(w, "%+v\n", claims)
	}
	return nil
}

func licensesList() *cobra.Command {
	var formatJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List licenses (including expired)",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := agpl.CreateClient(cmd)
			if err != nil {
				return err
			}

			licenses, err := client.Licenses(cmd.Context())
			if err != nil {
				return err
			}

			if formatJSON {
				// Ensure that we print "[]" instead of "null" when there are no licenses.
				if licenses == nil {
					licenses = make([]codersdk.License, 0)
				}

				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(licenses)
			}

			prettyPrintLicenses(cmd.OutOrStdout(), licenses)
			return nil
		},
	}
	cmd.Flags().BoolVar(&formatJSON, "json", false, "Show raw license JSON")
	return cmd
}

func licenseDelete() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <id>",
		Short:   "Delete license by ID",
		Aliases: []string{"del", "rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := agpl.CreateClient(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return xerrors.Errorf("license ID must be an integer: %s", args[0])
			}
			err = client.DeleteLicense(cmd.Context(), int32(id))
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "License with ID %d deleted\n", id)
			return nil
		},
	}
	return cmd
}
