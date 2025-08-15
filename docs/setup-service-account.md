# Service Account Setup for deck
This guide explains how to set up service accounts for deck, primarily for CI/CD automation purposes.

## Prerequisites
Service accounts are mainly used for automated workflows where interactive OAuth2 authentication is not possible.

### Required API Permissions
The service account needs the following OAuth2 scopes:
- `https://www.googleapis.com/auth/presentations`
- `https://www.googleapis.com/auth/drive`

### Shared Drive Requirement
Newly created service accounts don't have their own Google Drive storage quota. You must:

1. Create a **Shared Drive** (not a shared folder)
   - Shared folders use the owner's quota, which service accounts don't have
   - [Learn more about Shared Drives](https://support.google.com/a/answer/7212025)
2. Grant the service account **Content Manager** permission on the Shared Drive
3. Use the `--folder-id` flag with the Shared Drive ID:

```bash
deck apply slides.md --folder-id YOUR_SHARED_DRIVE_ID
```

## Authentication Methods

### Method 1: Service Account Key (Simple but less secure)
1. Create a service account in [Google Cloud Console](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Download the JSON key file
3. Set the environment variable:

```bash
export DECK_SERVICE_ACCOUNT_KEY='{"type":"service_account",...}'
deck apply slides.md
```

### Method 2: Workload Identity Federation (Recommended)

More secure as it doesn't require storing long-lived credentials.

#### GitHub Actions Setup
1. Configure Workload Identity Federation following [google-github-actions/auth documentation](https://github.com/google-github-actions/auth)
2. **Important**: You still need a service account for Google Drive permissions
   - Direct Workload Identity Federation without service account impersonation won't work
3. Use in your workflow:

```yaml
- uses: google-github-actions/auth@v2
  with:
    workload_identity_provider: 'projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<POOL_NAME>/providers/<PROVIDER_NAME>'
    service_account: '<SERVICE_ACCOUNT_NAME>@<PROJECT_ID>.iam.gserviceaccount.com'
- run: deck apply slides.md --folder-id <SHARED_DRIVE_ID>
  env:
    DECK_ENABLE_ADC: '1'
```

### Method 3: Access Token (For long-running tasks)
GitHub OIDC tokens expire in 5 minutes. For longer tasks, use an access token:

```yaml
- uses: google-github-actions/auth@v2
  id: auth
  with:
    token_format: 'access_token'
    workload_identity_provider: 'projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/<POOL_NAME>/providers/<PROVIDER_NAME>'
    service_account: '<SERVICE_ACCOUNT_NAME>@<PROJECT_ID>.iam.gserviceaccount.com'
- run: deck apply slides.md --folder-id <SHARED_DRIVE_ID>
  env:
    DECK_ACCESS_TOKEN: ${{ steps.auth.outputs.access_token }}
```

This method exchanges the OIDC token for a Google access token that typically lasts 1 hour.

## References
- [Google Cloud Service Accounts](https://cloud.google.com/iam/docs/service-accounts)
- [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation)
- [GitHub Actions with Google Cloud](https://github.com/google-github-actions/auth)
