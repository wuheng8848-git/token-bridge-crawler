# Test Crawler Script
# 用于本地测试爬虫功能

param(
    [string]$Vendor = "google",
    [string]$DatabaseURL = $env:CRAWLER_DATABASE_URL,
    [string]$TBBaseURL = "http://localhost:8080",
    [string]$TBToken = $env:TB_ADMIN_API_TOKEN
)

if (-not $DatabaseURL) {
    Write-Host "Error: DatabaseURL not set. Use env CRAWLER_DATABASE_URL or -DatabaseURL parameter" -ForegroundColor Red
    exit 1
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Token Bridge Crawler - Test Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. 检查数据库连接
Write-Host "Step 1: Checking database connection..." -ForegroundColor Yellow
try {
    $result = psql $DatabaseURL -c "SELECT 1;" 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Database connection: OK" -ForegroundColor Green
    } else {
        Write-Host "  Database connection: FAILED" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "  Database connection: FAILED - $_" -ForegroundColor Red
    exit 1
}

# 2. 检查表是否存在
Write-Host ""
Write-Host "Step 2: Checking database tables..." -ForegroundColor Yellow
$tables = @("vendor_price_snapshots", "vendor_price_details")
foreach ($table in $tables) {
    $exists = psql $DatabaseURL -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '$table');" -t 2>$null
    if ($exists -match "t") {
        Write-Host "  Table $table: EXISTS" -ForegroundColor Green
    } else {
        Write-Host "  Table $table: MISSING - Run migrations first!" -ForegroundColor Red
        exit 1
    }
}

# 3. 运行爬虫测试
Write-Host ""
Write-Host "Step 3: Running crawler for vendor: $Vendor" -ForegroundColor Yellow

# 构建并运行
cd $PSScriptRoot\..

Write-Host "  Building crawler..." -ForegroundColor Gray
go build -o crawler-test.exe .\cmd\crawler\main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "  Build failed!" -ForegroundColor Red
    exit 1
}

Write-Host "  Running crawler (once mode)..." -ForegroundColor Gray
.\crawler-test.exe -once -config config.yaml

if ($LASTEXITCODE -eq 0) {
    Write-Host "  Crawler execution: SUCCESS" -ForegroundColor Green
} else {
    Write-Host "  Crawler execution: FAILED" -ForegroundColor Red
}

# 4. 检查抓取结果
Write-Host ""
Write-Host "Step 4: Checking results..." -ForegroundColor Yellow

$snapshotCount = psql $DatabaseURL -c "SELECT COUNT(*) FROM vendor_price_snapshots WHERE vendor = '$Vendor' AND snapshot_date = CURRENT_DATE;" -t 2>$null
$detailCount = psql $DatabaseURL -c "SELECT COUNT(*) FROM vendor_price_details WHERE vendor = '$Vendor' AND snapshot_date = CURRENT_DATE;" -t 2>$null

Write-Host "  Snapshots today: $snapshotCount" -ForegroundColor Gray
Write-Host "  Details today: $detailCount" -ForegroundColor Gray

# 5. 显示样本数据
Write-Host ""
Write-Host "Step 5: Sample data" -ForegroundColor Yellow
psql $DatabaseURL -c "SELECT model_code, input_usd_per_million, output_usd_per_million, change_type FROM vendor_price_details WHERE vendor = '$Vendor' AND snapshot_date = CURRENT_DATE LIMIT 5;"

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test completed!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 清理
Remove-Item -Force crawler-test.exe -ErrorAction SilentlyContinue
