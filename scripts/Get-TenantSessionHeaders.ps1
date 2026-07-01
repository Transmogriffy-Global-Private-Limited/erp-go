param(
  [string] $BaseUrl = "http://localhost:8080",
  [string] $TenantID = "00000000-0000-0000-0000-000000000001",
  [ValidateSet("allowed", "noaccess")]
  [string] $UserKind = "allowed",
  [string] $Email = "",
  [string] $Password = ""
)

$ErrorActionPreference = "Stop"

if ($Email -eq "") {
  if ($UserKind -eq "allowed") {
    $Email = "inventory.allowed+$($TenantID.Substring(0, 8))@example.test"
  }
  else {
    $Email = "inventory.noaccess+$($TenantID.Substring(0, 8))@example.test"
  }
}

if ($Password -eq "") {
  if ($UserKind -eq "allowed") {
    $Password = "dev-tenant-password"
  }
  else {
    $Password = "dev-noaccess-password"
  }
}

$LoginBody = @{
  tenant_id = $TenantID
  email = $Email
  password = $Password
} | ConvertTo-Json

$Login = Invoke-RestMethod "$BaseUrl/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $LoginBody

if (-not $Login.session_token) {
  throw "Tenant login did not return session_token."
}

return @{
  "X-Tenant-ID" = $TenantID
  "X-ERP-Session" = $Login.session_token
}
