$ErrorActionPreference = 'Stop'

$ResultDir = $PSScriptRoot
$Root = Resolve-Path (Join-Path $ResultDir '..\..')
$Deploy = Join-Path $Root 'deploy'
$ResultPath = Join-Path $ResultDir 'g3_http_result.json'
$SqlRowsPath = Join-Path $ResultDir 'g3_sql_rows.json'

function Read-DotEnv {
    param([string]$Path)
    $map = @{}
    Get-Content -LiteralPath $Path | ForEach-Object {
        if ($_ -match '^([^#=]+)=(.*)$') {
            $map[$matches[1].Trim()] = $matches[2]
        }
    }
    return $map
}

function Write-JsonFile {
    param($Value, [string]$Path)
    $Value | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $Path -Encoding UTF8
}

function Invoke-JsonHttp {
    param(
        [string]$Method,
        [string]$Url,
        $Body = $null,
        [hashtable]$Headers = @{},
        [int]$TimeoutSec = 20
    )
    $jsonBody = $null
    if ($null -ne $Body) {
        $jsonBody = $Body | ConvertTo-Json -Depth 20 -Compress
    }
    try {
        $params = @{
            Method = $Method
            Uri = $Url
            Headers = $Headers
            TimeoutSec = $TimeoutSec
            UseBasicParsing = $true
        }
        if ($null -ne $jsonBody) {
            $params['Body'] = $jsonBody
            $params['ContentType'] = 'application/json'
        }
        $resp = Invoke-WebRequest @params
        $content = [string]$resp.Content
        $parsed = $null
        if ($content.Trim() -ne '') {
            try { $parsed = $content | ConvertFrom-Json -ErrorAction Stop } catch {}
        }
        return @{ ok = $true; status = [int]$resp.StatusCode; json = $parsed; content_length = $content.Length }
    } catch {
        $status = 0
        $content = ''
        if ($_.Exception.Response) {
            try { $status = [int]$_.Exception.Response.StatusCode } catch {}
            try {
                $stream = $_.Exception.Response.GetResponseStream()
                $reader = New-Object System.IO.StreamReader($stream)
                $content = $reader.ReadToEnd()
            } catch {}
        }
        $parsed = $null
        if ($content.Trim() -ne '') {
            try { $parsed = $content | ConvertFrom-Json -ErrorAction Stop } catch {}
        }
        return @{ ok = $false; status = $status; json = $parsed; content_length = $content.Length; error = $_.Exception.Message }
    }
}

function Get-JsonValue {
    param($Object, [string[]]$Path)
    $cur = $Object
    foreach ($part in $Path) {
        if ($null -eq $cur) { return $null }
        $prop = $cur.PSObject.Properties[$part]
        if ($null -eq $prop) { return $null }
        $cur = $prop.Value
    }
    return $cur
}

function Get-ResponseData {
    param($Json)
    $data = Get-JsonValue $Json @('data')
    if ($null -ne $data) { return $data }
    return $Json
}

function Get-Token {
    param($Json)
    $token = Get-JsonValue $Json @('data','access_token')
    if ($token) { return [string]$token }
    $token = Get-JsonValue $Json @('access_token')
    if ($token) { return [string]$token }
    return ''
}

function Get-KeyFromResponse {
    param($Json)
    $key = Get-JsonValue $Json @('data','key')
    if ($key) { return [string]$key }
    $key = Get-JsonValue $Json @('key')
    if ($key) { return [string]$key }
    return ''
}

function Get-IdFromResponse {
    param($Json)
    $id = Get-JsonValue $Json @('data','id')
    if ($null -ne $id) { return [int64]$id }
    $id = Get-JsonValue $Json @('id')
    if ($null -ne $id) { return [int64]$id }
    return 0
}

function Fail-G3 {
    param([hashtable]$Result, [string]$Phase, [string]$Message)
    $Result.status = 'error'
    $Result.phase = $Phase
    $Result.error = $Message
    Write-JsonFile $Result $ResultPath
    $public = @{
        status = $Result.status
        phase = $Result.phase
        error = $Result.error
        health = $Result.health
        rows_recent_15m = $Result.rows_recent_15m
        suspicious_rows = $Result.suspicious_rows
        is_live = $Result.is_live
        cleanup_required = $true
    }
    $public | ConvertTo-Json -Compress
    exit 1
}

