param (
    [int]$Users = 200,
    [int]$SpawnRate = 50,
    [string]$RunTime = "30s",
    [string]$HostUrl = "http://localhost:8080"
)

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   LOCUST STRESS TEST FOR GH-PIN-REPO-GO API             " -ForegroundColor Yellow
Write-Host "   Host:       $HostUrl                                  " -ForegroundColor White
Write-Host "   Users:      $Users concurrent simulated users         " -ForegroundColor White
Write-Host "   Spawn Rate: $SpawnRate users/sec                      " -ForegroundColor White
Write-Host "   Duration:   $RunTime                                  " -ForegroundColor White
Write-Host "==========================================================" -ForegroundColor Cyan

python -m locust -f test/locustfile.py --headless -u $Users -r $SpawnRate -t $RunTime --host $HostUrl --only-summary
