package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	clientapi "github.com/afterdarksys/adsm/internal/api"
	"github.com/afterdarksys/adsm/internal/output"
	"github.com/spf13/cobra"
)

var storageNoWait bool
var storageRegion string
var storageYes bool
var credentialBuckets []string
var credentialActions []string
var credentialDuration int64

var storageCmd = &cobra.Command{Use: "storage", Short: "Manage hosted object storage", Long: "Manage Dark Storage resources through the After Dark customer control plane.", GroupID: groupDelivery}
var bucketsCmd = &cobra.Command{Use: "buckets", Short: "Manage storage buckets"}
var credentialsCmd = &cobra.Command{Use: "credentials", Aliases: []string{"sts"}, Short: "Manage temporary S3 credentials"}

func operationKey(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}
func renderOperation(cmd *cobra.Command, operation clientapi.Operation) error {
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, operation)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{Headers: []string{"OPERATION", "STATE", "RESOURCE"}, Rows: [][]string{{operation.ID, operation.State, operation.ResourceID}}})
}
func waitIfRequested(cmd *cobra.Command, operation clientapi.Operation) (clientapi.Operation, error) {
	if storageNoWait {
		return operation, nil
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Waiting for operation", operation.ID)
	return apiClient().WaitOperation(cmd.Context(), operation.ID, time.Second)
}

var bucketListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List buckets", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().Buckets(cmd.Context())
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"NAME", "STATE", "REVISION", "ID"}}
	for _, item := range items {
		table.Rows = append(table.Rows, []string{item.CustomerName, item.LifecycleState, strconv.FormatInt(item.Revision, 10), item.ID})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}
var bucketGetCmd = &cobra.Command{Use: "get NAME", Short: "Get a bucket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	item, err := apiClient().Bucket(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, item)
	}
	return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{Headers: []string{"NAME", "STATE", "REGION", "REVISION", "ID"}, Rows: [][]string{{item.CustomerName, item.LifecycleState, fmt.Sprint(item.ObservedState["region"]), strconv.FormatInt(item.Revision, 10), item.ID}}})
}}
var bucketCreateCmd = &cobra.Command{Use: "create NAME", Short: "Create a bucket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	operation, err := apiClient().CreateBucket(cmd.Context(), args[0], storageRegion, operationKey("bucket-create"))
	if err != nil {
		return err
	}
	operation, err = waitIfRequested(cmd, operation)
	if err != nil {
		return err
	}
	return renderOperation(cmd, operation)
}}
var bucketDeleteCmd = &cobra.Command{Use: "delete NAME", Short: "Delete a bucket", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	if !storageYes {
		return fmt.Errorf("deletion requires --yes")
	}
	bucket, err := apiClient().Bucket(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	operation, err := apiClient().DeleteBucket(cmd.Context(), args[0], operationKey("bucket-delete"), bucket.Revision)
	if err != nil {
		return err
	}
	operation, err = waitIfRequested(cmd, operation)
	if err != nil {
		return err
	}
	return renderOperation(cmd, operation)
}}

var credentialIssueCmd = &cobra.Command{Use: "issue", Short: "Issue temporary S3 credentials", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	if len(credentialBuckets) == 0 || len(credentialActions) == 0 {
		return fmt.Errorf("--bucket and --action are required")
	}
	credential, err := apiClient().IssueStorageCredential(cmd.Context(), credentialBuckets, credentialActions, credentialDuration, operationKey("sts-issue"))
	if err != nil {
		return err
	}
	if outFormat == output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, &output.Table{Headers: []string{"ACCESS KEY", "SECRET KEY", "SESSION TOKEN", "EXPIRATION"}, Rows: [][]string{{credential.AccessKey, credential.SecretKey, credential.SessionToken, credential.Expiration.Format(time.RFC3339)}}})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, credential)
}}
var credentialListCmd = &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List temporary credentials without secret material", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
	items, err := apiClient().StorageCredentials(cmd.Context())
	if err != nil {
		return err
	}
	if outFormat != output.FormatTable {
		return output.Render(cmd.OutOrStdout(), outFormat, items)
	}
	table := &output.Table{Headers: []string{"ACCESS KEY", "PRINCIPAL", "EXPIRATION"}}
	for _, item := range items {
		table.Rows = append(table.Rows, []string{item.AccessKey, item.PrincipalID, item.Expiration.Format(time.RFC3339)})
	}
	return output.Render(cmd.OutOrStdout(), outFormat, table)
}}
var credentialRevokeCmd = &cobra.Command{Use: "revoke ACCESS_KEY", Short: "Revoke temporary credentials", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	if !storageYes {
		return fmt.Errorf("revocation requires --yes")
	}
	if err := apiClient().RevokeStorageCredential(cmd.Context(), args[0], operationKey("sts-revoke")); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Revoked", args[0])
	return err
}}

func init() {
	bucketCreateCmd.Flags().StringVar(&storageRegion, "region", "", "storage region")
	bucketCreateCmd.Flags().BoolVar(&storageNoWait, "no-wait", false, "return after the operation is accepted")
	bucketDeleteCmd.Flags().BoolVar(&storageNoWait, "no-wait", false, "return after the operation is accepted")
	bucketDeleteCmd.Flags().BoolVar(&storageYes, "yes", false, "confirm destructive deletion")
	credentialIssueCmd.Flags().StringSliceVar(&credentialBuckets, "bucket", nil, "bucket scope (repeatable)")
	credentialIssueCmd.Flags().StringSliceVar(&credentialActions, "action", []string{"s3:GetObject", "s3:PutObject", "s3:ListBucket"}, "S3 action (repeatable)")
	credentialIssueCmd.Flags().Int64Var(&credentialDuration, "duration", 3600, "credential lifetime in seconds")
	credentialRevokeCmd.Flags().BoolVar(&storageYes, "yes", false, "confirm credential revocation")
	bucketsCmd.AddCommand(bucketListCmd, bucketGetCmd, bucketCreateCmd, bucketDeleteCmd)
	credentialsCmd.AddCommand(credentialIssueCmd, credentialListCmd, credentialRevokeCmd)
	storageCmd.AddCommand(bucketsCmd, credentialsCmd)
	rootCmd.AddCommand(storageCmd)
}