$result = @{
    status = 'running'
    phase = 'start'
    health = $false
    login = @{ token_present = $false; attempts = 0; last_status = 0; message = $null }
    group = @{ created = $false; id_present = $false }
    account = @{ created = $false; id_present = $false }
    api_key = @{ created = $false; key_present = $false; id_present = $false }
    balance = @{ topped_up = $false; status = 0; user_id_present = $false }
    chat = @{ status = 0; ok = $false; content_length = 0 }
    video = @{ create_status = 0; task_id_present = $false; final_status = $null; result_url_present = $false; ResultURL_present = $false }
    rows_recent_15m = $null
    suspicious_rows = $null
    is_live = $null
    samples_live = $null
    sample_preview_safe = $null
}

$envPath = Join-Path $Deploy '.env'
if (!(Test-Path -LiteralPath $envPath)) {
    Fail-G3 $result 'env' 'deploy/.env is missing'
}
$envMap = Read-DotEnv $envPath
$adminEmail = 'admin@sub2api.local'
if ($envMap.ContainsKey('ADMIN_EMAIL') -and $envMap['ADMIN_EMAIL']) {
    $adminEmail = $envMap['ADMIN_EMAIL']
}
$adminPassword = $envMap['ADMIN_PASSWORD']
if (!$adminPassword) {
    Fail-G3 $result 'env' 'ADMIN_PASSWORD is empty'
}

$base = 'http://127.0.0.1:8080'
$healthOk = $false
for ($i = 1; $i -le 60; $i++) {
    try {
        $health = Invoke-WebRequest -Uri "$base/health" -TimeoutSec 3 -UseBasicParsing
        if ([int]$health.StatusCode -eq 200) {
            $healthOk = $true
            break
        }
    } catch {}
    Start-Sleep -Seconds 2
}
$result.health = $healthOk
if (!$healthOk) {
    Fail-G3 $result 'wait_health' 'health endpoint did not become ready'
}

$adminJwt = ''
$adminUserId = 0
for ($i = 1; $i -le 36; $i++) {
    $loginResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/auth/login" -Body @{ email = $adminEmail; password = $adminPassword }
    $adminJwt = Get-Token $loginResp.json
    $maybeUserId = Get-JsonValue $loginResp.json @('data','user','id')
    if ($maybeUserId) { $adminUserId = [int64]$maybeUserId }
    $message = Get-JsonValue $loginResp.json @('message')
    $result.login = @{ token_present = [bool]$adminJwt; attempts = $i; last_status = $loginResp.status; message = $message }
    if ($adminJwt) { break }
    Start-Sleep -Seconds 3
}
if (!$adminJwt) {
    Fail-G3 $result 'login' 'admin login did not return token'
}

$adminHeaders = @{ Authorization = "Bearer $adminJwt" }
$stamp = (Get-Date).ToString('yyyyMMddHHmmss')

if ($adminUserId -le 0) {
    $meResp = Invoke-JsonHttp -Method GET -Url "$base/api/v1/auth/me" -Headers $adminHeaders
    $maybeUserId = Get-JsonValue $meResp.json @('data','id')
    if ($maybeUserId) { $adminUserId = [int64]$maybeUserId }
}
if ($adminUserId -le 0) {
    Fail-G3 $result 'admin_user' 'admin user id not found'
}

$balanceResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/admin/users/$adminUserId/balance" -Headers $adminHeaders -Body @{
    balance = 10
    operation = "add"
    notes = "G3 dev capture local validation"
}
$result.balance = @{ topped_up = ($balanceResp.status -eq 200); status = $balanceResp.status; user_id_present = $true }
if ($balanceResp.status -ne 200) {
    Fail-G3 $result 'balance' 'admin dev balance top-up failed'
}

$groupResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/admin/groups" -Headers $adminHeaders -Body @{
    name = "g3-capture-$stamp"
    description = "G3 dev capture verification"
    platform = "anthropic"
    rate_multiplier = 1
    is_exclusive = $false
    subscription_type = "standard"
}
$groupId = Get-IdFromResponse $groupResp.json
$result.group = @{ created = ($groupResp.status -eq 200 -or $groupResp.status -eq 201); id_present = [bool]$groupId; status = $groupResp.status }
if (!$groupId) {
    Fail-G3 $result 'group' 'group creation did not return id'
}

$accountResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/admin/accounts" -Headers $adminHeaders -Body @{
    name = "g3-anthropic-mock-$stamp"
    platform = "anthropic"
    type = "apikey"
    credentials = @{
        api_key = "fake-local-anthropic-key-for-g3"
        base_url = "http://anthropic-mock:8081"
    }
    extra = @{
        anthropic_passthrough = $true
    }
    concurrency = 1
    priority = 100
    group_ids = @($groupId)
    confirm_mixed_channel_risk = $true
}
$accountId = Get-IdFromResponse $accountResp.json
$result.account = @{ created = ($accountResp.status -eq 200 -or $accountResp.status -eq 201); id_present = [bool]$accountId; status = $accountResp.status }
if (!$accountId) {
    Fail-G3 $result 'account' 'mock account creation did not return id'
}

$apiKeyResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/keys" -Headers $adminHeaders -Body @{
    name = "g3-dev-capture-$stamp"
    group_id = $groupId
}
$apiKey = Get-KeyFromResponse $apiKeyResp.json
$apiKeyId = Get-IdFromResponse $apiKeyResp.json
if (!$apiKey) {
    $apiKeyResp = Invoke-JsonHttp -Method POST -Url "$base/api/v1/keys" -Headers $adminHeaders -Body @{
        name = "g3-dev-capture-$stamp"
    }
    $apiKey = Get-KeyFromResponse $apiKeyResp.json
    $apiKeyId = Get-IdFromResponse $apiKeyResp.json
    if ($apiKeyId) {
        $bindResp = Invoke-JsonHttp -Method PUT -Url "$base/api/v1/admin/api-keys/$apiKeyId" -Headers $adminHeaders -Body @{ group_id = $groupId }
        $result.api_key_bind = @{ status = $bindResp.status; ok = ($bindResp.status -eq 200) }
    }
}
$result.api_key = @{ created = ($apiKeyResp.status -eq 200 -or $apiKeyResp.status -eq 201); key_present = [bool]$apiKey; id_present = [bool]$apiKeyId; status = $apiKeyResp.status }
if (!$apiKey) {
    Fail-G3 $result 'api_key' 'api key creation did not return key'
}

$apiHeaders = @{ Authorization = "Bearer $apiKey" }
$fakeToken = 'sk-test-placeholder-' + '12345678901234567890'
$fakePhone = '13800138000'
$chatResp = Invoke-JsonHttp -Method POST -Url "$base/v1/chat/completions" -Headers $apiHeaders -Body @{
    model = "claude-3-5-haiku-20241022"
    stream = $true
    messages = @(
        @{ role = "user"; content = "G3 dev capture chat check phone $fakePhone token $fakeToken" }
    )
}
$result.chat = @{ status = $chatResp.status; ok = ($chatResp.status -ge 200 -and $chatResp.status -lt 300); content_length = $chatResp.content_length }
if (!$result.chat.ok) {
    Fail-G3 $result 'chat' 'chat completions request failed'
}

$videoCreate = Invoke-JsonHttp -Method POST -Url "$base/v1/video/tasks" -Headers $apiHeaders -Body @{
    provider = "mock"
    task_type = "reference_to_video"
    model = "mock-video-v1"
    prompt = "D2 dev capture check, phone $fakePhone, token $fakeToken"
    reference_image_url = "https://example.invalid/ref.png"
    aspect_ratio = "16:9"
    duration = 5
    resolution = "720p"
}
$videoData = Get-ResponseData $videoCreate.json
$taskId = Get-JsonValue $videoData @('id')
if (!$taskId) { $taskId = Get-JsonValue $videoData @('task_id') }
$result.video.create_status = $videoCreate.status
$result.video.task_id_present = [bool]$taskId
if (!$taskId) {
    Fail-G3 $result 'video_create' 'mock video task creation did not return id'
}

for ($i = 1; $i -le 30; $i++) {
    $videoGet = Invoke-JsonHttp -Method GET -Url "$base/v1/video/tasks/$taskId" -Headers $apiHeaders
    $vData = Get-ResponseData $videoGet.json
    $vStatus = Get-JsonValue $vData @('status')
    $lower = if ($vStatus) { ([string]$vStatus).ToLowerInvariant() } else { '' }
    $resultUrl = Get-JsonValue $vData @('result_url')
    $resultURLUpper = Get-JsonValue $vData @('ResultURL')
    $result.video.final_status = $lower
    $result.video.result_url_present = [bool]$resultUrl
    $result.video.ResultURL_present = [bool]$resultURLUpper
    if ($lower -eq 'succeeded') { break }
    Start-Sleep -Seconds 2
}
if ($result.video.final_status -ne 'succeeded') {
    Fail-G3 $result 'video_poll' 'mock video task did not reach succeeded'
}
if (!$result.video.result_url_present -or !$result.video.ResultURL_present) {
    Fail-G3 $result 'video_result_url' 'mock video response missing result_url or ResultURL'
}

