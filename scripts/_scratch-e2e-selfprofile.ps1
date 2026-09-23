#requires -Version 5.1
<#
  Ad-hoc end-to-end check for the self-service profile endpoint.
  Not part of the repo's test suite - scratch verification only.
#>
$ErrorActionPreference = "Stop"
$base = "http://localhost:8099"
$suffix = [DateTime]::UtcNow.Ticks.ToString().Substring(10)

# The API enforces a CSRF header on every mutating request.
$csrf = @{ "X-Requested-With" = "XMLHttpRequest" }

function Show($label, $value) { Write-Host "  $label : $value" -ForegroundColor Cyan }

Write-Host "== 1. Create employee via one-step registration ==" -ForegroundColor Yellow
$empEmail = "e2e$suffix@faceclock.local"
$reg = @{
    full_name = "E2E Employee"
    email     = $empEmail
    password  = "E2ePassword123"
} | ConvertTo-Json

# Login as super admin to call the admin registration endpoint.
$adminLogin = @{ email = "admin@faceclock.local"; password = "Faceclock123!" } | ConvertTo-Json
$admin = Invoke-RestMethod -Uri "$base/api/v1/auth/login" -Method Post -Body $adminLogin -ContentType "application/json" -Headers $csrf
$adminToken = $admin.data.access_token
Show "admin token" ($adminToken.Substring(0, 12) + "...")

$adminHeaders = @{ Authorization = "Bearer $adminToken"; "X-Requested-With" = "XMLHttpRequest" }

try {
    $created = Invoke-RestMethod -Uri "$base/api/v1/users/register-employee" -Method Post -Body $reg -ContentType "application/json" -Headers $adminHeaders
    Show "created employee" $created.data.full_name
    Show "profile_completed" $created.data.profile_completed
    Show "employee_number (should be empty)" "[$($created.data.employee_number)]"
}
catch {
    Write-Host "  register failed: $($_.ErrorDetails.Message)" -ForegroundColor Red
    exit 1
}

Write-Host "== 2. Login as that employee ==" -ForegroundColor Yellow
$empLogin = @{ email = $empEmail; password = "E2ePassword123" } | ConvertTo-Json
$emp = Invoke-RestMethod -Uri "$base/api/v1/auth/login" -Method Post -Body $empLogin -ContentType "application/json" -Headers $csrf
$token = $emp.data.access_token
$headers = @{ Authorization = "Bearer $token"; "X-Requested-With" = "XMLHttpRequest" }
Show "has employee.update_self" (($emp.data.user.permissions -contains "employee.update_self"))
Show "has employee.update (must be false)" (($emp.data.user.permissions -contains "employee.update"))

Write-Host "== 3. Employee completes own profile ==" -ForegroundColor Yellow
$payload = @{
    employee_number = "REAL-$suffix"
    position        = "Staf Keuangan"
    join_date       = "2026-02-01"
    phone           = "081234567890"
} | ConvertTo-Json
$done = Invoke-RestMethod -Uri "$base/api/v1/employees/me/profile" -Method Patch -Body $payload -ContentType "application/json" -Headers $headers
Show "employee_number" $done.data.employee_number
Show "position" $done.data.position
Show "join_date" $done.data.join_date
Show "profile_completed" $done.data.profile_completed

Write-Host "== 4. Admin-only field must be REJECTED ==" -ForegroundColor Yellow
$bad = @{ employment_status = "inactive" } | ConvertTo-Json
try {
    Invoke-RestMethod -Uri "$base/api/v1/employees/me/profile" -Method Patch -Body $bad -ContentType "application/json" -Headers $headers | Out-Null
    Write-Host "  FAIL: admin-only field was accepted!" -ForegroundColor Red
}
catch {
    $code = $_.Exception.Response.StatusCode.value__
    Show "status" $code
    if ($code -eq 400) { Write-Host "  PASS: rejected with 400" -ForegroundColor Green }
    else { Write-Host "  UNEXPECTED: $($_.ErrorDetails.Message)" -ForegroundColor Red }
}

Write-Host "== 5. Bad date must be 422 with field name ==" -ForegroundColor Yellow
$badDate = @{ join_date = "01-02-2026" } | ConvertTo-Json
try {
    Invoke-RestMethod -Uri "$base/api/v1/employees/me/profile" -Method Patch -Body $badDate -ContentType "application/json" -Headers $headers | Out-Null
    Write-Host "  FAIL: bad date was accepted!" -ForegroundColor Red
}
catch {
    $code = $_.Exception.Response.StatusCode.value__
    Show "status" $code
    Show "body" $_.ErrorDetails.Message
}

Write-Host "== 6. Employee cannot use the admin PATCH route ==" -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$base/api/v1/employees/me" -Method Patch -Body $payload -ContentType "application/json" -Headers $headers | Out-Null
    Write-Host "  FAIL: admin route reachable (or PATCH /me exists)" -ForegroundColor Red
}
catch {
    $code = $_.Exception.Response.StatusCode.value__
    Show "status" $code
    if ($code -eq 403 -or $code -eq 404 -or $code -eq 405) { Write-Host "  PASS: not permitted" -ForegroundColor Green }
    else { Write-Host "  UNEXPECTED: $($_.ErrorDetails.Message)" -ForegroundColor Red }
}