Start-Sleep -Seconds 4

$sqlRows = @"
SELECT COALESCE(json_agg(row_to_json(t)), '[]'::json)
FROM (
  SELECT id, task_id, model,
         left(prompt_redacted, 120) AS prompt_preview,
         left(response_redacted, 120) AS response_preview,
         prompt_bytes, response_bytes, response_truncated
  FROM ai_generation_content
  WHERE created_at >= NOW() - INTERVAL '15 minutes'
  ORDER BY created_at DESC
) t;
"@
$rowsJson = docker compose -p sub2api_g3 -f docker-compose.dev.yml exec -T postgres psql -U sub2api -d sub2api -t -A -c $sqlRows
$rowsParsed = @()
try { $rowsParsed = $rowsJson | ConvertFrom-Json -ErrorAction Stop } catch { $rowsParsed = @() }
Write-JsonFile $rowsParsed $SqlRowsPath
$result.rows_recent_15m = @($rowsParsed).Count

$sqlSuspicious = @"
SELECT COUNT(*) AS suspicious_rows
FROM ai_generation_content
WHERE created_at >= NOW() - INTERVAL '15 minutes'
  AND (
    prompt_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR response_redacted ~ 'sk-[A-Za-z0-9_-]{20,}'
    OR prompt_redacted ~ '13800138000'
    OR response_redacted ~ '13800138000'
  );
"@
$suspiciousText = docker compose -p sub2api_g3 -f docker-compose.dev.yml exec -T postgres psql -U sub2api -d sub2api -t -A -c $sqlSuspicious
$result.suspicious_rows = [int]([string]$suspiciousText).Trim()

$statsResp = Invoke-JsonHttp -Method GET -Url "$base/api/v1/admin/generation-content/stats" -Headers $adminHeaders
$statsData = Get-ResponseData $statsResp.json
$samplesResp = Invoke-JsonHttp -Method GET -Url "$base/api/v1/admin/generation-content/samples" -Headers $adminHeaders
$samplesData = Get-ResponseData $samplesResp.json
$result.is_live = [bool](Get-JsonValue $statsData @('is_live'))
$result.samples_live = [bool](Get-JsonValue $samplesData @('is_live'))
$samples = Get-JsonValue $samplesData @('samples')
$safe = $true
if ($samples) {
    foreach ($s in @($samples)) {
        $preview = ([string](Get-JsonValue $s @('prompt_preview'))) + ' ' + ([string](Get-JsonValue $s @('response_preview')))
        if ($preview -match 'sk-[A-Za-z0-9_-]{20,}' -or $preview -match '13800138000') {
            $safe = $false
        }
    }
}
$result.sample_preview_safe = $safe

if ($result.rows_recent_15m -lt 2) {
    Fail-G3 $result 'sql_rows' 'ai_generation_content recent rows less than 2'
}
if ($result.suspicious_rows -ne 0) {
    Fail-G3 $result 'sql_redaction' 'suspicious_rows is not zero'
}
if (!$result.is_live -or !$result.samples_live) {
    Fail-G3 $result 'admin_dashboard' 'generation-content admin dashboard is not live'
}
if (!$result.sample_preview_safe) {
    Fail-G3 $result 'admin_samples' 'admin sample preview still contains suspicious raw data'
}

$result.status = 'ok'
$result.phase = 'complete'
Write-JsonFile $result $ResultPath
@{
    status = $result.status
    phase = $result.phase
    health = $result.health
    rows_recent_15m = $result.rows_recent_15m
    suspicious_rows = $result.suspicious_rows
    is_live = $result.is_live
    samples_live = $result.samples_live
    sample_preview_safe = $result.sample_preview_safe
    video_result_url_present = $result.video.result_url_present
    video_ResultURL_present = $result.video.ResultURL_present
} | ConvertTo-Json -Compress
